// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package push

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

	form := url.Values{}
	form.Set("a", "1")
	form.Set("b", "two")
	out, err := PostForm(context.Background(), srv.URL, form, map[string]string{"X-Test": "1"}, "", "")
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

	_, err := PostForm(context.Background(), srv.URL, url.Values{"x": {"y"}}, nil, "sid", "tok")
	require.NoError(t, err)
	assert.Equal(t, "sid", user)
	assert.Equal(t, "tok", pass)
}

func TestPostForm_EmptyEndpoint(t *testing.T) {
	_, err := PostForm(context.Background(), "", url.Values{}, nil, "", "")
	require.Error(t, err)
}

func TestPostForm_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer srv.Close()

	out, err := PostForm(context.Background(), srv.URL, url.Values{}, nil, "", "")
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

	out, err := PostJSON(context.Background(), srv.URL, []byte(`{"k":"v"}`), nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(out))
	assert.Contains(t, gotCT, "application/json")
	assert.Equal(t, `{"k":"v"}`, gotBody)
}

func TestPostJSON_EmptyEndpoint(t *testing.T) {
	_, err := PostJSON(context.Background(), "", []byte("{}"), nil, "", "")
	require.Error(t, err)
}

func TestGetURL(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("get-ok"))
	}))
	defer srv.Close()

	out, err := GetURL(context.Background(), srv.URL, map[string]string{"X-A": "1"}, "", "")
	require.NoError(t, err)
	assert.Equal(t, "get-ok", string(out))
	assert.Equal(t, http.MethodGet, gotMethod)
}

func TestGetURL_EmptyEndpoint(t *testing.T) {
	_, err := GetURL(context.Background(), "", nil, "", "")
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

	_, err := GetURL(context.Background(), srv.URL, nil, "u", "p")
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

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := PostForm(ctx, srv.URL, url.Values{}, nil, "", "")
	require.Error(t, err)
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "abc", truncate("abc", 10))
	assert.Equal(t, strings.Repeat("a", 10)+"...", truncate(strings.Repeat("a", 20), 10))
}

// ──────────────────────────────────────────────
// HTTP client helpers coverage (PostFormRaw / PostJSONRaw / GetURLRaw)
// ──────────────────────────────────────────────

func TestPostFormRaw_EmptyEndpoint(t *testing.T) {
	_, _, err := PostFormRaw(context.Background(), "", nil, nil, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty endpoint")
}

func TestPostJSONRaw_EmptyEndpoint(t *testing.T) {
	_, _, err := PostJSONRaw(context.Background(), "", nil, nil, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty endpoint")
}

func TestGetURLRaw_EmptyEndpoint(t *testing.T) {
	_, _, err := GetURLRaw(context.Background(), "", nil, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty endpoint")
}

func TestPostFormRaw_BadURL(t *testing.T) {
	_, _, err := PostFormRaw(context.Background(), "http://[::1]:bad", nil, nil, "", "")
	require.Error(t, err)
}

func TestPostJSONRaw_BadURL(t *testing.T) {
	_, _, err := PostJSONRaw(context.Background(), "http://[::1]:bad", []byte(`{}`), nil, "", "")
	require.Error(t, err)
}

func TestGetURLRaw_BadURL(t *testing.T) {
	_, _, err := GetURLRaw(context.Background(), "http://[::1]:bad", nil, "", "")
	require.Error(t, err)
}

func TestPostFormRaw_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := PostFormRaw(ctx, "http://localhost:9999", nil, nil, "", "")
	require.Error(t, err)
}

func TestPostJSONRaw_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := PostJSONRaw(ctx, "http://localhost:9999", []byte(`{}`), nil, "", "")
	require.Error(t, err)
}

func TestGetURLRaw_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := GetURLRaw(ctx, "http://localhost:9999", nil, "", "")
	require.Error(t, err)
}

func TestPostFormRaw_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	_, _, err := PostFormRaw(context.Background(), srv.URL, nil, nil, "", "")
	require.Error(t, err)
}

func TestPostJSONRaw_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	_, _, err := PostJSONRaw(context.Background(), srv.URL, []byte(`{}`), nil, "", "")
	require.Error(t, err)
}

func TestGetURLRaw_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	_, _, err := GetURLRaw(context.Background(), srv.URL, nil, "", "")
	require.Error(t, err)
}

func TestPostFormRaw_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()
	status, body, err := PostFormRaw(context.Background(), srv.URL, nil, nil, "", "")
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
	status, body, err := PostJSONRaw(context.Background(), srv.URL, []byte(`{}`), nil, "", "")
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
	status, body, err := GetURLRaw(context.Background(), srv.URL, nil, "", "")
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

func TestFirstDeviceTokenStr(t *testing.T) {
	s, err := FirstDeviceTokenStr(SendRequest{
		To: []DeviceToken{{Token: "tok"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "tok", s)

	_, err = FirstDeviceTokenStr(SendRequest{})
	require.Error(t, err)
}

func TestErrProviderRejected(t *testing.T) {
	assert.Equal(t, "push: provider rejected", errProviderRejected.Error())
}
