// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package geocode provides forward and reverse geocoding utilities
// using free, no-API-key-required services.
//
// Providers:
//
//   - Nominatim (OpenStreetMap): free, no key, 1 req/sec — set a Limiter in production
//   - BigDataCloud: free reverse geocoding, no key
//
// # Quick start
//
//	ctx := context.Background()
//	client := geocode.NewClient(geocode.WithLimiter(geocode.NewIntervalLimiter(time.Second)))
//
//	addr, err := client.Reverse(ctx, 39.9042, 116.4074)
//	result, err := client.Forward(ctx, "Eiffel Tower, Paris")
package geocode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Provider identifies the geocoding service.
type Provider string

const (
	// ProviderNominatim uses OpenStreetMap Nominatim API.
	ProviderNominatim Provider = "nominatim"

	// ProviderBigDataCloud uses BigDataCloud reverse geocoding API.
	ProviderBigDataCloud Provider = "bigdatacloud"
)

const (
	defaultNominatimBaseURL    = "https://nominatim.openstreetmap.org"
	defaultBigDataCloudBaseURL = "https://api.bigdatacloud.net/data/reverse-geocode-client"
	defaultUserAgent           = "ling-base-geocode/1.0 (https://github.com/LingByte/ling-base)"
	defaultTimeout             = 10 * time.Second
	defaultNominatimZoom       = 18
	maxResponseBytes           = 1 << 20 // 1 MiB
	coordPrecision             = 7
)

var (
	// ErrNotFound is returned when a provider finds no matching place.
	ErrNotFound = errors.New("geocode: not found")

	// ErrRateLimited is returned when a provider responds with HTTP 429.
	ErrRateLimited = errors.New("geocode: rate limited")

	// ErrEmptyQuery is returned for empty forward queries.
	ErrEmptyQuery = errors.New("geocode: empty query")
)

// Limiter serializes / spaces outbound requests (e.g. Nominatim 1 req/sec).
type Limiter interface {
	Wait(ctx context.Context) error
}

// IntervalLimiter enforces a minimum interval between Wait calls.
type IntervalLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
}

// NewIntervalLimiter returns a limiter that waits at least interval between calls.
func NewIntervalLimiter(interval time.Duration) *IntervalLimiter {
	if interval <= 0 {
		interval = time.Second
	}
	return &IntervalLimiter{interval: interval}
}

