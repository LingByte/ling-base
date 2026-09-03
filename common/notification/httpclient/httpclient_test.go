// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(prefix string) *Client { return New(prefix) }

func TestPostForm(t *testing.T) {
	var gotMethod, gotBody, gotCT string
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		gotForm, _ = url.ParseQuery(gotBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok-form"))
	}))
	defer srv.Close()

	c := newTestClient("push:")
	form := url.Values{}
	form.Set("a", "1")
	form.Set("b", "two")
	out, err := c.PostForm(context.Background(), srv.URL, form, map[string]string{"X-Test": "1"}, "", "")
	require.NoError(t, err)
	assert.Equal(t, "ok-form", string(out))
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Contains(t, gotCT, "application/x-www-form-urlencoded")
	assert.Equal(t, "1", gotForm.Get("a"))
	assert.Equal(t, "two", gotForm.Get("b"))
}

func TestPostForm_BasicAuth(t *testing.T) {
	var user, pass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ = r.BasicAuth()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := newTestClient("sms:")
	_, err := c.PostForm(context.Background(), srv.URL, url.Values{"x": {"y"}}, nil, "sid", "tok")
	require.NoError(t, err)
	assert.Equal(t, "sid", user)
	assert.Equal(t, "tok", pass)
}

func TestPostForm_EmptyEndpoint(t *testing.T) {
	c := newTestClient("push:")
	_, err := c.PostForm(context.Background(), "", url.Values{}, nil, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "push: empty endpoint")
}

func TestPostForm_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := newTestClient("push:")
	out, err := c.PostForm(context.Background(), srv.URL, url.Values{}, nil, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	assert.Equal(t, "boom", string(out))
}

func TestPostJSON(t *testing.T) {
	var gotCT string
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newTestClient("sms:")
	out, err := c.PostJSON(context.Background(), srv.URL, []byte(`{"k":"v"}`), nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(out))
	assert.Contains(t, gotCT, "application/json")
	assert.Equal(t, `{"k":"v"}`, gotBody)
}

func TestPostJSON_EmptyEndpoint(t *testing.T) {
	c := newTestClient("sms:")
	_, err := c.PostJSON(context.Background(), "", []byte("{}"), nil, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sms: empty endpoint")
}

func TestGetURL(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("get-ok"))
	}))
	defer srv.Close()

	c := newTestClient("push:")
	out, err := c.GetURL(context.Background(), srv.URL, map[string]string{"X-A": "1"}, "", "")
	require.NoError(t, err)
	assert.Equal(t, "get-ok", string(out))
	assert.Equal(t, http.MethodGet, gotMethod)
}

func TestGetURL_EmptyEndpoint(t *testing.T) {
	c := newTestClient("push:")
	_, err := c.GetURL(context.Background(), "", nil, "", "")
	require.Error(t, err)
}

func TestGetURL_BasicAuth(t *testing.T) {
	var user, pass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ = r.BasicAuth()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := newTestClient("sms:")
	_, err := c.GetURL(context.Background(), srv.URL, nil, "u", "p")
	require.NoError(t, err)
	assert.Equal(t, "u", user)
	assert.Equal(t, "p", pass)
}

func TestPostForm_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient("push:")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.PostForm(ctx, srv.URL, url.Values{}, nil, "", "")
	require.Error(t, err)
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "abc", Truncate("abc", 10))
	assert.Equal(t, strings.Repeat("a", 10)+"...", Truncate(strings.Repeat("a", 20), 10))
}

// ──────────────────────────────────────────────
// Raw HTTP helpers coverage
// ──────────────────────────────────────────────

