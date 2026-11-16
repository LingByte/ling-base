// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package geocode

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHaversineDistance_BeijingToShanghai(t *testing.T) {
	d := HaversineDistance(39.9042, 116.4074, 31.2304, 121.4737)
	assert.InDelta(t, 1067, d, 10)
}

func TestHaversineDistance_SamePoint(t *testing.T) {
	assert.InDelta(t, 0, HaversineDistance(40.0, 116.0, 40.0, 116.0), 0.001)
}

func TestHaversineDistanceMeters(t *testing.T) {
	d := HaversineDistanceMeters(39.9042, 116.4074, 39.9042, 116.4174)
	assert.True(t, d > 500 && d < 1500)
}

func TestIsInRadius(t *testing.T) {
	assert.True(t, IsInRadius(39.9042, 116.4074, 39.9042, 116.4074, 1.0))
	assert.False(t, IsInRadius(39.9042, 116.4074, 31.2304, 121.4737, 500.0))
}

func TestFormatCoord(t *testing.T) {
	assert.Equal(t, "39.9042000", formatCoord(39.9042))
	assert.NotContains(t, formatCoord(1e-7), "e")
}

func TestClient_Forward_HTTP(t *testing.T) {
	mockResp := `[{
		"place_id": 12345,
		"lat": "48.8584",
		"lon": "2.2945",
		"display_name": "Eiffel Tower, Paris, France",
		"type": "tourism",
		"importance": 0.9,
		"address": {
			"road": "Avenue Anatole France",
			"city": "Paris",
			"country": "France",
			"country_code": "fr",
			"postcode": "75007",
			"neighbourhood": "7th"
		}
	}]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search", r.URL.Path)
		assert.Equal(t, "zh", r.Header.Get("Accept-Language"))
		assert.NotEmpty(t, r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockResp))
	}))
	defer server.Close()

	client := NewClient(
		WithNominatimBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithLanguage("zh"),
	)
	got, err := client.Forward(context.Background(), "Eiffel Tower")
	require.NoError(t, err)
	assert.Equal(t, 48.8584, got.Lat)
	assert.Equal(t, 2.2945, got.Lon)
	assert.Equal(t, "Paris", got.City)
	assert.Equal(t, "FR", got.CountryCode)
	assert.Equal(t, "7th", got.Suburb)
	assert.Equal(t, ProviderNominatim, got.Provider)
}

func TestClient_Forward_EmptyAddress(t *testing.T) {
	_, err := NewClient().Forward(context.Background(), "")
	require.ErrorIs(t, err, ErrEmptyQuery)
}

func TestClient_ForwardStructured_EmptyQuery(t *testing.T) {
	_, err := NewClient().ForwardStructured(context.Background(), &GeocodeQuery{})
	require.ErrorIs(t, err, ErrEmptyQuery)
}

func TestClient_Forward_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewClient(WithNominatimBaseURL(server.URL), WithHTTPClient(server.Client()))
	_, err := client.Forward(context.Background(), "nowhere")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestClient_Forward_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient(WithNominatimBaseURL(server.URL), WithHTTPClient(server.Client()))
	_, err := client.Forward(context.Background(), "x")
	require.ErrorIs(t, err, ErrRateLimited)
}

func TestClient_ReverseNominatim_TownFallback(t *testing.T) {
	mockResp := `{
		"place_id": 1,
		"lat": "40.0",
		"lon": "116.0",
		"display_name": "SmallTown, Country",
		"address": {
			"town": "SmallTown",
			"state": "State",
			"country": "Country",
			"country_code": "co",
			"neighbourhood": "Old Quarter",
			"borough": "North"
		}
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/reverse", r.URL.Path)
		assert.Equal(t, "40.0000000", r.URL.Query().Get("lat"))
		_, _ = w.Write([]byte(mockResp))
	}))
	defer server.Close()

	client := NewClient(WithNominatimBaseURL(server.URL), WithHTTPClient(server.Client()))
	got, err := client.ReverseNominatim(context.Background(), 40, 116)
	require.NoError(t, err)
	assert.Equal(t, "SmallTown", got.Locality)
	assert.Equal(t, "Old Quarter", got.SubLocality)
	assert.Equal(t, "North", got.LocalityInfo["borough"])
}

