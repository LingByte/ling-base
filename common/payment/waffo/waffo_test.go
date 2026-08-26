package waffo

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/common/payment"
	"github.com/stretchr/testify/require"
)

func TestAdapter_Configured(t *testing.T) {
	require.False(t, New(Config{}).Configured())
	require.True(t, New(Config{
		APIKey: "k", PrivateKey: "pk", PublicKey: "pub",
	}).Configured())
}

func TestAdapter_Provider(t *testing.T) {
	require.Equal(t, payment.ProviderWaffo, New(Config{}).Provider())
}

func TestAdapter_CreateCheckout_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{TradeNo: "T1", Money: 10})
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

func TestAdapter_CreateCheckout_InvalidRequest(t *testing.T) {
	a := New(Config{APIKey: "k", PrivateKey: "pk", PublicKey: "pub"})
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
	a := New(Config{APIKey: "k", PrivateKey: "pk", PublicKey: "pub"})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{})
	require.ErrorIs(t, err, payment.ErrInvalidRequest)
}

func TestAdapter_VerifyWebhook_MissingSignature(t *testing.T) {
	a := New(Config{APIKey: "k", PrivateKey: "pk", PublicKey: "pub"})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{RawBody: []byte("{}")})
	require.ErrorIs(t, err, payment.ErrInvalidSignature)
}

func TestAdapter_BuildWebhookResponse_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.BuildWebhookResponse(context.Background(), true, "")
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

func TestAdapter_QueryOrder_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.QueryOrder(context.Background(), &payment.OrderQuery{TradeNo: "T1"})
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

func TestAdapter_QueryOrder_InvalidRequest(t *testing.T) {
	a := New(Config{APIKey: "k", PrivateKey: "pk", PublicKey: "pub"})
	_, err := a.QueryOrder(context.Background(), &payment.OrderQuery{})
	require.ErrorIs(t, err, payment.ErrInvalidRequest)
}

func TestFormatAmount(t *testing.T) {
	require.Equal(t, "12.34", formatAmount(12.34, "USD"))
	require.Equal(t, "12", formatAmount(12.34, "JPY"))
	require.Equal(t, "1000", formatAmount(1000.0, "VND"))
}