func TestPostFormRaw_EmptyEndpoint(t *testing.T) {
	c := newTestClient("push:")
	_, _, err := c.PostFormRaw(context.Background(), "", nil, nil, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty endpoint")
}

func TestPostJSONRaw_EmptyEndpoint(t *testing.T) {
	c := newTestClient("sms:")
	_, _, err := c.PostJSONRaw(context.Background(), "", nil, nil, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty endpoint")
}

func TestGetURLRaw_EmptyEndpoint(t *testing.T) {
	c := newTestClient("push:")
	_, _, err := c.GetURLRaw(context.Background(), "", nil, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty endpoint")
}

func TestPostFormRaw_BadURL(t *testing.T) {
	c := newTestClient("push:")
	_, _, err := c.PostFormRaw(context.Background(), "http://[::1]:bad", nil, nil, "", "")
	require.Error(t, err)
}

func TestPostJSONRaw_BadURL(t *testing.T) {
	c := newTestClient("sms:")
	_, _, err := c.PostJSONRaw(context.Background(), "http://[::1]:bad", []byte(`{}`), nil, "", "")
	require.Error(t, err)
}

func TestGetURLRaw_BadURL(t *testing.T) {
	c := newTestClient("push:")
	_, _, err := c.GetURLRaw(context.Background(), "http://[::1]:bad", nil, "", "")
	require.Error(t, err)
}

func TestPostFormRaw_CancelledContext(t *testing.T) {
	c := newTestClient("push:")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := c.PostFormRaw(ctx, "http://localhost:9999", nil, nil, "", "")
	require.Error(t, err)
}

func TestPostJSONRaw_CancelledContext(t *testing.T) {
	c := newTestClient("sms:")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := c.PostJSONRaw(ctx, "http://localhost:9999", []byte(`{}`), nil, "", "")
	require.Error(t, err)
}

func TestGetURLRaw_CancelledContext(t *testing.T) {
	c := newTestClient("push:")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := c.GetURLRaw(ctx, "http://localhost:9999", nil, "", "")
	require.Error(t, err)
}

func TestPostFormRaw_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	c := newTestClient("push:")
	_, _, err := c.PostFormRaw(context.Background(), srv.URL, nil, nil, "", "")
	require.Error(t, err)
}

func TestPostJSONRaw_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	c := newTestClient("sms:")
	_, _, err := c.PostJSONRaw(context.Background(), srv.URL, []byte(`{}`), nil, "", "")
	require.Error(t, err)
}

func TestGetURLRaw_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	c := newTestClient("push:")
	_, _, err := c.GetURLRaw(context.Background(), srv.URL, nil, "", "")
	require.Error(t, err)
}

func TestPostFormRaw_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	c := newTestClient("push:")
	status, body, err := c.PostFormRaw(context.Background(), srv.URL, nil, nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "ok", string(body))
}

func TestPostJSONRaw_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	c := newTestClient("sms:")
	status, body, err := c.PostJSONRaw(context.Background(), srv.URL, []byte(`{}`), nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Contains(t, string(body), "ok")
}

func TestGetURLRaw_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "get-ok")
	}))
	defer srv.Close()

	c := newTestClient("push:")
	status, body, err := c.GetURLRaw(context.Background(), srv.URL, nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "get-ok", string(body))
}

func TestIs2xx(t *testing.T) {
	assert.True(t, Is2xx(200))
	assert.True(t, Is2xx(299))
	assert.False(t, Is2xx(199))
	assert.False(t, Is2xx(300))
}

func TestNowUnix(t *testing.T) {
	assert.NotZero(t, NowUnix())
}

func TestTruncateRaw(t *testing.T) {
	assert.Equal(t, "abc", TruncateRaw("abc", 10))
	assert.Equal(t, "ab…", TruncateRaw("abcdef", 2))
}

func TestJSONStringAny(t *testing.T) {
	assert.Equal(t, `{"a":"b"}`, JSONStringAny(map[string]string{"a": "b"}))
}

func TestCtxOrBackground(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, ctx, CtxOrBackground(ctx))
	assert.NotNil(t, CtxOrBackground(nil))
}

func TestNewWithOptions(t *testing.T) {
	h := &http.Client{Timeout: 5 * time.Second}
	c := New("push:", WithHTTPClient(h))
	assert.Equal(t, "push:", c.Prefix)
	assert.Same(t, h, c.HTTPClient)
}

func TestNewWithTimeout(t *testing.T) {
	c := New("sms:", WithTimeout(10 * time.Second))
	require.NotNil(t, c.HTTPClient)
	assert.Equal(t, 10*time.Second, c.HTTPClient.Timeout)
}

func TestEmptyPrefix(t *testing.T) {
	c := New("")
	_, err := c.PostForm(context.Background(), "", nil, nil, "", "")
	require.Error(t, err)
	// No prefix means no leading "push:"/"sms:" and no stray leading space.
	assert.Equal(t, "empty endpoint", err.Error())
}
