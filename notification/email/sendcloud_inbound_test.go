// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package email

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// ParseSendCloudInboundEvent
// ──────────────────────────────────────────────

func TestParseSendCloudInboundEvent_JSON(t *testing.T) {
	raw := `{
		"event": "route",
		"message": "mx route",
		"timestamp": 1692000000,
		"from": "user@example.com",
		"fromname": "User",
		"to": "reply@yourdomain.com",
		"subject": "Re: Welcome",
		"text": "Thanks for the welcome!",
		"html": "<p>Thanks!</p>",
		"reference": "1644468027883_1024_25239_6195.sg-10_1_253_1-inbound0@ifaxin.com",
		"emailId": "1644468027883_1024_25239_6195",
		"token": "abc123",
		"signature": "def456",
		"labelId": 42,
		"labelName": "welcome"
	}`

	event, err := ParseSendCloudInboundEvent([]byte(raw))
	require.NoError(t, err)
	assert.Equal(t, "route", event.Event)
	assert.Equal(t, "user@example.com", event.From)
	assert.Equal(t, "User", event.FromName)
	assert.Equal(t, "reply@yourdomain.com", event.To)
	assert.Equal(t, "Re: Welcome", event.Subject)
	assert.Equal(t, "Thanks for the welcome!", event.Text)
	assert.Equal(t, int64(1692000000), event.Timestamp)
	assert.Equal(t, 42, event.LabelID)
	assert.Equal(t, "welcome", event.LabelName)
	assert.Equal(t, "abc123", event.Token)
	assert.Equal(t, "def456", event.Signature)
}

func TestParseSendCloudInboundEvent_FormEncoded(t *testing.T) {
	form := url.Values{}
	form.Set("event", "route")
	form.Set("from", "user@example.com")
	form.Set("to", "reply@yourdomain.com")
	form.Set("subject", "Re: Hello")
	form.Set("text", "Hello back")
	form.Set("timestamp", "1692000000")
	form.Set("labelId", "10")
	form.Set("token", "tok")
	form.Set("signature", "sig")

	event, err := ParseSendCloudInboundEvent([]byte(form.Encode()))
	require.NoError(t, err)
	assert.Equal(t, "route", event.Event)
	assert.Equal(t, "user@example.com", event.From)
	assert.Equal(t, "reply@yourdomain.com", event.To)
	assert.Equal(t, "Re: Hello", event.Subject)
	assert.Equal(t, "Hello back", event.Text)
	assert.Equal(t, int64(1692000000), event.Timestamp)
	assert.Equal(t, 10, event.LabelID)
	assert.Equal(t, "tok", event.Token)
	assert.Equal(t, "sig", event.Signature)
}

func TestParseSendCloudInboundEvent_InvalidJSON_NonRoute(t *testing.T) {
	// JSON that's not a route event should fall through to form parsing.
	raw := `{"event":"deliver","messageId":"123"}`
	event, err := ParseSendCloudInboundEvent([]byte(raw))
	require.NoError(t, err)
	// Form parsing of the JSON string will produce empty fields.
	assert.NotEqual(t, "route", event.Event)
}

func TestParseSendCloudInboundEvent_EmptyBody(t *testing.T) {
	event, err := ParseSendCloudInboundEvent([]byte(""))
	require.NoError(t, err)
	assert.Empty(t, event.Event)
}

// ──────────────────────────────────────────────
// ToMailMessage
// ──────────────────────────────────────────────

func TestSendCloudInboundEvent_ToMailMessage_FieldsOnly(t *testing.T) {
	event := &SendCloudInboundEvent{
		Reference: "ref-123",
		From:      "user@example.com",
		FromName:  "User",
		To:        "reply@yourdomain.com",
		Subject:   "Re: Welcome",
		Text:      "Thanks!",
		HTML:      "<p>Thanks!</p>",
		Timestamp: 1692000000,
	}

	msg, err := event.ToMailMessage()
	require.NoError(t, err)
	assert.Equal(t, "ref-123", msg.ID)
	assert.Equal(t, "user@example.com", msg.From)
	assert.Equal(t, "User", msg.FromName)
	assert.Equal(t, []string{"reply@yourdomain.com"}, msg.To)
	assert.Equal(t, "Re: Welcome", msg.Subject)
	assert.Equal(t, "Thanks!", msg.TextBody)
	assert.Equal(t, "<p>Thanks!</p>", msg.HTMLBody)
	assert.False(t, msg.Date.IsZero())
}

