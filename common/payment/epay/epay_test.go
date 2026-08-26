package epay

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/common/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapter_Configured(t *testing.T) {
	a := New(Config{})
	require.False(t, a.Configured())

	a = New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	require.True(t, a.Configured())
}

func TestAdapter_Provider(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	require.Equal(t, payment.ProviderEpay, a.Provider())
}

func TestAdapter_CreateCheckout_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{TradeNo: "T1", Money: 10})
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

func TestAdapter_CreateCheckout_InvalidRequest(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	cases := []struct {
		name string
		req  *payment.CheckoutRequest
	}{
		{"missing trade_no", &payment.CheckoutRequest{Money: 10}},
		{"zero money", &payment.CheckoutRequest{TradeNo: "T1"}},
		{"negative money", &payment.CheckoutRequest{TradeNo: "T1", Money: -1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := a.CreateCheckout(context.Background(), c.req)
			require.ErrorIs(t, err, payment.ErrInvalidRequest)
		})
	}
}

func TestAdapter_CreateCheckout_OK(t *testing.T) {
	a := New(Config{PartnerID: "1001", Key: "secret", BaseURL: "https://pay.example.com"})
	res, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{
		TradeNo:    "ORDER-1",
		Money:      12.34,
		Currency:   "CNY",
		Name:       "Test Goods",
		NotifyURL:  "https://me.example.com/notify",
		ReturnURL:  "https://me.example.com/return",
		ProductID:  "alipay",
		Device:     payment.DevicePC,
	})
	require.NoError(t, err)
	require.Equal(t, payment.ProviderEpay, res.Provider)
	require.Equal(t, "https://pay.example.com/submit.php", res.CheckoutURL)

	require.Equal(t, "1001", res.Params["pid"])
	require.Equal(t, "ORDER-1", res.Params["out_trade_no"])
	require.Equal(t, "alipay", res.Params["type"])
	require.Equal(t, "12.34", res.Params["money"])
	require.Equal(t, "MD5", res.Params["sign_type"])
	require.NotEmpty(t, res.Params["sign"])
}

func TestAdapter_CreateCheckout_DefaultsDeviceAndName(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	res, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{
		TradeNo: "T1", Money: 1,
	})
	require.NoError(t, err)
	require.Equal(t, "pc", res.Params["device"])
	require.Equal(t, "Order T1", res.Params["name"])
}

func TestAdapter_CreateCheckout_NoTypeOmitsField(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	res, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{
		TradeNo: "T1", Money: 1,
	})
	require.NoError(t, err)
	_, hasType := res.Params["type"]
	require.False(t, hasType, "type should be omitted when ProductID is empty")
}

func TestAdapter_VerifyWebhook_SignatureRoundTrip(t *testing.T) {
	a := New(Config{PartnerID: "1001", Key: "secret", BaseURL: "https://pay.example.com"})

	// Build a valid signed callback.
	params := map[string]string{
		"pid":          "1001",
		"type":         "alipay",
		"out_trade_no": "ORDER-1",
		"trade_no":     "EPAY-999",
		"name":         "Test",
		"money":        "12.34",
		"trade_status": "TRADE_SUCCESS",
	}
	params["sign"] = signParams(filterSigned(params), "secret")

	evt, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{
		Provider:   payment.ProviderEpay,
		FormParams: params,
	})
	require.NoError(t, err)
	require.Equal(t, payment.ProviderEpay, evt.Provider)
	require.Equal(t, payment.EventCheckoutCompleted, evt.EventType)
	require.Equal(t, payment.StatusSuccess, evt.Status)
	require.Equal(t, "ORDER-1", evt.TradeNo)
	require.Equal(t, "EPAY-999", evt.ProviderOrderID)
	require.Equal(t, payment.Money(12.34), evt.MoneyPaid)
}

func TestAdapter_VerifyWebhook_BadSignature(t *testing.T) {
	a := New(Config{PartnerID: "1001", Key: "secret", BaseURL: "https://pay.example.com"})
	params := map[string]string{
		"pid":          "1001",
		"out_trade_no": "ORDER-1",
		"trade_status": "TRADE_SUCCESS",
		"sign":         "deadbeef",
	}
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{
		FormParams: params,
	})
	require.ErrorIs(t, err, payment.ErrInvalidSignature)
}

func TestAdapter_VerifyWebhook_MissingSign(t *testing.T) {
	a := New(Config{PartnerID: "1001", Key: "secret", BaseURL: "https://pay.example.com"})
	_, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{
		FormParams: map[string]string{"pid": "1001"},
	})
	require.ErrorIs(t, err, payment.ErrInvalidSignature)
}

func TestAdapter_VerifyWebhook_NonSuccessStatus(t *testing.T) {
	a := New(Config{PartnerID: "1001", Key: "secret", BaseURL: "https://pay.example.com"})
	params := map[string]string{
		"pid":          "1001",
		"out_trade_no": "ORDER-1",
		"trade_status": "WAIT_BUYER_PAY",
	}
	params["sign"] = signParams(filterSigned(params), "secret")
	evt, err := a.VerifyWebhook(context.Background(), &payment.WebhookVerifyRequest{FormParams: params})
	require.NoError(t, err)
	assert.Equal(t, payment.EventUnknown, evt.EventType)
	assert.Equal(t, payment.StatusPending, evt.Status)
}

func TestAdapter_BuildWebhookResponse(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	r, err := a.BuildWebhookResponse(context.Background(), true, "")
	require.NoError(t, err)
	require.Equal(t, 200, r.StatusCode)
	require.Equal(t, "success", string(r.Body))

	r, err = a.BuildWebhookResponse(context.Background(), false, "boom")
	require.NoError(t, err)
	require.Equal(t, "fail", string(r.Body))
}

func TestAdapter_QueryOrder_Unsupported(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	_, err := a.QueryOrder(context.Background(), &payment.OrderQuery{TradeNo: "T1"})
	require.ErrorIs(t, err, payment.ErrProviderError)
}
