// Package epay implements the [payment.PaymentProvider] interface for
// Epay (易支付) aggregated payment gateways.
//
// Epay is a self-hosted payment aggregation protocol commonly used to
// proxy Alipay (支付宝), WeChat Pay (微信支付), and QQ Pay behind a single
// MD5-signed form-post endpoint. This adapter wraps the
// github.com/Calcium-Ion/go-epay/epay client.
//
// The adapter is stateless after construction and safe for concurrent use.
package epay

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/LingByte/ling-base/common/payment"
	"github.com/mitchellh/mapstructure"
)

// Config holds the Epay gateway credentials.
type Config struct {
	// PartnerID is the merchant id issued by the Epay gateway.
	PartnerID string

	// Key is the merchant signing key shared with the gateway.
	Key string

	// BaseURL is the gateway root URL (e.g. https://pay.example.com).
	BaseURL string

	// HTTPTimeout is the timeout for outbound API calls. Defaults to 15s.
	HTTPTimeout time.Duration
}

// Option mutates Config.
type Option func(*Config)

// WithHTTPTimeout sets the outbound HTTP timeout.
func WithHTTPTimeout(d time.Duration) Option {
	return func(c *Config) { c.HTTPTimeout = d }
}

// New constructs an Epay adapter. Returns nil (not an error) when the
// configuration is incomplete, so callers can treat a nil adapter as
// "disabled" without branching.
func New(cfg Config, opts ...Option) *Adapter {
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 15 * time.Second
	}
	return &Adapter{cfg: cfg}
}

// Adapter implements payment.PaymentProvider for Epay.
type Adapter struct {
	cfg Config
}

// Configured reports whether the adapter has the minimum credentials.
func (a *Adapter) Configured() bool {
	return a.cfg.PartnerID != "" && a.cfg.Key != "" && a.cfg.BaseURL != ""
}

// Provider returns the gateway identifier.
func (a *Adapter) Provider() payment.Provider { return payment.ProviderEpay }

