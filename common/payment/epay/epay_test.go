package epay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestAdapter_CreateCheckout_Redirect_OK(t *testing.T) {
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

func TestAdapter_CreateCheckout_Redirect_DefaultsDeviceAndName(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	res, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{
		TradeNo: "T1", Money: 1,
	})
	require.NoError(t, err)
	require.Equal(t, "pc", res.Params["device"])
	require.Equal(t, "Order T1", res.Params["name"])
}

func TestAdapter_CreateCheckout_Redirect_NoTypeOmitsField(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	res, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{
		TradeNo: "T1", Money: 1,
	})
	require.NoError(t, err)
	_, hasType := res.Params["type"]
	require.False(t, hasType, "type should be omitted when ProductID is empty")
}

func TestAdapter_CreateCheckout_Redirect_ParamAndCid(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	res, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{
		TradeNo:  "T1",
		Money:    1,
		Metadata: map[string]string{"param": "金色256G", "cid": "1234"},
	})
	require.NoError(t, err)
	require.Equal(t, "金色256G", res.Params["param"])
	require.Equal(t, "1234", res.Params["cid"])
}

func TestAdapter_CreateCheckout_API_RequiresClientIP(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"}, WithCheckoutMode(ModeAPI))
	_, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{
		TradeNo: "T1", Money: 1,
	})
	require.ErrorIs(t, err, payment.ErrInvalidRequest)
	require.ErrorContains(t, err, "client_ip")
}

func TestAdapter_CreateCheckout_API_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/mapi.php", r.URL.Path)
		require.Equal(t, "POST", r.Method)
		_ = r.ParseForm()
		require.Equal(t, "1001", r.PostForm.Get("pid"))
		require.Equal(t, "ORDER-API-1", r.PostForm.Get("out_trade_no"))
		require.Equal(t, "alipay", r.PostForm.Get("type"))
		require.Equal(t, "192.168.1.100", r.PostForm.Get("clientip"))
		require.NotEmpty(t, r.PostForm.Get("sign"))

		json.NewEncoder(w).Encode(map[string]any{
			"code":     1,
			"msg":      "",
			"O_id":     "ZPAY-999",
			"trade_no": "ORDER-API-1",
			"payurl":   "https://pay.example.com/pay/abc",
		})
	}))
	defer srv.Close()

	a := New(Config{PartnerID: "1001", Key: "secret", BaseURL: srv.URL}, WithCheckoutMode(ModeAPI))
	res, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{
		TradeNo:   "ORDER-API-1",
		Money:     1.00,
		Name:      "Test",
		ProductID: "alipay",
		ClientIP:  "192.168.1.100",
		NotifyURL: "https://me.com/notify",
		ReturnURL: "https://me.com/return",
	})
	require.NoError(t, err)
	require.Equal(t, payment.ProviderEpay, res.Provider)
	require.Equal(t, "https://pay.example.com/pay/abc", res.CheckoutURL)
	require.Equal(t, "ZPAY-999", res.SessionID)
}

func TestAdapter_CreateCheckout_API_QrcodeFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"code":   1,
			"qrcode": "https://pay.example.com/qr/abc",
			"img":    "https://pay.example.com/qr/abc.png",
		})
	}))
	defer srv.Close()

	a := New(Config{PartnerID: "p", Key: "k", BaseURL: srv.URL}, WithCheckoutMode(ModeAPI))
	res, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{
		TradeNo: "T1", Money: 1, ClientIP: "1.2.3.4",
	})
	require.NoError(t, err)
	require.Equal(t, "https://pay.example.com/qr/abc", res.CheckoutURL)
}

func TestAdapter_CreateCheckout_API_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1,
			"msg":  "签名错误",
		})
	}))
	defer srv.Close()

	a := New(Config{PartnerID: "p", Key: "k", BaseURL: srv.URL}, WithCheckoutMode(ModeAPI))
	_, err := a.CreateCheckout(context.Background(), &payment.CheckoutRequest{
		TradeNo: "T1", Money: 1, ClientIP: "1.2.3.4",
	})
	require.ErrorIs(t, err, payment.ErrProviderError)
	require.ErrorContains(t, err, "签名错误")
}

func TestAdapter_VerifyWebhook_SignatureRoundTrip(t *testing.T) {
	a := New(Config{PartnerID: "1001", Key: "secret", BaseURL: "https://pay.example.com"})

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

// --- QueryOrder ---

func TestAdapter_QueryOrder_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api.php", r.URL.Path)
		require.Equal(t, "order", r.URL.Query().Get("act"))
		require.Equal(t, "1001", r.URL.Query().Get("pid"))
		require.Equal(t, "secret", r.URL.Query().Get("key"))
		require.Equal(t, "ORDER-1", r.URL.Query().Get("out_trade_no"))

		json.NewEncoder(w).Encode(map[string]any{
			"code":         1,
			"msg":          "查询订单号成功",
			"trade_no":     "EPAY-999",
			"out_trade_no": "ORDER-1",
			"type":         "alipay",
			"money":        "12.34",
			"status":       1,
			"endtime":      "2024-01-15 10:30:00",
		})
	}))
	defer srv.Close()

	a := New(Config{PartnerID: "1001", Key: "secret", BaseURL: srv.URL})
	res, err := a.QueryOrder(context.Background(), &payment.OrderQuery{TradeNo: "ORDER-1"})
	require.NoError(t, err)
	require.Equal(t, payment.StatusSuccess, res.Status)
	require.Equal(t, "ORDER-1", res.TradeNo)
	require.Equal(t, "EPAY-999", res.ProviderOrderID)
	require.Equal(t, payment.Money(12.34), res.MoneyPaid)
	require.NotNil(t, res.PaidAt)
}