func TestSendCloudInboundEvent_ToMailMessage_WithRawMessage(t *testing.T) {
	rawMsg := "From: user@example.com\r\n" +
		"To: reply@yourdomain.com\r\n" +
		"Subject: Re: Welcome\r\n" +
		"Message-ID: <abc@example.com>\r\n" +
		"Date: Mon, 14 Aug 2026 10:00:00 +0000\r\n" +
		"\r\n" +
		"This is the reply body."

	event := &SendCloudInboundEvent{
		From:       "user@example.com",
		To:         "reply@yourdomain.com",
		Subject:    "Re: Welcome",
		Text:       "fallback text",
		RawMessage: rawMsg,
	}

	msg, err := event.ToMailMessage()
	require.NoError(t, err)
	// Raw message parsing should override the text field.
	assert.Equal(t, "This is the reply body.", msg.TextBody)
	assert.Equal(t, "<abc@example.com>", msg.ID)
	assert.Equal(t, "Re: Welcome", msg.Subject)
}

func TestSendCloudInboundEvent_ToMailMessage_EmptyEvent(t *testing.T) {
	event := &SendCloudInboundEvent{}
	msg, err := event.ToMailMessage()
	require.NoError(t, err)
	assert.Empty(t, msg.From)
	assert.Empty(t, msg.Subject)
	assert.True(t, msg.Date.IsZero())
}

// ──────────────────────────────────────────────
// VerifySendCloudInboundSignature
// ──────────────────────────────────────────────

func TestVerifySendCloudInboundSignature_Valid(t *testing.T) {
	apiKey := "mysecretkey"
	token := "randomtoken123"
	sig := md5Hex(token + apiKey)

	event := &SendCloudInboundEvent{
		Token:     token,
		Signature: sig,
	}
	assert.True(t, VerifySendCloudInboundSignature(event, apiKey))
}

func TestVerifySendCloudInboundSignature_Invalid(t *testing.T) {
	event := &SendCloudInboundEvent{
		Token:     "tok",
		Signature: "wrong",
	}
	assert.False(t, VerifySendCloudInboundSignature(event, "key"))
}

func TestVerifySendCloudInboundSignature_EmptyToken(t *testing.T) {
	event := &SendCloudInboundEvent{
		Signature: "sig",
	}
	assert.False(t, VerifySendCloudInboundSignature(event, "key"))
}

func TestVerifySendCloudInboundSignature_EmptySignature(t *testing.T) {
	event := &SendCloudInboundEvent{
		Token: "tok",
	}
	assert.False(t, VerifySendCloudInboundSignature(event, "key"))
}

func TestVerifySendCloudInboundSignature_NilEvent(t *testing.T) {
	assert.False(t, VerifySendCloudInboundSignature(nil, "key"))
}

// ──────────────────────────────────────────────
// md5Hex
// ──────────────────────────────────────────────

func TestMd5Hex(t *testing.T) {
	// md5("hello") = 5d41402abc4b2a76b9719d911017c592
	assert.Equal(t, "5d41402abc4b2a76b9719d911017c592", md5Hex("hello"))
}

func TestMd5Hex_Empty(t *testing.T) {
	// md5("") = d41d8cd98f00b204e9800998ecf8427e
	assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", md5Hex(""))
}

// ──────────────────────────────────────────────
// Route management API
// ──────────────────────────────────────────────

func newSendCloudRouteTestServer(t *testing.T, handler http.HandlerFunc) *SendCloudReader {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	r, err := NewSendCloudReader(SendCloudConfig{APIUser: "u", APIKey: "k"})
	require.NoError(t, err)
	r.endpoint = srv.URL
	r.client = srv.Client()
	return r
}

func TestSendCloudReader_ListInboundRoutes(t *testing.T) {
	body := `{
		"result": true,
		"statusCode": 200,
		"message": "",
		"info": {
			"total": 2,
			"count": 2,
			"dataList": [
				{"id": 1, "domain": "example.com", "expression": "reply@example.com", "action": "https://app.com/wh"},
				{"id": 2, "domain": "example.com", "expression": ".*@example.com", "action": "admin@example.com", "apiUserRoute": "test"}
			]
		}
	}`

	reader := newSendCloudRouteTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "u", r.FormValue("apiUser"))
		assert.Equal(t, "example.com", r.FormValue("domain"))
		assert.Equal(t, "100", r.FormValue("limit"))
		assert.True(t, strings.Contains(r.URL.Path, "route/list"))
		w.Write([]byte(body))
	})

	routes, err := reader.ListInboundRoutes("example.com", 0, 100)
	require.NoError(t, err)
	require.Len(t, routes, 2)
	assert.Equal(t, 1, routes[0].ID)
	assert.Equal(t, "reply@example.com", routes[0].Expression)
	assert.Equal(t, "https://app.com/wh", routes[0].Action)
	assert.Equal(t, 2, routes[1].ID)
	assert.Equal(t, "admin@example.com", routes[1].Action)
	assert.Equal(t, "test", routes[1].APIUserRoute)
}