// CreateCheckout builds the Epay form-post payment URL and signed
// parameters. The caller should redirect the payer's browser to
// result.CheckoutURL with result.Params submitted as a form.
func (a *Adapter) CreateCheckout(ctx context.Context, req *payment.CheckoutRequest) (*payment.CheckoutResult, error) {
	if !a.Configured() {
		return nil, payment.ErrNotConfigured
	}
	if req == nil || req.TradeNo == "" {
		return nil, fmt.Errorf("%w: missing trade_no", payment.ErrInvalidRequest)
	}
	if req.Money <= 0 {
		return nil, fmt.Errorf("%w: money must be > 0", payment.ErrInvalidRequest)
	}

	base, err := url.Parse(a.cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base url: %v", payment.ErrInvalidRequest, err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/submit.php"

	device := string(req.Device)
	if device == "" {
		device = string(payment.DevicePC)
	}

	currency := string(req.Currency)
	if currency == "" {
		currency = "CNY"
	}

	params := map[string]string{
		"pid":          a.cfg.PartnerID,
		"type":         req.ProductID, // Epay "type" = alipay/wxpay/qqpay
		"out_trade_no": req.TradeNo,
		"notify_url":   req.NotifyURL,
		"return_url":   req.ReturnURL,
		"name":         req.Name,
		"money":        fmt.Sprintf("%.2f", float64(req.Money)),
		"device":       device,
		"sign_type":    "MD5",
	}
	if params["type"] == "" {
		// Let the gateway auto-select when no specific channel is chosen.
		delete(params, "type")
	}
	if params["name"] == "" {
		params["name"] = fmt.Sprintf("Order %s", req.TradeNo)
	}

	signed := signParams(params, a.cfg.Key)
	params["sign"] = signed

	return &payment.CheckoutResult{
		Provider:    payment.ProviderEpay,
		CheckoutURL: base.String(),
		Params:      params,
		Raw:         params,
	}, nil
}

// VerifyWebhook verifies the Epay MD5 callback signature and parses the
// form parameters into a normalized WebhookEvent.
//
// Epay sends callbacks either as GET query strings or POST form bodies.
// The caller is responsible for collecting them into req.FormParams.
// req.RawBody may be empty for Epay.
func (a *Adapter) VerifyWebhook(ctx context.Context, req *payment.WebhookVerifyRequest) (*payment.WebhookEvent, error) {
	if !a.Configured() {
		return nil, payment.ErrNotConfigured
	}
	if req == nil || len(req.FormParams) == 0 {
		return nil, fmt.Errorf("%w: empty form params", payment.ErrInvalidRequest)
	}

	provided := strings.TrimSpace(req.FormParams["sign"])
	if provided == "" {
		return nil, fmt.Errorf("%w: missing sign", payment.ErrInvalidSignature)
	}

	expected := signParams(filterSigned(req.FormParams), a.cfg.Key)
	if !strings.EqualFold(provided, expected) {
		return nil, payment.ErrInvalidSignature
	}

	var res struct {
		Type           string `mapstructure:"type"`
		TradeNo        string `mapstructure:"trade_no"`
		ServiceTradeNo string `mapstructure:"out_trade_no"`
		Name           string `mapstructure:"name"`
		Money          string `mapstructure:"money"`
		TradeStatus    string `mapstructure:"trade_status"`
	}
	if err := mapstructure.Decode(req.FormParams, &res); err != nil {
		return nil, fmt.Errorf("%w: decode: %v", payment.ErrInvalidRequest, err)
	}

	event := &payment.WebhookEvent{
		Provider:       payment.ProviderEpay,
		TradeNo:        res.ServiceTradeNo,
		ProviderOrderID: res.TradeNo,
		OrderType:      res.Type,
		Raw:            req.FormParams,
		RawBody:        req.RawBody,
	}

	switch strings.ToUpper(res.TradeStatus) {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		event.EventType = payment.EventCheckoutCompleted
		event.Status = payment.StatusSuccess
		if res.Money != "" {
			var m float64
			_, _ = fmt.Sscanf(res.Money, "%f", &m)
			event.MoneyPaid = payment.Money(m)
		}
	default:
		event.EventType = payment.EventUnknown
		event.Status = payment.StatusPending
	}

	return event, nil
}

// BuildWebhookResponse returns the Epay ack body. Epay expects the literal
// string "success" on successful processing and "fail" on failure.
func (a *Adapter) BuildWebhookResponse(ctx context.Context, success bool, msg string) (*payment.WebhookResponse, error) {
	body := "fail"
	if success {
		body = "success"
	}
	return &payment.WebhookResponse{
		StatusCode: 200,
		Body:       []byte(body),
		ContentType: "text/plain; charset=utf-8",
	}, nil
}

// QueryOrder is not supported by the Epay protocol; callers should rely on
// webhooks. Returns an error wrapping payment.ErrProviderError.
func (a *Adapter) QueryOrder(ctx context.Context, req *payment.OrderQuery) (*payment.OrderQueryResult, error) {
	return nil, fmt.Errorf("%w: epay does not support order query api", payment.ErrProviderError)
}

// signParams computes the Epay MD5 signature over the sorted, non-empty
// key=value pairs (excluding sign and sign_type), appended with the
// merchant key.
func signParams(params map[string]string, key string) string {
	filtered := filterSigned(params)
	keys := make([]string, 0, len(filtered))
	for k := range filtered {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(filtered[k])
	}
	b.WriteString(key)
	sum := md5.Sum([]byte(b.String()))
	return fmt.Sprintf("%x", sum)
}

// filterSigned removes sign, sign_type, and empty-valued entries.
func filterSigned(params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	for k, v := range params {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		out[k] = v
	}
	return out
}

// ErrMissingConfig is returned when required configuration is absent.
// Use Configured() to check ahead of time.
var ErrMissingConfig = errors.New("epay: missing configuration")
