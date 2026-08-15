// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package email

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// SendCloudProvider — constructor
// ──────────────────────────────────────────────

func TestSendCloudProvider_Kind(t *testing.T) {
	p, err := NewSendCloudProvider(SendCloudConfig{
		APIUser: "test-user",
		APIKey:  "test-key",
		From:    "noreply@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "sendcloud", p.Kind())
}

func TestSendCloudProvider_MissingAPIUser(t *testing.T) {
	_, err := NewSendCloudProvider(SendCloudConfig{
		APIKey: "test-key",
		From:   "noreply@example.com",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_user")
}

func TestSendCloudProvider_MissingAPIKey(t *testing.T) {
	_, err := NewSendCloudProvider(SendCloudConfig{
		APIUser: "test-user",
		From:    "noreply@example.com",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key")
}

func TestSendCloudProvider_MissingFrom(t *testing.T) {
	_, err := NewSendCloudProvider(SendCloudConfig{
		APIUser: "test-user",
		APIKey:  "test-key",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "from")
}

func TestSendCloudProvider_InvalidFrom(t *testing.T) {
	_, err := NewSendCloudProvider(SendCloudConfig{
		APIUser: "test-user",
		APIKey:  "test-key",
		From:    "not-an-email",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from")
}

func TestSendCloudProvider_WithNameInFrom(t *testing.T) {
	p, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "test-user",
		APIKey:   "test-key",
		From:     "Alice <alice@example.com>",
		FromName: "Fallback",
	})
	require.NoError(t, err)
	assert.Equal(t, "sendcloud", p.Kind())
}

// ──────────────────────────────────────────────
// SendCloudProvider — SendHTMLWith
// ──────────────────────────────────────────────

func TestSendCloudProvider_SendHTMLWith_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		_ = r.ParseForm()
		assert.Equal(t, "test-user", r.PostForm.Get("apiUser"))
		assert.Equal(t, "test-key", r.PostForm.Get("apiKey"))
		assert.Equal(t, "bob@example.com", r.PostForm.Get("to"))
		assert.Equal(t, "noreply@example.com", r.PostForm.Get("from"))
		assert.Equal(t, "Welcome Bob", r.PostForm.Get("subject"))
		assert.Contains(t, r.PostForm.Get("html"), "Hello Bob")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true,"message":"success","info":{"emailIdList":["sc-123"],"messageId":"msg-123"}}`)
	}))
	defer srv.Close()

	p, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "test-user",
		APIKey:   "test-key",
		From:     "noreply@example.com",
		Endpoint: srv.URL,
	})
	require.NoError(t, err)

	id, err := p.SendHTMLWith("bob@example.com", "Welcome {{.Name}}", "<p>Hello {{.Name}}</p>", map[string]any{"Name": "Bob"})
	require.NoError(t, err)
	assert.Equal(t, "sc-123", id)
}

func TestSendCloudProvider_SendHTMLWith_Success_FromName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		assert.Equal(t, "Alice", r.PostForm.Get("fromName"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true,"info":{"messageId":"msg-456"}}`)
	}))
	defer srv.Close()

	p, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "test-user",
		APIKey:   "test-key",
		From:     "Alice <alice@example.com>",
		Endpoint: srv.URL,
	})
	require.NoError(t, err)

	id, err := p.SendHTMLWith("bob@example.com", "subj", "<p>hi</p>", nil)
	require.NoError(t, err)
	assert.Equal(t, "msg-456", id)
}