func TestAdapter_QueryOrder_Unpaid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"code":         1,
			"out_trade_no": "ORDER-1",
			"status":       0,
		})
	}))
	defer srv.Close()

	a := New(Config{PartnerID: "p", Key: "k", BaseURL: srv.URL})
	res, err := a.QueryOrder(context.Background(), &payment.OrderQuery{TradeNo: "ORDER-1"})
	require.NoError(t, err)
	require.Equal(t, payment.StatusPending, res.Status)
}

func TestAdapter_QueryOrder_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1,
			"msg":  "订单不存在",
		})
	}))
	defer srv.Close()

	a := New(Config{PartnerID: "p", Key: "k", BaseURL: srv.URL})
	_, err := a.QueryOrder(context.Background(), &payment.OrderQuery{TradeNo: "NOPE"})
	require.ErrorIs(t, err, payment.ErrOrderNotFound)
}

func TestAdapter_QueryOrder_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.QueryOrder(context.Background(), &payment.OrderQuery{TradeNo: "T1"})
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

func TestAdapter_QueryOrder_InvalidRequest(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	_, err := a.QueryOrder(context.Background(), &payment.OrderQuery{})
	require.ErrorIs(t, err, payment.ErrInvalidRequest)
}

// --- QueryBalance ---

func TestAdapter_QueryBalance_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "balance", r.URL.Query().Get("act"))
		json.NewEncoder(w).Encode(map[string]any{
			"code":    1,
			"msg":     "查询账户余额成功",
			"balance": "123.45",
		})
	}))
	defer srv.Close()

	a := New(Config{PartnerID: "p", Key: "k", BaseURL: srv.URL})
	res, err := a.QueryBalance(context.Background())
	require.NoError(t, err)
	require.Equal(t, payment.Money(123.45), res.Balance)
}

func TestAdapter_QueryBalance_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.QueryBalance(context.Background())
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

// --- Refund ---

func TestAdapter_Refund_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api.php", r.URL.Path)
		require.Equal(t, "POST", r.Method)
		_ = r.ParseForm()
		require.Equal(t, "refund", r.PostForm.Get("act"))
		require.Equal(t, "1001", r.PostForm.Get("pid"))
		require.Equal(t, "secret", r.PostForm.Get("key"))
		require.Equal(t, "ORDER-1", r.PostForm.Get("out_trade_no"))
		require.Equal(t, "12.34", r.PostForm.Get("money"))

		json.NewEncoder(w).Encode(map[string]any{
			"code": 1,
			"msg":  "退款成功",
		})
	}))
	defer srv.Close()

	a := New(Config{PartnerID: "1001", Key: "secret", BaseURL: srv.URL})
	res, err := a.Refund(context.Background(), &payment.RefundRequest{
		TradeNo: "ORDER-1",
		Money:   12.34,
	})
	require.NoError(t, err)
	require.Equal(t, payment.StatusRefunded, res.Status)
	require.Equal(t, "ORDER-1", res.TradeNo)
}

func TestAdapter_Refund_ByProviderOrderID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		require.Equal(t, "EPAY-999", r.PostForm.Get("trade_no"))
		require.Empty(t, r.PostForm.Get("out_trade_no"))
		json.NewEncoder(w).Encode(map[string]any{"code": 1, "msg": "退款成功"})
	}))
	defer srv.Close()

	a := New(Config{PartnerID: "p", Key: "k", BaseURL: srv.URL})
	_, err := a.Refund(context.Background(), &payment.RefundRequest{
		ProviderOrderID: "EPAY-999",
		Money:           1.00,
	})
	require.NoError(t, err)
}

func TestAdapter_Refund_Failed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1,
			"msg":  "退款金额超出限制",
		})
	}))
	defer srv.Close()

	a := New(Config{PartnerID: "p", Key: "k", BaseURL: srv.URL})
	_, err := a.Refund(context.Background(), &payment.RefundRequest{
		TradeNo: "T1", Money: 999.99,
	})
	require.ErrorIs(t, err, payment.ErrRefundFailed)
	require.ErrorContains(t, err, "退款金额超出限制")
}

func TestAdapter_Refund_InvalidRequest(t *testing.T) {
	a := New(Config{PartnerID: "p", Key: "k", BaseURL: "https://pay.example.com"})
	cases := []struct {
		name string
		req  *payment.RefundRequest
	}{
		{"nil", nil},
		{"no ids", &payment.RefundRequest{Money: 1}},
		{"zero money", &payment.RefundRequest{TradeNo: "T1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := a.Refund(context.Background(), c.req)
			require.ErrorIs(t, err, payment.ErrInvalidRequest)
		})
	}
}

func TestAdapter_Refund_NotConfigured(t *testing.T) {
	a := New(Config{})
	_, err := a.Refund(context.Background(), &payment.RefundRequest{TradeNo: "T1", Money: 1})
	require.ErrorIs(t, err, payment.ErrNotConfigured)
}

// --- Interface compliance ---

func TestAdapter_ImplementsRefunder(t *testing.T) {
	var _ payment.Refunder = (*Adapter)(nil)
}

func TestAdapter_ImplementsBalanceQuerier(t *testing.T) {
	var _ payment.BalanceQuerier = (*Adapter)(nil)
}
