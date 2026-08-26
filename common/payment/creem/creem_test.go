package creem

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/LingByte/ling-base/common/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapter_Configured(t *testing.T) {
	a := New(Config{})
	require.False(t, a.Configured())

	a = New(Config{APIKey: "k", WebhookSecret: "s"})
	require.True(t, a.Configured())
}

func TestAdapter_Provider(t *testing.T) {
	a := New(Config{APIKey: "k", WebhookSecret: "s"})
	require.Equal(t, payment.ProviderCreem, a.Provider())
}

func TestAdapter_apiBase_TestMode(t *testing.T) {
	require.Equal(t, "https://test-api.creem.io", New(Config{TestMode: true}).apiBase())
	require.Equal(t, "https://api.creem.io", New(Config{}).apiBase())
}

func TestAdapter_VerifyWebhook_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{RawBody: []byte("{}")})
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

func TestAdapter_VerifyWebhook_EmptyBody(t *testing.T) {
	a := New(Config{APIKey: "k", WebhookSecret: "s"})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{})
	require.ErrorIs(t, err, payment.ErrInvalidRequest)
}

func TestAdapter_VerifyWebhook_MissingSignature(t *testing.T) {
	a := New(Config{APIKey: "k", WebhookSecret: "s"})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{RawBody: []byte("{}")})
	require.ErrorIs(t, err, payment.ErrInvalidSignature)
}

func TestAdapter_VerifyWebhook_BadSignature(t *testing.T) {
	a := New(Config{APIKey: "k", WebhookSecret: "s"})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{
		RawBody:   []byte(`{"eventType":"checkout.completed"}`),
		Signature: "deadbeef",
	})
	require.ErrorIs(t, err, payment.ErrInvalidSignature)
}

func TestAdapter_VerifyWebhook_OK(t *testing.T) {
	secret := "whsec_test"
	a := New(Config{APIKey: "k", WebhookSecret: secret})

	payload := buildCreemWebhookPayload(t, "checkout.completed", "paid", "onetime", "REQ-1", "ORD-1", 1299, "USD", "test")
	sig := computeSignature(string(payload), secret)

	evt, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{
		RawBody:   payload,
		Signature: sig,
	})
	require.NoError(t, err)
	require.Equal(t, payment.ProviderCreem, evt.Provider)
	require.Equal(t, payment.EventCheckoutCompleted, evt.EventType)
	require.Equal(t, payment.StatusSuccess, evt.Status)
	require.Equal(t, "REQ-1", evt.TradeNo)
	require.Equal(t, "ORD-1", evt.ProviderOrderID)
	require.Equal(t, "onetime", evt.OrderType)
	require.Equal(t, "test", evt.Mode)
	require.Equal(t, payment.Currency("USD"), evt.Currency)
	require.Equal(t, payment.Money(12.99), evt.MoneyPaid)
}

func TestAdapter_VerifyWebhook_NonPaidStatus(t *testing.T) {
	secret := "whsec_test"
	a := New(Config{APIKey: "k", WebhookSecret: secret})
	payload := buildCreemWebhookPayload(t, "checkout.completed", "unpaid", "onetime", "REQ-1", "ORD-1", 0, "USD", "test")
	sig := computeSignature(string(payload), secret)

	evt, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{RawBody: payload, Signature: sig})
	require.NoError(t, err)
	assert.Equal(t, payment.EventCheckoutCompleted, evt.EventType)
	assert.Equal(t, payment.StatusPending, evt.Status)
}

func TestAdapter_VerifyWebhook_UnknownEventType(t *testing.T) {
	secret := "whsec_test"
	a := New(Config{APIKey: "k", WebhookSecret: secret})
	payload := buildCreemWebhookPayload(t, "product.updated", "paid", "onetime", "REQ-1", "ORD-1", 100, "USD", "test")
	sig := computeSignature(string(payload), secret)

	evt, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{RawBody: payload, Signature: sig})
	require.NoError(t, err)
	assert.Equal(t, payment.EventUnknown, evt.EventType)
	assert.Equal(t, payment.StatusPending, evt.Status)
}

func TestAdapter_BuildWebhookResponse(t *testing.T) {
	a := New(Config{APIKey: "k", WebhookSecret: "s"})
	r, err := a.BuildWebhookResponse(context.Background(), true, "")
	require.NoError(t, err)
	require.Equal(t, 200, r.StatusCode)

	r, err = a.BuildWebhookResponse(context.Background(), false, "boom")
	require.NoError(t, err)
	require.Equal(t, 500, r.StatusCode)
}

func TestAdapter_QueryOrder_Unsupported(t *testing.T) {
	a := New(Config{APIKey: "k", WebhookSecret: "s"})
	_, err := a.QueryOrder(context.Background(), &payment.OrderQuery{TradeNo: "T1"})
	require.ErrorIs(t, err, payment.ErrProviderError)
}

func TestVerifySignature_EmptySecret(t *testing.T) {
	require.False(t, verifySignature("payload", "sig", ""))
}

// buildCreemWebhookPayload constructs a minimal Creem webhook JSON payload
// for testing.
func buildCreemWebhookPayload(t *testing.T, eventType, orderStatus, orderType, requestID, orderID string, amountPaid int, currency, mode string) []byte {
	t.Helper()
	payload := map[string]any{
		"id":        "evt_1",
		"eventType": eventType,
		"created_at": 1700000000,
		"object": map[string]any{
			"id":         "obj_1",
			"object":     "checkout",
			"request_id": requestID,
			"order": map[string]any{
				"id":          orderID,
				"status":      orderStatus,
				"type":        orderType,
				"amount_paid": amountPaid,
				"currency":    currency,
				"mode":        mode,
			},
			"product": map[string]any{
				"id":   "prod_1",
				"name": "Test Product",
			},
			"customer": map[string]any{
				"id":    "cus_1",
				"email": "buyer@example.com",
				"name":  "Buyer One",
			},
			"status": orderStatus,
			"mode":   mode,
		},
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return b
}