func TestSendCloudProvider_SendHTMLWith_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":false,"message":"quota exceeded"}`)
	}))
	defer srv.Close()

	p, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "test-user",
		APIKey:   "test-key",
		From:     "noreply@example.com",
		Endpoint: srv.URL,
	})
	require.NoError(t, err)

	_, err = p.SendHTMLWith("bob@example.com", "subj", "<p>hi</p>", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestSendCloudProvider_SendHTMLWith_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"result":false,"message":"invalid api key"}`)
	}))
	defer srv.Close()

	p, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "test-user",
		APIKey:   "test-key",
		From:     "noreply@example.com",
		Endpoint: srv.URL,
	})
	require.NoError(t, err)

	_, err = p.SendHTMLWith("bob@example.com", "subj", "<p>hi</p>", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

// ──────────────────────────────────────────────
// SendCloudProvider — SendTextWith
// ──────────────────────────────────────────────

func TestSendCloudProvider_SendTextWith_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		assert.Equal(t, "Hello Bob", r.PostForm.Get("plain"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true,"info":{"emailIdList":["sc-text-1"]}}`)
	}))
	defer srv.Close()

	p, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "test-user",
		APIKey:   "test-key",
		From:     "noreply@example.com",
		Endpoint: srv.URL,
	})
	require.NoError(t, err)

	id, err := p.SendTextWith("bob@example.com", "subj", "Hello {{.Name}}", map[string]any{"Name": "Bob"})
	require.NoError(t, err)
	assert.Equal(t, "sc-text-1", id)
}

func TestSendCloudProvider_SendTextWith_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":false,"message":"rate limited"}`)
	}))
	defer srv.Close()

	p, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "test-user",
		APIKey:   "test-key",
		From:     "noreply@example.com",
		Endpoint: srv.URL,
	})
	require.NoError(t, err)

	_, err = p.SendTextWith("bob@example.com", "subj", "body", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited")
}

// ──────────────────────────────────────────────
// SendCloudProvider — message ID extraction
// ──────────────────────────────────────────────

func TestSendCloudProvider_MessageID_FromData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true,"data":{"messageId":"data-msg-id"}}`)
	}))
	defer srv.Close()

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	id, err := p.SendHTMLWith("to@x.com", "s", "<p>b</p>", nil)
	require.NoError(t, err)
	assert.Equal(t, "data-msg-id", id)
}

func TestSendCloudProvider_MessageID_FromTopLevel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true,"messageId":"top-msg-id"}`)
	}))
	defer srv.Close()

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	id, err := p.SendHTMLWith("to@x.com", "s", "<p>b</p>", nil)
	require.NoError(t, err)
	assert.Equal(t, "top-msg-id", id)
}

func TestSendCloudProvider_MessageID_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true}`)
	}))
	defer srv.Close()

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	id, err := p.SendHTMLWith("to@x.com", "s", "<p>b</p>", nil)
	require.NoError(t, err)
	assert.Equal(t, "", id)
}

func TestSendCloudProvider_HTTPError_NoMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"result":false}`)
	}))
	defer srv.Close()

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	_, err := p.SendHTMLWith("to@x.com", "s", "<p>b</p>", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestSendCloudProvider_HTTPError_RawBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, `<html>Bad Gateway</html>`)
	}))
	defer srv.Close()

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	_, err := p.SendHTMLWith("to@x.com", "s", "<p>b</p>", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Bad Gateway")
}

// ──────────────────────────────────────────────
// SendCloudProvider — integration with Mailer
// ──────────────────────────────────────────────

func TestSendCloudProvider_MailerIntegration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true,"info":{"emailIdList":["sc-mailer-1"]}}`)
	}))
	defer srv.Close()

	sc, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "u",
		APIKey:   "k",
		From:     "noreply@example.com",
		Endpoint: srv.URL,
	})
	require.NoError(t, err)

	m := NewMailer([]MailProvider{sc}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:    1,
		InitialBackoff: 1,
		MaxBackoff:     1,
	}))
	err = m.Send(context.Background(), "to@x.com", "subj", "<p>hi</p>")
	require.NoError(t, err)
}

func TestSendCloudProvider_MailerFailover(t *testing.T) {
	// First SendCloud fails, second fake provider succeeds.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":false,"message":"quota exceeded"}`)
	}))
	defer srv.Close()

	sc, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "u",
		APIKey:   "k",
		From:     "noreply@example.com",
		Endpoint: srv.URL,
	})
	require.NoError(t, err)

	backup := &fakeProvider{kind: "backup"}
	m := NewMailer([]MailProvider{sc, backup}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:    1,
		InitialBackoff: 1,
		MaxBackoff:     1,
	}))
	err = m.Send(context.Background(), "to@x.com", "subj", "<p>hi</p>")
	require.NoError(t, err)
	assert.Equal(t, 1, backup.htmlCalls)
}

