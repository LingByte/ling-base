package waffopancake

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/common/payment"
	"github.com/stretchr/testify/require"
)

func TestAdapter_Configured(t *testing.T) {
	require.False(t, New(Config{}).Configured())
	require.True(t, New(Config{
		MerchantID: "MER_x", PrivateKey: "pk", ProductID: "PROD_x",
	}).Configured())
}

func TestAdapter_Provider(t *testing.T) {
	require.Equal(t, payment.ProviderWaffoPancake, New(Config{}).Provider())
}

func TestAdapter_CreateCheckout_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{TradeNo: "T1", Money: 10})
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

func TestAdapter_CreateCheckout_InvalidRequest(t *testing.T) {
	a := New(Config{MerchantID: "MER_x", PrivateKey: "pk", ProductID: "PROD_x"})
	_, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{Money: 10})
	require.ErrorIs(t, err, payment.ErrInvalidRequest)
}

func TestAdapter_VerifyWebhook_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{RawBody: []byte("{}")})
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

func TestAdapter_VerifyWebhook_EmptyBody(t *testing.T) {
	a := New(Config{MerchantID: "MER_x", PrivateKey: "pk", ProductID: "PROD_x"})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{})
	require.ErrorIs(t, err, payment.ErrInvalidRequest)
}

func TestAdapter_VerifyWebhook_MissingSignature(t *testing.T) {
	a := New(Config{MerchantID: "MER_x", PrivateKey: "pk", ProductID: "PROD_x"})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{RawBody: []byte("{}")})
	require.ErrorIs(t, err, payment.ErrInvalidSignature)
}

func TestAdapter_BuildWebhookResponse(t *testing.T) {
	a := New(Config{})
	r, err := a.BuildWebhookResponse(context.Background(), true, "")
	require.NoError(t, err)
	require.Equal(t, 200, r.StatusCode)
	require.Equal(t, "OK", string(r.Body))

	r, err = a.BuildWebhookResponse(context.Background(), false, "boom")
	require.NoError(t, err)
	require.Equal(t, 500, r.StatusCode)
	require.Equal(t, "retry", string(r.Body))
}

func TestAdapter_QueryOrder_Unsupported(t *testing.T) {
	a := New(Config{MerchantID: "MER_x", PrivateKey: "pk", ProductID: "PROD_x"})
	_, err := a.QueryOrder(context.Background(), &payment.OrderQuery{TradeNo: "T1"})
	require.ErrorIs(t, err, payment.ErrProviderError)
}

func TestOptionalString(t *testing.T) {
	require.Nil(t, optionalString(""))
	require.Nil(t, optionalString("   "))
	v := optionalString("x")
	require.NotNil(t, v)
	require.Equal(t, "x", *v)
}