// Wait blocks until the next request is allowed or ctx is cancelled.
func (l *IntervalLimiter) Wait(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.last.IsZero() {
		wait := l.interval - time.Since(l.last)
		if wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	l.last = time.Now()
	return nil
}

// GeocodeResult holds the result of a forward geocoding lookup.
type GeocodeResult struct {
	Lat           float64  // Latitude
	Lon           float64  // Longitude
	DisplayName   string   // Full human-readable address
	Country       string   // Country name
	CountryCode   string   // ISO 3166-1 alpha-2 country code
	State         string   // State / province / region
	StateDistrict string   // State district
	County        string   // County
	City          string   // City / town / village
	Postcode      string   // Postal code
	Street        string   // Street name
	HouseNumber   string   // House number
	Suburb        string   // Suburb / neighborhood
	Type          string   // OSM type (e.g. "city", "road", "building")
	Importance    float64  // Search rank importance (0-1)
	Provider      Provider // Which provider returned this result
}

// IsZero reports whether r is nil or has no useful coordinates / display name.
func (r *GeocodeResult) IsZero() bool {
	return r == nil || (r.Lat == 0 && r.Lon == 0 && r.DisplayName == "")
}

// ReverseResult holds the result of a reverse geocoding lookup.
type ReverseResult struct {
	Lat          float64        // Latitude
	Lon          float64        // Longitude
	DisplayName  string         // Full human-readable address
	Country      string         // Country name
	CountryCode  string         // ISO 3166-1 alpha-2
	Locality     string         // City / locality
	PrincipalSub string         // State / province / principal subdivision
	SubLocality  string         // Sub-locality / district / neighbourhood
	PostCode     string         // Postal code
	LocalityInfo map[string]any // Extra locality metadata from the provider
	Provider     Provider       // Which provider returned this result
}

// IsZero reports whether r is nil or has no useful display / locality data.
func (r *ReverseResult) IsZero() bool {
	return r == nil || (r.DisplayName == "" && r.Locality == "" && r.Country == "")
}

// GeocodeQuery is a structured address query for forward geocoding.
type GeocodeQuery struct {
	Query    string // Free-form query string (e.g. "Eiffel Tower, Paris")
	City     string // City name
	County   string // County name
	State    string // State / province
	Country  string // Country name
	Postcode string // Postal code
	Street   string // Street name
	// ViewBox optionally restricts Nominatim search (left,top,right,bottom).
	ViewBox string
}

// Client is a reusable geocoding client.
type Client struct {
	provider         Provider
	userAgent        string
	language         string
	timeout          time.Duration
	http             *http.Client
	limiter          Limiter
	nominatimZoom    int
	fallback         bool
	nominatimBaseURL string
	bdcBaseURL       string
}

// Option configures a Client.
type Option func(*Client)

// WithProvider sets the primary geocoding provider.
func WithProvider(p Provider) Option {
	return func(c *Client) { c.provider = p }
}

// WithUserAgent sets the User-Agent header (required by Nominatim).
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

// WithLanguage sets Accept-Language / locality language (e.g. "zh", "en").
func WithLanguage(lang string) Option {
	return func(c *Client) { c.language = lang }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// WithLimiter sets a request limiter (recommended for Nominatim).
func WithLimiter(l Limiter) Option {
	return func(c *Client) { c.limiter = l }
}

// WithNominatimZoom sets Nominatim reverse zoom (0–18, higher = more detail).
func WithNominatimZoom(zoom int) Option {
	return func(c *Client) { c.nominatimZoom = zoom }
}

// WithFallback enables trying the other reverse provider when the primary fails.
func WithFallback(enabled bool) Option {
	return func(c *Client) { c.fallback = enabled }
}

// WithNominatimBaseURL overrides the Nominatim base URL (tests).
func WithNominatimBaseURL(u string) Option {
	return func(c *Client) { c.nominatimBaseURL = strings.TrimRight(u, "/") }
}

// WithBigDataCloudBaseURL overrides the BigDataCloud endpoint (tests).
func WithBigDataCloudBaseURL(u string) Option {
	return func(c *Client) { c.bdcBaseURL = u }
}

// NewClient creates a geocoding client with the given options.
func NewClient(opts ...Option) *Client {
	c := &Client{
		provider:         ProviderNominatim,
		userAgent:        defaultUserAgent,
		timeout:          defaultTimeout,
		nominatimZoom:    defaultNominatimZoom,
		nominatimBaseURL: defaultNominatimBaseURL,
		bdcBaseURL:       defaultBigDataCloudBaseURL,
		http:             &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: c.timeout}
	} else if c.timeout > 0 {
		c.http.Timeout = c.timeout
	}
	if c.nominatimZoom <= 0 {
		c.nominatimZoom = defaultNominatimZoom
	}
	return c
}

// defaultClient is reused by package-level helpers so callers share HTTP settings.
var defaultClient = NewClient(WithLimiter(NewIntervalLimiter(time.Second)))

// ─── Package-level helpers (reuse defaultClient) ───────────────

// Forward geocodes a free-form address using the package default client.
func Forward(address string) (*GeocodeResult, error) {
	return defaultClient.Forward(context.Background(), address)
}

// ForwardStructured geocodes a structured address using the package default client.
func ForwardStructured(q *GeocodeQuery) (*GeocodeResult, error) {
	return defaultClient.ForwardStructured(context.Background(), q)
}

// Reverse reverse-geocodes coordinates using the package default client.
func Reverse(lat, lon float64) (*ReverseResult, error) {
	return defaultClient.Reverse(context.Background(), lat, lon)
}

// ─── Forward Geocoding ─────────────────────────────────────────

// Forward geocodes a free-form address string to coordinates (Nominatim).
func (c *Client) Forward(ctx context.Context, address string) (*GeocodeResult, error) {
	if strings.TrimSpace(address) == "" {
		return nil, ErrEmptyQuery
	}
	params := url.Values{}
	params.Set("q", address)
	params.Set("format", "jsonv2")
	params.Set("addressdetails", "1")
	params.Set("limit", "1")
	return c.doNominatimSearch(ctx, params)
}

// ForwardStructured geocodes a structured address to coordinates (Nominatim).
func (c *Client) ForwardStructured(ctx context.Context, q *GeocodeQuery) (*GeocodeResult, error) {
	if q == nil {
		return nil, fmt.Errorf("%w: query is nil", ErrEmptyQuery)
	}

	params := url.Values{}
	if q.Query != "" {
		params.Set("q", q.Query)
	} else {
		if q.Street != "" {
			params.Set("street", q.Street)
		}
		if q.City != "" {
			params.Set("city", q.City)
		}
		if q.County != "" {
			params.Set("county", q.County)
		}
		if q.State != "" {
			params.Set("state", q.State)
		}
		if q.Country != "" {
			params.Set("country", q.Country)
		}
		if q.Postcode != "" {
			params.Set("postalcode", q.Postcode)
		}
	}
	if q.ViewBox != "" {
		params.Set("viewbox", q.ViewBox)
		params.Set("bounded", "1")
	}
	if len(params) == 0 {
		return nil, fmt.Errorf("%w: at least one address field is required", ErrEmptyQuery)
	}

	params.Set("format", "jsonv2")
	params.Set("addressdetails", "1")
	params.Set("limit", "1")
	return c.doNominatimSearch(ctx, params)
}

func (c *Client) doNominatimSearch(ctx context.Context, params url.Values) (*GeocodeResult, error) {
	if err := c.wait(ctx); err != nil {
		return nil, err
	}

	searchURL := fmt.Sprintf("%s/search?%s", c.nominatimBaseURL, params.Encode())
	body, err := c.doJSON(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	var results []nominatimSearchResponse
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("geocode: parse nominatim search: %w", err)
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return parseNominatimSearchResult(&results[0])
}

// ─── Reverse Geocoding ─────────────────────────────────────────

// Reverse reverse-geocodes coordinates using the configured provider.
func (c *Client) Reverse(ctx context.Context, lat, lon float64) (*ReverseResult, error) {
	primary, secondary := c.reversePair()
	res, err := primary(ctx, lat, lon)
	if err == nil || !c.fallback || secondary == nil {
		return res, err
	}
	return secondary(ctx, lat, lon)
}

type reverseFn func(ctx context.Context, lat, lon float64) (*ReverseResult, error)

func (c *Client) reversePair() (reverseFn, reverseFn) {
	switch c.provider {
	case ProviderBigDataCloud:
		return c.doReverseBigDataCloud, c.doReverseNominatim
	default:
		return c.doReverseNominatim, c.doReverseBigDataCloud
	}
}

// ReverseNominatim reverse-geocodes with Nominatim.
func (c *Client) ReverseNominatim(ctx context.Context, lat, lon float64) (*ReverseResult, error) {
	return c.doReverseNominatim(ctx, lat, lon)
}

// ReverseBigDataCloud reverse-geocodes with BigDataCloud.
func (c *Client) ReverseBigDataCloud(ctx context.Context, lat, lon float64) (*ReverseResult, error) {
	return c.doReverseBigDataCloud(ctx, lat, lon)
}

func (c *Client) doReverseNominatim(ctx context.Context, lat, lon float64) (*ReverseResult, error) {
	if err := c.wait(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("lat", formatCoord(lat))
	params.Set("lon", formatCoord(lon))
	params.Set("format", "jsonv2")
	params.Set("addressdetails", "1")
	params.Set("zoom", strconv.Itoa(c.nominatimZoom))

	reverseURL := fmt.Sprintf("%s/reverse?%s", c.nominatimBaseURL, params.Encode())
	body, err := c.doJSON(ctx, reverseURL)
	if err != nil {
		return nil, err
	}

	var nr nominatimReverseResponse
	if err := json.Unmarshal(body, &nr); err != nil {
		return nil, fmt.Errorf("geocode: parse nominatim reverse: %w", err)
	}
	if nr.Error != nil {
		msg := nr.Error.Message
		if strings.Contains(strings.ToLower(msg), "unable to geocode") {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("geocode: nominatim: %s", msg)
	}
	if nr.DisplayName == "" && nr.Address.Country == "" {
		return nil, ErrNotFound
	}

	return &ReverseResult{
		Lat:          lat,
		Lon:          lon,
		DisplayName:  nr.DisplayName,
		Country:      nr.Address.Country,
		CountryCode:  strings.ToUpper(nr.Address.CountryCode),
		PrincipalSub: nr.Address.State,
		Locality:     firstNonEmpty(nr.Address.City, nr.Address.Town, nr.Address.Village),
		PostCode:     nr.Address.Postcode,
		SubLocality:  firstNonEmpty(nr.Address.Suburb, nr.Address.Neighbourhood),
		LocalityInfo: nr.Address.Extra,
		Provider:     ProviderNominatim,
	}, nil
}

func (c *Client) doReverseBigDataCloud(ctx context.Context, lat, lon float64) (*ReverseResult, error) {
	if err := c.wait(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("latitude", formatCoord(lat))
	params.Set("longitude", formatCoord(lon))
	lang := c.language
	if lang == "" {
		lang = "en"
	}
	params.Set("localityLanguage", lang)

	requestURL := fmt.Sprintf("%s?%s", c.bdcBaseURL, params.Encode())
	body, err := c.doJSON(ctx, requestURL)
	if err != nil {
		return nil, err
	}

	var bdc bigDataCloudResponse
	if err := json.Unmarshal(body, &bdc); err != nil {
		return nil, fmt.Errorf("geocode: parse bigdatacloud reverse: %w", err)
	}

	locality := firstNonEmpty(bdc.City, bdc.Locality)
	subLocality := ""
	if bdc.Locality != "" && bdc.Locality != locality {
		subLocality = bdc.Locality
	}

	display := firstNonEmpty(
		joinNonEmpty(", ", bdc.Locality, bdc.City, bdc.PrincipalSubdivision, bdc.CountryName),
		bdc.City,
		bdc.Locality,
		bdc.CountryName,
	)
	if display == "" {
		return nil, ErrNotFound
	}

	return &ReverseResult{
		Lat:          lat,
		Lon:          lon,
		DisplayName:  display,
		Country:      bdc.CountryName,
		CountryCode:  strings.ToUpper(bdc.CountryCode),
		PrincipalSub: bdc.PrincipalSubdivision,
		Locality:     locality,
		SubLocality:  subLocality,
		PostCode:     bdc.Postcode,
		LocalityInfo: bdc.LocalityInfo,
		Provider:     ProviderBigDataCloud,
	}, nil
}

// ─── HTTP helpers ──────────────────────────────────────────────

func (c *Client) wait(ctx context.Context) error {
	if c.limiter == nil {
		return nil
	}
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("geocode: limiter: %w", err)
	}
	return nil
}

func (c *Client) doJSON(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("geocode: create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	if c.language != "" {
		req.Header.Set("Accept-Language", c.language)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("geocode: request failed: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		return nil, ErrRateLimited
	case resp.StatusCode != http.StatusOK:
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		return nil, fmt.Errorf("geocode: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("geocode: read response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("geocode: response exceeds %d bytes", maxResponseBytes)
	}
	return body, nil
}

// ─── Distance helpers ──────────────────────────────────────────

// HaversineDistance calculates the great-circle distance in kilometers.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	lat1Rad := lat1 * (math.Pi / 180)
	lat2Rad := lat2 * (math.Pi / 180)
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// HaversineDistanceMeters returns the distance in meters.
func HaversineDistanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	return HaversineDistance(lat1, lon1, lat2, lon2) * 1000
}

// IsInRadius reports whether two coordinates are within radiusKm.
func IsInRadius(lat1, lon1, lat2, lon2, radiusKm float64) bool {
	return HaversineDistance(lat1, lon1, lat2, lon2) <= radiusKm
}

// ─── Internal types ────────────────────────────────────────────

type nominatimSearchResponse struct {
	PlaceID     int64            `json:"place_id"`
	Lat         string           `json:"lat"`
	Lon         string           `json:"lon"`
	DisplayName string           `json:"display_name"`
	Type        string           `json:"type"`
	Importance  float64          `json:"importance"`
	Address     nominatimAddress `json:"address"`
}

type nominatimReverseResponse struct {
	PlaceID     int64            `json:"place_id"`
	Lat         string           `json:"lat"`
	Lon         string           `json:"lon"`
	DisplayName string           `json:"display_name"`
	Address     nominatimAddress `json:"address"`
	Error       *nominatimError  `json:"error,omitempty"`
}

type nominatimError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// UnmarshalJSON accepts both {"error":"msg"} and {"error":{"code":..,"message":..}}.
func (e *nominatimError) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var msg string
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		e.Message = msg
		return nil
	}
	type alias nominatimError
	var tmp alias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*e = nominatimError(tmp)
	return nil
}

type nominatimAddress struct {
	HouseNumber   string         `json:"house_number"`
	Road          string         `json:"road"`
	Suburb        string         `json:"suburb"`
	Neighbourhood string         `json:"neighbourhood"`
	City          string         `json:"city"`
	Town          string         `json:"town"`
	Village       string         `json:"village"`
	County        string         `json:"county"`
	State         string         `json:"state"`
	Postcode      string         `json:"postcode"`
	Country       string         `json:"country"`
	CountryCode   string         `json:"country_code"`
	Extra         map[string]any `json:"-"`
}

func (a *nominatimAddress) UnmarshalJSON(data []byte) error {
	type alias nominatimAddress
	var tmp alias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = nominatimAddress(tmp)

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	known := map[string]bool{
		"house_number": true, "road": true, "suburb": true, "neighbourhood": true,
		"city": true, "town": true, "village": true,
		"county": true, "state": true, "postcode": true,
		"country": true, "country_code": true,
	}
	a.Extra = make(map[string]any)
	for k, v := range raw {
		if !known[k] {
			a.Extra[k] = v
		}
	}
	return nil
}

type bigDataCloudResponse struct {
	CountryCode          string         `json:"countryCode"`
	CountryName          string         `json:"countryName"`
	PrincipalSubdivision string         `json:"principalSubdivision"`
	Locality             string         `json:"locality"`
	City                 string         `json:"city"`
	Postcode             string         `json:"postcode"`
	LocalityInfo         map[string]any `json:"localityInfo"`
}

func parseNominatimSearchResult(r *nominatimSearchResponse) (*GeocodeResult, error) {
	lat, err := strconv.ParseFloat(r.Lat, 64)
	if err != nil {
		return nil, fmt.Errorf("geocode: parse lat %q: %w", r.Lat, err)
	}
	lon, err := strconv.ParseFloat(r.Lon, 64)
	if err != nil {
		return nil, fmt.Errorf("geocode: parse lon %q: %w", r.Lon, err)
	}

	return &GeocodeResult{
		Lat:         lat,
		Lon:         lon,
		DisplayName: r.DisplayName,
		Country:     r.Address.Country,
		CountryCode: strings.ToUpper(r.Address.CountryCode),
		State:       r.Address.State,
		County:      r.Address.County,
		City:        firstNonEmpty(r.Address.City, r.Address.Town, r.Address.Village),
		Postcode:    r.Address.Postcode,
		Street:      r.Address.Road,
		HouseNumber: r.Address.HouseNumber,
		Suburb:      firstNonEmpty(r.Address.Suburb, r.Address.Neighbourhood),
		Type:        r.Type,
		Importance:  r.Importance,
		Provider:    ProviderNominatim,
	}, nil
}

func formatCoord(v float64) string {
	return strconv.FormatFloat(v, 'f', coordPrecision, 64)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func joinNonEmpty(sep string, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(out) > 0 && out[len(out)-1] == p {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, sep)
}