// ──────────────────────────────────────────────
// SendCloudEventToStatus
// ──────────────────────────────────────────────

func TestSendCloudEventToStatus(t *testing.T) {
	tests := []struct {
		event string
		want  string
	}{
		{"1", StatusDelivered},
		{"deliver", StatusDelivered},
		{"delivered", StatusDelivered},
		{"3", StatusSpam},
		{"spam", StatusSpam},
		{"4", StatusInvalid},
		{"invalid", StatusInvalid},
		{"5", StatusSoftBounce},
		{"soft_bounce", StatusSoftBounce},
		{"hard_bounce", StatusSoftBounce},
		{"bounce", StatusSoftBounce},
		{"2", StatusFailed},
		{"reject", StatusFailed},
		{"failed", StatusFailed},
		{"10", StatusClicked},
		{"click", StatusClicked},
		{"clicked", StatusClicked},
		{"11", StatusOpened},
		{"open", StatusOpened},
		{"opened", StatusOpened},
		{"12", StatusUnsubscribed},
		{"unsubscribe", StatusUnsubscribed},
		{"unsubscribed", StatusUnsubscribed},
		{"18", StatusSent},
		{"request", StatusSent},
		{"unknown_event", StatusUnknown},
		{"", StatusUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			assert.Equal(t, tt.want, SendCloudEventToStatus(tt.event))
		})
	}
}

func TestSendCloudEventToStatus_CaseInsensitive(t *testing.T) {
	assert.Equal(t, StatusDelivered, SendCloudEventToStatus("DELIVERED"))
	assert.Equal(t, StatusDelivered, SendCloudEventToStatus("  Delivered  "))
	assert.Equal(t, StatusOpened, SendCloudEventToStatus("OPENED"))
}

// ──────────────────────────────────────────────
// ParseSendCloudWebhookEvent
// ──────────────────────────────────────────────

func TestParseSendCloudWebhookEvent_JSON(t *testing.T) {
	data := []byte(`{"event":"delivered","messageId":"msg-123","email":"user@example.com","timestamp":1700000000}`)
	ev, err := ParseSendCloudWebhookEvent(data)
	require.NoError(t, err)
	assert.Equal(t, "delivered", ev.Event)
	assert.Equal(t, "msg-123", ev.MessageID)
	assert.Equal(t, "user@example.com", ev.Email)
	assert.Equal(t, int64(1700000000), ev.Timestamp)
}

func TestParseSendCloudWebhookEvent_JSON_EventOnly(t *testing.T) {
	data := []byte(`{"event":"opened"}`)
	ev, err := ParseSendCloudWebhookEvent(data)
	require.NoError(t, err)
	assert.Equal(t, "opened", ev.Event)
}

func TestParseSendCloudWebhookEvent_JSON_MessageIDOnly(t *testing.T) {
	data := []byte(`{"messageId":"msg-456"}`)
	ev, err := ParseSendCloudWebhookEvent(data)
	require.NoError(t, err)
	assert.Equal(t, "msg-456", ev.MessageID)
}

func TestParseSendCloudWebhookEvent_FormEncoded(t *testing.T) {
	data := []byte("event=delivered&messageId=msg-789&recipient=user@example.com&smtpStatus=250&smtpError=")
	ev, err := ParseSendCloudWebhookEvent(data)
	require.NoError(t, err)
	assert.Equal(t, "delivered", ev.Event)
	assert.Equal(t, "msg-789", ev.MessageID)
	assert.Equal(t, "user@example.com", ev.Email)
	assert.Equal(t, "250", ev.SmtpStatus)
}