func TestClient_ReverseBigDataCloud_Mapping(t *testing.T) {
	mockResp := `{
		"countryCode": "CN",
		"countryName": "China",
		"principalSubdivision": "Beijing",
		"locality": "Dongcheng",
		"city": "Beijing",
		"postcode": "100010",
		"localityInfo": {"administrative": []}
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "zh", r.URL.Query().Get("localityLanguage"))
		_, _ = w.Write([]byte(mockResp))
	}))
	defer server.Close()

	client := NewClient(
		WithBigDataCloudBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithLanguage("zh"),
	)
	got, err := client.ReverseBigDataCloud(context.Background(), 39.9042, 116.4074)
	require.NoError(t, err)
	assert.Equal(t, "Beijing", got.Locality)
	assert.Equal(t, "Dongcheng", got.SubLocality)
	assert.Contains(t, got.DisplayName, "Dongcheng")
	assert.Contains(t, got.DisplayName, "China")
	assert.NotEqual(t, got.Locality, got.SubLocality)
	assert.Equal(t, ProviderBigDataCloud, got.Provider)
}

func TestClient_Reverse_Fallback(t *testing.T) {
	var nominatimHits atomic.Int32
	nom := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nominatimHits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer nom.Close()

	bdc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"countryCode":"CN","countryName":"China","city":"Beijing","locality":"Beijing"
		}`))
	}))
	defer bdc.Close()

	client := NewClient(
		WithProvider(ProviderNominatim),
		WithFallback(true),
		WithNominatimBaseURL(nom.URL),
		WithBigDataCloudBaseURL(bdc.URL),
		WithHTTPClient(http.DefaultClient),
	)
	got, err := client.Reverse(context.Background(), 39.9, 116.4)
	require.NoError(t, err)
	assert.Equal(t, ProviderBigDataCloud, got.Provider)
	assert.GreaterOrEqual(t, nominatimHits.Load(), int32(1))
}

func TestIntervalLimiter(t *testing.T) {
	l := NewIntervalLimiter(30 * time.Millisecond)
	ctx := context.Background()
	start := time.Now()
	require.NoError(t, l.Wait(ctx))
	require.NoError(t, l.Wait(ctx))
	assert.GreaterOrEqual(t, time.Since(start), 25*time.Millisecond)
}

func TestNominatimAddress_ExtraFields(t *testing.T) {
	raw := `{
		"road": "Main St",
		"city": "Test City",
		"country": "Test Country",
		"country_code": "tc",
		"custom_field": "custom_value",
		"neighbourhood": "Downtown"
	}`
	var addr nominatimAddress
	require.NoError(t, json.Unmarshal([]byte(raw), &addr))
	assert.Equal(t, "Downtown", addr.Neighbourhood)
	assert.Equal(t, "custom_value", addr.Extra["custom_field"])
	_, hasRoad := addr.Extra["road"]
	assert.False(t, hasRoad)
}

func TestParseNominatimSearchResult_Fallbacks(t *testing.T) {
	r := &nominatimSearchResponse{
		Lat: "40.0", Lon: "116.0", DisplayName: "Test",
		Address: nominatimAddress{Village: "SmallVillage", CountryCode: "co"},
	}
	got, err := parseNominatimSearchResult(r)
	require.NoError(t, err)
	assert.Equal(t, "SmallVillage", got.City)
}

func TestIsZero(t *testing.T) {
	assert.True(t, (*GeocodeResult)(nil).IsZero())
	assert.True(t, (&GeocodeResult{}).IsZero())
	assert.False(t, (&GeocodeResult{DisplayName: "x"}).IsZero())
	assert.True(t, (*ReverseResult)(nil).IsZero())
	assert.False(t, (&ReverseResult{Locality: "Beijing"}).IsZero())
}

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient()
	assert.Equal(t, ProviderNominatim, c.provider)
	assert.Equal(t, defaultUserAgent, c.userAgent)
	assert.Equal(t, defaultTimeout, c.timeout)
	assert.NotNil(t, c.http)
}

func TestNominatimError_StringForm(t *testing.T) {
	var nr nominatimReverseResponse
	err := json.Unmarshal([]byte(`{"error":"Unable to geocode"}`), &nr)
	require.NoError(t, err)
	require.NotNil(t, nr.Error)
	assert.Equal(t, "Unable to geocode", nr.Error.Message)
}

func TestErrorsAreDistinct(t *testing.T) {
	assert.False(t, errors.Is(ErrNotFound, ErrRateLimited))
}