func TestSendCloudReader_ListInboundRoutes_Empty(t *testing.T) {
	reader := newSendCloudRouteTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":true,"statusCode":200,"message":"","info":{"voListSize":0,"voList":[]}}`))
	})

	routes, err := reader.ListInboundRoutes("", 0, 0)
	require.NoError(t, err)
	assert.Empty(t, routes)
}

func TestSendCloudReader_ListInboundRoutes_APIError(t *testing.T) {
	reader := newSendCloudRouteTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":false,"statusCode":500,"message":"denied","info":{}}`))
	})

	_, err := reader.ListInboundRoutes("", 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "denied")
}

func TestSendCloudReader_AddInboundRoute(t *testing.T) {
	reader := newSendCloudRouteTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "contact@example.com", r.FormValue("expression"))
		assert.Equal(t, "https://app.com/webhook", r.FormValue("action"))
		assert.True(t, strings.Contains(r.URL.Path, "route/add"))
		w.Write([]byte(`{"result":true,"statusCode":200,"message":"","info":{"id":42}}`))
	})

	id, err := reader.AddInboundRoute("contact@example.com", "https://app.com/webhook", "")
	require.NoError(t, err)
	assert.Equal(t, 42, id)
}

func TestSendCloudReader_AddInboundRoute_WithEmail(t *testing.T) {
	reader := newSendCloudRouteTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "admin@example.com", r.FormValue("action"))
		assert.Equal(t, "myapiuser", r.FormValue("apiUserRoute"))
		w.Write([]byte(`{"result":true,"statusCode":200,"message":"","info":{"id":99}}`))
	})

	id, err := reader.AddInboundRoute("contact@example.com", "admin@example.com", "myapiuser")
	require.NoError(t, err)
	assert.Equal(t, 99, id)
}

func TestSendCloudReader_AddInboundRoute_Error(t *testing.T) {
	reader := newSendCloudRouteTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":false,"statusCode":400,"message":"invalid expression","info":{}}`))
	})

	_, err := reader.AddInboundRoute("bad", "URL", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid expression")
}

func TestSendCloudReader_DeleteInboundRoute(t *testing.T) {
	reader := newSendCloudRouteTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "42", r.FormValue("routeId"))
		assert.True(t, strings.Contains(r.URL.Path, "route/delete"))
		w.Write([]byte(`{"result":true,"statusCode":200,"message":"","info":{}}`))
	})

	err := reader.DeleteInboundRoute(42)
	require.NoError(t, err)
}

func TestSendCloudReader_DeleteInboundRoute_Error(t *testing.T) {
	reader := newSendCloudRouteTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":false,"statusCode":404,"message":"route not found","info":{}}`))
	})

	err := reader.DeleteInboundRoute(999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "route not found")
}

func TestSendCloudReader_DeleteInboundRoute_HTTPError(t *testing.T) {
	reader := newSendCloudRouteTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`server error`))
	})

	err := reader.DeleteInboundRoute(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http 500")
}

// ──────────────────────────────────────────────
// JSON round-trip
// ──────────────────────────────────────────────

func TestSendCloudInboundEvent_Unmarshal(t *testing.T) {
	raw := `{"event":"route","from":"a@b.com","to":"c@d.com","subject":"test","timestamp":123}`
	var event SendCloudInboundEvent
	require.NoError(t, json.Unmarshal([]byte(raw), &event))
	assert.Equal(t, "route", event.Event)
	assert.Equal(t, "a@b.com", event.From)
	assert.Equal(t, int64(123), event.Timestamp)
}

func TestSendCloudRoute_Unmarshal(t *testing.T) {
	raw := `{"id":5,"domain":"x.com","expression":"reply@x.com","action":"URL"}`
	var route SendCloudRoute
	require.NoError(t, json.Unmarshal([]byte(raw), &route))
	assert.Equal(t, 5, route.ID)
	assert.Equal(t, "x.com", route.Domain)
	assert.Equal(t, "reply@x.com", route.Expression)
}

// Ensure strconv import is used.
var _ = strconv.Itoa