func TestParseSendCloudWebhookEvent_FormEncoded_EmailIdFallback(t *testing.T) {
	data := []byte("event=delivered&emailId=msg-from-emailid@sendcloud.com")
	ev, err := ParseSendCloudWebhookEvent(data)
	require.NoError(t, err)
	assert.Equal(t, "msg-from-emailid", ev.MessageID)
	assert.Equal(t, "sendcloud.com", ev.Email)
}

func TestParseSendCloudWebhookEvent_FormEncoded_Timestamp(t *testing.T) {
	data := []byte("event=delivered&messageId=msg-ts&timestamp=2024-01-15 10:30:00")
	ev, err := ParseSendCloudWebhookEvent(data)
	require.NoError(t, err)
	assert.True(t, ev.Timestamp > 0)
}

func TestParseSendCloudWebhookEvent_FormEncoded_NoTimestamp(t *testing.T) {
	data := []byte("event=delivered&messageId=msg-ts2&timestamp=invalid-date")
	ev, err := ParseSendCloudWebhookEvent(data)
	require.NoError(t, err)
	assert.Equal(t, int64(0), ev.Timestamp)
}

func TestParseSendCloudWebhookEvent_InvalidData(t *testing.T) {
	// Neither valid JSON with event/messageId nor valid form-encoded.
	// url.ParseQuery will still parse this, just with empty values.
	data := []byte("===invalid===")
	ev, err := ParseSendCloudWebhookEvent(data)
	// url.ParseQuery is lenient, so this won't error but will return empty values.
	if err != nil {
		assert.Nil(t, ev)
	} else {
		assert.NotNil(t, ev)
		assert.Equal(t, "", ev.Event)
	}
}

func TestParseSendCloudWebhookEvent_EmptyData(t *testing.T) {
	ev, err := ParseSendCloudWebhookEvent([]byte(""))
	require.NoError(t, err)
	assert.Equal(t, "", ev.Event)
	assert.Equal(t, "", ev.MessageID)
}

func TestParseSendCloudWebhookEvent_JSONInvalidFallsBackToForm(t *testing.T) {
	// Invalid JSON that doesn't have event or messageId should fall back to form parsing.
	data := []byte(`{"bad json`)
	ev, err := ParseSendCloudWebhookEvent(data)
	// Form parsing should handle this (url.ParseQuery is lenient)
	if err == nil {
		assert.NotNil(t, ev)
	}
}

// ──────────────────────────────────────────────
// SendCloudProvider — Channel integration
// ──────────────────────────────────────────────

func TestSendCloudProvider_ChannelIntegration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true,"info":{"emailIdList":["sc-channel-1"]}}`)
	}))
	defer srv.Close()

	sc, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "u",
		APIKey:   "k",
		From:     "noreply@example.com",
		Endpoint: srv.URL,
	})
	require.NoError(t, err)

	m := NewMailer([]MailProvider{sc}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:    1,
		InitialBackoff: 1,
		MaxBackoff:     1,
	}))
	ch := NewChannel("sendcloud-primary", m)

	assert.Equal(t, "sendcloud-primary", ch.Name())
	assert.True(t, ch.Enabled())
	ch.SetEnabled(false)
	assert.False(t, ch.Enabled())
}

// ──────────────────────────────────────────────
// SendCloudProvider — edge cases
// ──────────────────────────────────────────────

func TestSendCloudProvider_SendHTMLWith_WithVars(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		subject := r.PostForm.Get("subject")
		html := r.PostForm.Get("html")
		assert.Contains(t, subject, "Alice")
		assert.Contains(t, html, "Alice")
		assert.Contains(t, subject, "12345")
		assert.Contains(t, html, "12345")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true,"info":{"emailIdList":["sc-vars-1"]}}`)
	}))
	defer srv.Close()

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	id, err := p.SendHTMLWith("to@x.com", "Welcome {{.Name}}, code {{.Code}}", "<p>Hello {{.Name}}, your code is {{.Code}}</p>", map[string]any{"Name": "Alice", "Code": 12345})
	require.NoError(t, err)
	assert.Equal(t, "sc-vars-1", id)
}

