package stripe

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/common/payment"
	"github.com/stretchr/testify/require"
)

func TestAdapter_Configured(t *testing.T) {
	require.False(t, New(Config{}).Configured())
	require.True(t, New(Config{APIKey: "sk_test_x", WebhookSecret: "whsec_x"}).Configured())
}

func TestAdapter_Provider(t *testing.T) {
	require.Equal(t, payment.ProviderStripe, New(Config{}).Provider())
}

func TestAdapter_CreateCheckout_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{TradeNo: "T1", Money: 10})
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

func TestAdapter_CreateCheckout_InvalidRequest(t *testing.T) {
	a := New(Config{APIKey: "sk_test_x", WebhookSecret: "whsec_x"})
	cases := []struct {
		name string
		req  *payment.CheckoutRequest
	}{
		{"missing trade_no", &payment.CheckoutRequest{Money: 10}},
		{"zero money", &payment.CheckoutRequest{TradeNo: "T1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := a.CreateCheckout(context.Background(), c.req)
			require.ErrorIs(t, err, payment.ErrInvalidRequest)
		})
	}
}

func TestAdapter_VerifyWebhook_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{RawBody: []byte("{}")})
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

func TestAdapter_VerifyWebhook_EmptyBody(t *testing.T) {
	a := New(Config{APIKey: "sk_test_x", WebhookSecret: "whsec_x"})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{})
	require.ErrorIs(t, err, payment.ErrInvalidRequest)
}

func TestAdapter_VerifyWebhook_MissingSignature(t *testing.T) {
	a := New(Config{APIKey: "sk_test_x", WebhookSecret: "whsec_x"})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{RawBody: []byte("{}")})
	require.ErrorIs(t, err, payment.ErrInvalidSignature)
}

func TestAdapter_VerifyWebhook_BadSignature(t *testing.T) {
	a := New(Config{APIKey: "sk_test_x", WebhookSecret: "whsec_x"})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{
		RawBody:   []byte(`{"type":"checkout.session.completed"}`),
		Signature: "t=1,v1=deadbeef",
	})
	require.ErrorIs(t, err, payment.ErrInvalidSignature)
}

func TestAdapter_BuildWebhookResponse(t *testing.T) {
	a := New(Config{APIKey: "sk_test_x", WebhookSecret: "whsec_x"})
	r, err := a.BuildWebhookResponse(context.Background(), true, "")
	require.NoError(t, err)
	require.Equal(t, 200, r.StatusCode)

	r, err = a.BuildWebhookResponse(context.Background(), false, "boom")
	require.NoError(t, err)
	require.Equal(t, 500, r.StatusCode)
}

func TestOrDefault(t *testing.T) {
	require.Equal(t, "def", orDefault("", "def"))
	require.Equal(t, "v", orDefault("v", "def"))
}