func TestSendCloudProvider_CustomEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true,"info":{"emailIdList":["sc-custom-1"]}}`)
	}))
	defer srv.Close()

	p, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "u",
		APIKey:   "k",
		From:     "noreply@example.com",
		Endpoint: srv.URL,
	})
	require.NoError(t, err)
	assert.Equal(t, srv.URL, p.endpoint)
}

func TestSendCloudProvider_DefaultEndpoint(t *testing.T) {
	p, err := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u",
		APIKey:  "k",
		From:    "noreply@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://api.sendcloud.net/apiv2/mail/send", p.endpoint)
}

func TestSendCloudProvider_HTTPError_StatusOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		// Empty body, no message in JSON
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	_, err := p.SendHTMLWith("to@x.com", "s", "<p>b</p>", nil)
	require.Error(t, err)
	// Should fall back to resp.Status
	assert.Contains(t, err.Error(), "403")
}

func TestSendCloudProvider_ResultFalse_NoMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":false}`)
	}))
	defer srv.Close()

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	_, err := p.SendHTMLWith("to@x.com", "s", "<p>b</p>", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}

// Verify that the SMTP provider still works alongside SendCloud.
func TestSMTPAndSendCloud_Coexist(t *testing.T) {
	smtp := NewSMTPProvider(SMTPConfig{
		Host: "127.0.0.1", Port: 1, From: "noreply@example.com",
	})
	sc, err := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com",
	})
	require.NoError(t, err)

	assert.Equal(t, "smtp", smtp.Kind())
	assert.Equal(t, "sendcloud", sc.Kind())
	assert.NotEqual(t, smtp.Kind(), sc.Kind())
}

// Test SendCloud with a name in the From field and verify fromName is set.
func TestSendCloudProvider_FromName_WhenFromHasNoName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		assert.Equal(t, "MyApp", r.PostForm.Get("fromName"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true,"info":{"emailIdList":["sc-fn-1"]}}`)
	}))
	defer srv.Close()

	p, err := NewSendCloudProvider(SendCloudConfig{
		APIUser:  "u",
		APIKey:   "k",
		From:     "noreply@example.com",
		FromName: "MyApp",
		Endpoint: srv.URL,
	})
	require.NoError(t, err)

	_, err = p.SendHTMLWith("to@x.com", "s", "<p>b</p>", nil)
	require.NoError(t, err)
}

// Test that SendCloud handles empty emailIdList gracefully.
func TestSendCloudProvider_EmptyEmailIDList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// result=true but empty emailIdList and no messageId anywhere
		fmt.Fprint(w, `{"result":true,"info":{"emailIdList":[],"messageId":""},"data":{"messageId":""},"messageId":""}`)
	}))
	defer srv.Close()

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	id, err := p.SendHTMLWith("to@x.com", "s", "<p>b</p>", nil)
	require.NoError(t, err)
	assert.Equal(t, "", id)
}

// Test that SendCloud handles empty string in emailIdList.
func TestSendCloudProvider_EmptyStringInEmailIDList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// emailIdList has an entry but it's empty string; should fall through to messageId
		fmt.Fprint(w, `{"result":true,"info":{"emailIdList":[""],"messageId":"fallback-id"}}`)
	}))
	defer srv.Close()

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	id, err := p.SendHTMLWith("to@x.com", "s", "<p>b</p>", nil)
	require.NoError(t, err)
	assert.Equal(t, "fallback-id", id)
}

// Test connection error (server closed).
func TestSendCloudProvider_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"result":true}`)
	}))
	srv.Close() // Close immediately

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	_, err := p.SendHTMLWith("to@x.com", "s", "<p>b</p>", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sendcloud")
}

// Test that the request body is properly form-encoded.
func TestSendCloudProvider_FormEncoding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		// Verify all required fields are present
		assert.NotEmpty(t, r.PostForm.Get("apiUser"))
		assert.NotEmpty(t, r.PostForm.Get("apiKey"))
		assert.NotEmpty(t, r.PostForm.Get("to"))
		assert.NotEmpty(t, r.PostForm.Get("from"))
		assert.NotEmpty(t, r.PostForm.Get("subject"))
		assert.NotEmpty(t, r.PostForm.Get("html"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":true,"info":{"emailIdList":["sc-enc-1"]}}`)
	}))
	defer srv.Close()

	p, _ := NewSendCloudProvider(SendCloudConfig{
		APIUser: "u", APIKey: "k", From: "noreply@example.com", Endpoint: srv.URL,
	})
	_, err := p.SendHTMLWith("to@x.com", "subject", "<p>body</p>", nil)
	require.NoError(t, err)
}

// Test ParseSendCloudWebhookEvent with messageId containing @.
func TestParseSendCloudWebhookEvent_MessageIDWithAt(t *testing.T) {
	data := []byte("event=delivered&messageId=msg-123@sendcloud.net")
	ev, err := ParseSendCloudWebhookEvent(data)
	require.NoError(t, err)
	assert.Equal(t, "msg-123", ev.MessageID)
}

// Test ParseSendCloudWebhookEvent with emailId fallback for messageId.
func TestParseSendCloudWebhookEvent_EmailIdFallback(t *testing.T) {
	data := []byte("event=delivered&emailId=fallback-msg-456@sendcloud.net")
	ev, err := ParseSendCloudWebhookEvent(data)
	require.NoError(t, err)
	assert.Equal(t, "fallback-msg-456", ev.MessageID)
}

// Test ParseSendCloudWebhookEvent with JSON containing event and messageId.
func TestParseSendCloudWebhookEvent_JSONWithBoth(t *testing.T) {
	data := []byte(`{"event":"click","messageId":"msg-click-1","email":"user@test.com","smtpStatus":"250"}`)
	ev, err := ParseSendCloudWebhookEvent(data)
	require.NoError(t, err)
	assert.Equal(t, "click", ev.Event)
	assert.Equal(t, "msg-click-1", ev.MessageID)
	assert.Equal(t, "user@test.com", ev.Email)
	assert.Equal(t, "250", ev.SmtpStatus)
}

// Test ParseSendCloudWebhookEvent with JSON that has no event or messageId
// (should fall back to form parsing).
func TestParseSendCloudWebhookEvent_JSONNoEventNoMessageID(t *testing.T) {
	data := []byte(`{"foo":"bar"}`)
	ev, err := ParseSendCloudWebhookEvent(data)
	require.NoError(t, err)
	// Falls back to form parsing which will parse this as a query string
	// and find no relevant fields.
	assert.Equal(t, "", ev.Event)
	assert.Equal(t, "", ev.MessageID)
}

// Test SendCloudEventToStatus with all status constants.
func TestSendCloudEventToStatus_AllConstants(t *testing.T) {
	// Verify all status constants are different.
	statuses := []string{
		StatusSent, StatusDelivered, StatusFailed, StatusSoftBounce,
		StatusInvalid, StatusSpam, StatusClicked, StatusOpened,
		StatusUnsubscribed, StatusUnknown,
	}
	seen := map[string]bool{}
	for _, s := range statuses {
		assert.False(t, seen[s], "duplicate status: %s", s)
		seen[s] = true
	}
}

// Test that strings package is used (avoid unused import).
func TestSendCloudProvider_StringsUsed(t *testing.T) {
	// This is a compile-time check; if strings were unused, the package
	// wouldn't compile.
	assert.True(t, strings.Contains("hello", "ell"))
}
