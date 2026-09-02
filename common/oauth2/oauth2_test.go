// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_NilProvider(t *testing.T) {
	assert.Panics(t, func() {
		NewClient(nil)
	})
}

func TestAuthURL(t *testing.T) {
	provider := GoogleProvider("cid", "csecret", "https://example.com/cb")
	client := NewClient(provider)

	got := client.AuthURL("xyz123")
	u, err := url.Parse(got)
	require.NoError(t, err)

	assert.Equal(t, "https", u.Scheme)
	assert.Equal(t, "accounts.google.com", u.Host)
	assert.Equal(t, "/o/oauth2/v2/auth", u.Path)

	q := u.Query()
	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, "cid", q.Get("client_id"))
	assert.Equal(t, "https://example.com/cb", q.Get("redirect_uri"))
	assert.Equal(t, "xyz123", q.Get("state"))
	assert.Equal(t, "openid email profile", q.Get("scope"))
}

func TestAuthURLWithParams(t *testing.T) {
	provider := GitHubProvider("cid", "csecret", "https://example.com/cb")
	client := NewClient(provider)

	got := client.AuthURLWithParams("st", map[string]string{
		"login": "octocat",
	})
	u, err := url.Parse(got)
	require.NoError(t, err)

	q := u.Query()
	assert.Equal(t, "st", q.Get("state"))
	assert.Equal(t, "octocat", q.Get("login"))
	// default scope still present
	assert.Equal(t, "read:user user:email", q.Get("scope"))
}

func TestAuthURL_NoScopes(t *testing.T) {
	provider := &Provider{
		Name:        "custom",
		AuthURL:     "https://example.com/auth",
		TokenURL:    "https://example.com/token",
		UserInfoURL: "https://example.com/user",
		RedirectURL: "https://app.example.com/cb",
		ClientID:    "cid",
	}
	client := NewClient(provider)

	got := client.AuthURL("s")
	u, err := url.Parse(got)
	require.NoError(t, err)
	q := u.Query()
	assert.Equal(t, "", q.Get("scope"))
	assert.Equal(t, "code", q.Get("response_type"))
}

func TestGenerateState(t *testing.T) {
	s1 := GenerateState()
	s2 := GenerateState()
	assert.NotEmpty(t, s1)
	assert.Len(t, s1, 32) // 16 bytes hex-encoded
	assert.NotEqual(t, s1, s2)
}

func TestValidateState(t *testing.T) {
	assert.True(t, ValidateState("abc", "abc"))
	assert.False(t, ValidateState("abc", "abd"))
	assert.False(t, ValidateState("", "abc"))
	assert.False(t, ValidateState("abc", ""))
	assert.False(t, ValidateState("", ""))
}

func TestExchange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		require.NoError(t, r.ParseForm())
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.Equal(t, "the-code", r.Form.Get("code"))
		assert.Equal(t, "https://example.com/cb", r.Form.Get("redirect_uri"))
		assert.Equal(t, "cid", r.Form.Get("client_id"))
		assert.Equal(t, "csecret", r.Form.Get("client_secret"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "at-123",
			"refresh_token": "rt-456",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "openid email",
		}))
	}))
	defer srv.Close()

	provider := &Provider{
		Name:         "custom",
		AuthURL:      "https://example.com/auth",
		TokenURL:     srv.URL,
		UserInfoURL:  "https://example.com/user",
		RedirectURL:  "https://example.com/cb",
		ClientID:     "cid",
		ClientSecret: "csecret",
	}
	client := NewClient(provider)

	tok, err := client.Exchange(context.Background(), "the-code")
	require.NoError(t, err)
	require.NotNil(t, tok)

	assert.Equal(t, "at-123", tok.AccessToken)
	assert.Equal(t, "rt-456", tok.RefreshToken)
	assert.Equal(t, "Bearer", tok.TokenType)
	assert.Equal(t, int64(3600), tok.ExpiresIn)
	assert.Equal(t, "openid email", tok.Scope)
	assert.False(t, tok.ExpiresAt.IsZero())
	assert.Equal(t, "at-123", tok.Raw["access_token"])
}

func TestExchange_FormEncoded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		_, _ = w.Write([]byte("access_token=at-form&token_type=bearer&expires_in=900&scope=read"))
	}))
	defer srv.Close()

	provider := &Provider{
		Name:     "custom",
		TokenURL: srv.URL,
	}
	client := NewClient(provider)

	tok, err := client.Exchange(context.Background(), "code")
	require.NoError(t, err)
	assert.Equal(t, "at-form", tok.AccessToken)
	assert.Equal(t, "bearer", tok.TokenType)
	assert.Equal(t, int64(900), tok.ExpiresIn)
	assert.Equal(t, "read", tok.Scope)
}

func TestExchange_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	provider := &Provider{Name: "custom", TokenURL: srv.URL}
	client := NewClient(provider)

	tok, err := client.Exchange(context.Background(), "bad")
	require.Error(t, err)
	assert.Nil(t, tok)
	assert.Contains(t, err.Error(), "status 400")
}

func TestRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, "rt-old", r.Form.Get("refresh_token"))
		assert.Equal(t, "cid", r.Form.Get("client_id"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"access_token": "at-new",
			"token_type":   "Bearer",
			"expires_in":   7200,
		}))
	}))
	defer srv.Close()

	provider := &Provider{Name: "custom", TokenURL: srv.URL, ClientID: "cid", ClientSecret: "cs"}
	client := NewClient(provider)

	tok, err := client.Refresh(context.Background(), "rt-old")
	require.NoError(t, err)
	assert.Equal(t, "at-new", tok.AccessToken)
	assert.Equal(t, int64(7200), tok.ExpiresIn)
}

func TestIsExpired(t *testing.T) {
	client := NewClient(&Provider{Name: "custom"})

	assert.True(t, client.IsExpired(nil))

	past := &Token{ExpiresAt: time.Now().Add(-time.Minute)}
	assert.True(t, client.IsExpired(past))

	future := &Token{ExpiresAt: time.Now().Add(time.Minute)}
	assert.False(t, client.IsExpired(future))

	zero := &Token{}
	assert.False(t, client.IsExpired(zero))
}

func TestGetUserInfo_Google(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer at-123", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":      "12345",
			"name":    "Alice",
			"email":   "alice@example.com",
			"picture": "https://example.com/a.png",
		}))
	}))
	defer srv.Close()

	provider := GoogleProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)

	tok := &Token{AccessToken: "at-123", TokenType: "Bearer"}
	raw, err := client.GetUserInfo(context.Background(), tok)
	require.NoError(t, err)
	assert.Equal(t, "12345", raw["id"])

	user, err := client.GetUser(context.Background(), tok)
	require.NoError(t, err)
	assert.Equal(t, "12345", user.ID)
	assert.Equal(t, "Alice", user.Name)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "https://example.com/a.png", user.Avatar)
	assert.Equal(t, "google", user.Provider)
}

func TestGetUser_GitHub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":           float64(67890),
			"login":        "octocat",
			"name":         "The Octocat",
			"email":        "octo@example.com",
			"avatar_url":   "https://example.com/octo.png",
		}))
	}))
	defer srv.Close()

	provider := GitHubProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)

	tok := &Token{AccessToken: "at", TokenType: "Bearer"}
	user, err := client.GetUser(context.Background(), tok)
	require.NoError(t, err)
	assert.Equal(t, "67890", user.ID)
	assert.Equal(t, "The Octocat", user.Name)
	assert.Equal(t, "octo@example.com", user.Email)
	assert.Equal(t, "https://example.com/octo.png", user.Avatar)
	assert.Equal(t, "github", user.Provider)
}

func TestGetUser_GitHub_NoName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":         float64(1),
			"login":      "someone",
			"avatar_url": "https://example.com/x.png",
		}))
	}))
	defer srv.Close()

	provider := GitHubProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)

	user, err := client.GetUser(context.Background(), &Token{AccessToken: "at", TokenType: "Bearer"})
	require.NoError(t, err)
	assert.Equal(t, "someone", user.Name)
}

func TestGetUser_WeChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// WeChat expects access_token + openid as query params.
		q := r.URL.Query()
		assert.Equal(t, "at-wx", q.Get("access_token"))
		assert.Equal(t, "openid-1", q.Get("openid"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"openid":     "openid-1",
			"nickname":   "Bob",
			"headimgurl": "https://example.com/bob.png",
		}))
	}))
	defer srv.Close()

	provider := WeChatProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)

	tok := &Token{
		AccessToken: "at-wx",
		Raw:         map[string]any{"openid": "openid-1"},
	}
	user, err := client.GetUser(context.Background(), tok)
	require.NoError(t, err)
	assert.Equal(t, "openid-1", user.ID)
	assert.Equal(t, "Bob", user.Name)
	assert.Equal(t, "https://example.com/bob.png", user.Avatar)
	assert.Equal(t, "wechat", user.Provider)
}

func TestGetUser_DingTalk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "at-dt", r.Header.Get("x-acs-dingtalk-access-token"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"unionId":   "union-1",
			"openId":    "open-1",
			"nick":      "Carol",
			"email":     "carol@example.com",
			"avatarUrl": "https://example.com/carol.png",
		}))
	}))
	defer srv.Close()

	provider := DingTalkProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)

	user, err := client.GetUser(context.Background(), &Token{AccessToken: "at-dt"})
	require.NoError(t, err)
	assert.Equal(t, "union-1", user.ID)
	assert.Equal(t, "Carol", user.Name)
	assert.Equal(t, "carol@example.com", user.Email)
	assert.Equal(t, "https://example.com/carol.png", user.Avatar)
	assert.Equal(t, "dingtalk", user.Provider)
}

func TestGetUser_Feishu(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer at-fs", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"open_id":     "open-fs",
			"name":        "Dave",
			"email":       "dave@example.com",
			"avatar_url":  "https://example.com/dave.png",
		}))
	}))
	defer srv.Close()

	provider := FeishuProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)

	user, err := client.GetUser(context.Background(), &Token{AccessToken: "at-fs", TokenType: "Bearer"})
	require.NoError(t, err)
	assert.Equal(t, "open-fs", user.ID)
	assert.Equal(t, "Dave", user.Name)
	assert.Equal(t, "dave@example.com", user.Email)
	assert.Equal(t, "https://example.com/dave.png", user.Avatar)
	assert.Equal(t, "feishu", user.Provider)
}

func TestGetUserInfo_NilToken(t *testing.T) {
	client := NewClient(&Provider{Name: "custom", UserInfoURL: "https://example.com"})
	_, err := client.GetUserInfo(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestGetUser_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_token"}`))
	}))
	defer srv.Close()

	provider := GoogleProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)

	_, err := client.GetUser(context.Background(), &Token{AccessToken: "bad"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 401")
}

func TestPresetProviders(t *testing.T) {
	g := GoogleProvider("g", "gs", "https://app/cb")
	assert.Equal(t, "google", g.Name)
	assert.Equal(t, "https://accounts.google.com/o/oauth2/v2/auth", g.AuthURL)
	assert.Equal(t, "https://oauth2.googleapis.com/token", g.TokenURL)
	assert.Equal(t, "https://www.googleapis.com/oauth2/v2/userinfo", g.UserInfoURL)
	assert.Equal(t, []string{"openid", "email", "profile"}, g.Scopes)

	gh := GitHubProvider("gh", "ghs", "https://app/cb")
	assert.Equal(t, "github", gh.Name)
	assert.Equal(t, "https://github.com/login/oauth/authorize", gh.AuthURL)
	assert.Equal(t, "https://github.com/login/oauth/access_token", gh.TokenURL)
	assert.Equal(t, "https://api.github.com/user", gh.UserInfoURL)

	wx := WeChatProvider("wx", "wxs", "https://app/cb")
	assert.Equal(t, "wechat", wx.Name)
	assert.Equal(t, "https://open.weixin.qq.com/connect/qrconnect", wx.AuthURL)
	assert.Equal(t, "https://api.weixin.qq.com/sns/oauth2/access_token", wx.TokenURL)
	assert.Equal(t, "https://api.weixin.qq.com/sns/userinfo", wx.UserInfoURL)

	dt := DingTalkProvider("dt", "dts", "https://app/cb")
	assert.Equal(t, "dingtalk", dt.Name)
	assert.Equal(t, "https://login.dingtalk.com/oauth2/auth", dt.AuthURL)
	assert.Equal(t, "https://api.dingtalk.com/v1.0/oauth2/userAccessToken", dt.TokenURL)
	assert.Equal(t, "https://api.dingtalk.com/v1.0/contact/users/me", dt.UserInfoURL)

	fs := FeishuProvider("fs", "fss", "https://app/cb")
	assert.Equal(t, "feishu", fs.Name)
	assert.Equal(t, "https://open.feishu.cn/open-apis/authen/v1/index", fs.AuthURL)
	assert.Equal(t, "https://open.feishu.cn/open-apis/authen/v1/access_token", fs.TokenURL)
	assert.Equal(t, "https://open.feishu.cn/open-apis/authen/v1/user_info", fs.UserInfoURL)
}

func TestSetHTTPClient(t *testing.T) {
	client := NewClient(&Provider{Name: "custom"})
	client.SetHTTPClient(nil)
	assert.NotNil(t, client.httpClient)
	assert.Same(t, http.DefaultClient, client.httpClient)
}

func TestAuthURL_ContainsExistingQuery(t *testing.T) {
	provider := &Provider{
		Name:    "custom",
		AuthURL: "https://example.com/auth?foo=bar",
	}
	client := NewClient(provider)
	got := client.AuthURL("s")
	assert.True(t, strings.Contains(got, "foo=bar"))
	assert.True(t, strings.Contains(got, "state=s"))
}

// errBody is an io.ReadCloser that always returns an error on Read.
type errBody struct{}

func (errBody) Read(p []byte) (int, error) { return 0, fmt.Errorf("read boom") }
func (errBody) Close() error               { return nil }

// newClientWithReadErrorTransport returns a Client whose http client uses a
// transport that returns a 200 response whose body fails to read.
func newClientWithReadErrorTransport(t *testing.T, provider *Provider) *Client {
	t.Helper()
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       errBody{},
		}, nil
	})
	hc := &http.Client{Transport: rt}
	c := NewClient(provider)
	c.SetHTTPClient(hc)
	return c
}

// roundTripFunc is an http.RoundTripper backed by a function.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestExchange_BuildRequestError(t *testing.T) {
	provider := &Provider{Name: "custom", TokenURL: "http://exa mple.com/token"}
	client := NewClient(provider)
	tok, err := client.Exchange(context.Background(), "code")
	require.Error(t, err)
	assert.Nil(t, tok)
	assert.Contains(t, err.Error(), "build token request")
}

func TestExchange_RequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider := &Provider{Name: "custom", TokenURL: "https://example.com/token"}
	client := NewClient(provider)
	tok, err := client.Exchange(ctx, "code")
	require.Error(t, err)
	assert.Nil(t, tok)
	assert.Contains(t, err.Error(), "token request")
}

func TestExchange_ReadBodyError(t *testing.T) {
	provider := &Provider{Name: "custom", TokenURL: "https://example.com/token"}
	client := newClientWithReadErrorTransport(t, provider)
	tok, err := client.Exchange(context.Background(), "code")
	require.Error(t, err)
	assert.Nil(t, tok)
	assert.Contains(t, err.Error(), "read token response")
}

func TestRefresh_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server_error"}`))
	}))
	defer srv.Close()

	provider := &Provider{Name: "custom", TokenURL: srv.URL, ClientID: "cid", ClientSecret: "cs"}
	client := NewClient(provider)
	tok, err := client.Refresh(context.Background(), "rt-old")
	require.Error(t, err)
	assert.Nil(t, tok)
	assert.Contains(t, err.Error(), "status 500")
}

func TestRefresh_RequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider := &Provider{Name: "custom", TokenURL: "https://example.com/token", ClientID: "cid"}
	client := NewClient(provider)
	tok, err := client.Refresh(ctx, "rt-old")
	require.Error(t, err)
	assert.Nil(t, tok)
	assert.Contains(t, err.Error(), "token request")
}

func TestParseToken_JSONExpiresInString(t *testing.T) {
	tok, err := parseToken([]byte(`{"access_token":"at","expires_in":"3600"}`))
	require.NoError(t, err)
	assert.Equal(t, "at", tok.AccessToken)
	assert.Equal(t, int64(3600), tok.ExpiresIn)
	assert.False(t, tok.ExpiresAt.IsZero())
}

func TestParseToken_JSONNull(t *testing.T) {
	// "null" unmarshals to a nil map, so the form-encoded fallback runs.
	tok, err := parseToken([]byte("null"))
	require.NoError(t, err)
	assert.Equal(t, "", tok.AccessToken)
	assert.NotNil(t, tok.Raw)
}

func TestParseToken_FormEncodedExpiresInInvalid(t *testing.T) {
	// Invalid JSON -> form-encoded path; expires_in non-numeric so ExpiresAt stays zero.
	tok, err := parseToken([]byte("access_token=at&expires_in=notanumber"))
	require.NoError(t, err)
	assert.Equal(t, "at", tok.AccessToken)
	assert.Equal(t, int64(0), tok.ExpiresIn)
	assert.True(t, tok.ExpiresAt.IsZero())
}

func TestParseToken_BothPathsFail(t *testing.T) {
	// Invalid JSON and invalid query (semicolon) -> error.
	_, err := parseToken([]byte("a=b;c=d"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse token response")
}

func TestGetUserInfo_NoAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "1"}))
	}))
	defer srv.Close()

	provider := &Provider{Name: "custom", UserInfoURL: srv.URL}
	client := NewClient(provider)
	raw, err := client.GetUserInfo(context.Background(), &Token{})
	require.NoError(t, err)
	assert.Equal(t, "1", raw["id"])
}

func TestGetUserInfo_CustomTokenType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Mac at-1", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{}))
	}))
	defer srv.Close()

	provider := &Provider{Name: "custom", UserInfoURL: srv.URL}
	client := NewClient(provider)
	_, err := client.GetUserInfo(context.Background(), &Token{AccessToken: "at-1", TokenType: "Mac"})
	require.NoError(t, err)
}

func TestGetUserInfo_BuildRequestError(t *testing.T) {
	provider := &Provider{Name: "custom", UserInfoURL: "http://exa mple.com/u"}
	client := NewClient(provider)
	_, err := client.GetUserInfo(context.Background(), &Token{AccessToken: "at"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build user-info request")
}

func TestGetUserInfo_RequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider := &Provider{Name: "custom", UserInfoURL: "https://example.com/u"}
	client := NewClient(provider)
	_, err := client.GetUserInfo(ctx, &Token{AccessToken: "at"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user-info request")
}

func TestGetUserInfo_ReadBodyError(t *testing.T) {
	provider := &Provider{Name: "custom", UserInfoURL: "https://example.com/u"}
	client := newClientWithReadErrorTransport(t, provider)
	_, err := client.GetUserInfo(context.Background(), &Token{AccessToken: "at"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read user-info response")
}

func TestGetUserInfo_ParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	provider := &Provider{Name: "custom", UserInfoURL: srv.URL}
	client := NewClient(provider)
	_, err := client.GetUserInfo(context.Background(), &Token{AccessToken: "at"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse user-info response")
}

func TestGetUser_NilToken(t *testing.T) {
	client := NewClient(&Provider{Name: "custom", UserInfoURL: "https://example.com/u"})
	_, err := client.GetUser(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestGetUser_WeChat_NoOpenID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "at-wx", q.Get("access_token"))
		assert.Empty(t, q.Get("openid"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"openid": "x", "nickname": "n"}))
	}))
	defer srv.Close()

	provider := WeChatProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)
	user, err := client.GetUser(context.Background(), &Token{AccessToken: "at-wx", Raw: map[string]any{}})
	require.NoError(t, err)
	assert.Equal(t, "x", user.ID)
}

func TestGetUser_WeChat_InvalidUserInfoURL(t *testing.T) {
	provider := WeChatProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = "http://exa mple.com/u"
	client := NewClient(provider)
	_, err := client.GetUser(context.Background(), &Token{AccessToken: "at-wx", Raw: map[string]any{"openid": "o"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse wechat user-info url")
}

func TestGetUser_WeChat_RequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider := WeChatProvider("cid", "csecret", "https://example.com/cb")
	client := NewClient(provider)
	_, err := client.GetUser(ctx, &Token{AccessToken: "at-wx", Raw: map[string]any{"openid": "o"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wechat user-info request")
}

func TestGetUser_WeChat_ReadBodyError(t *testing.T) {
	provider := WeChatProvider("cid", "csecret", "https://example.com/cb")
	client := newClientWithReadErrorTransport(t, provider)
	_, err := client.GetUser(context.Background(), &Token{AccessToken: "at-wx", Raw: map[string]any{"openid": "o"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read wechat user-info response")
}

func TestGetUser_WeChat_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errcode":40001}`))
	}))
	defer srv.Close()

	provider := WeChatProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)
	_, err := client.GetUser(context.Background(), &Token{AccessToken: "at-wx", Raw: map[string]any{"openid": "o"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 401")
}

func TestGetUser_WeChat_ParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	provider := WeChatProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)
	_, err := client.GetUser(context.Background(), &Token{AccessToken: "at-wx", Raw: map[string]any{"openid": "o"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse wechat user-info response")
}

func TestGetUser_DingTalk_RequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider := DingTalkProvider("cid", "csecret", "https://example.com/cb")
	client := NewClient(provider)
	_, err := client.GetUser(ctx, &Token{AccessToken: "at-dt"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dingtalk user-info request")
}

func TestGetUser_DingTalk_ReadBodyError(t *testing.T) {
	provider := DingTalkProvider("cid", "csecret", "https://example.com/cb")
	client := newClientWithReadErrorTransport(t, provider)
	_, err := client.GetUser(context.Background(), &Token{AccessToken: "at-dt"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read dingtalk user-info response")
}

func TestGetUser_DingTalk_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`forbidden`))
	}))
	defer srv.Close()

	provider := DingTalkProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)
	_, err := client.GetUser(context.Background(), &Token{AccessToken: "at-dt"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 403")
}

func TestGetUser_DingTalk_ParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	provider := DingTalkProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)
	_, err := client.GetUser(context.Background(), &Token{AccessToken: "at-dt"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse dingtalk user-info response")
}

func TestGetUser_Feishu_RequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider := FeishuProvider("cid", "csecret", "https://example.com/cb")
	client := NewClient(provider)
	_, err := client.GetUser(ctx, &Token{AccessToken: "at-fs"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "feishu user-info request")
}

func TestGetUser_Feishu_ReadBodyError(t *testing.T) {
	provider := FeishuProvider("cid", "csecret", "https://example.com/cb")
	client := newClientWithReadErrorTransport(t, provider)
	_, err := client.GetUser(context.Background(), &Token{AccessToken: "at-fs"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read feishu user-info response")
}

func TestGetUser_Feishu_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`unauthorized`))
	}))
	defer srv.Close()

	provider := FeishuProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)
	_, err := client.GetUser(context.Background(), &Token{AccessToken: "at-fs"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 401")
}

func TestGetUser_Feishu_ParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	provider := FeishuProvider("cid", "csecret", "https://example.com/cb")
	provider.UserInfoURL = srv.URL
	client := NewClient(provider)
	_, err := client.GetUser(context.Background(), &Token{AccessToken: "at-fs"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse feishu user-info response")
}

func TestMapUserInfo_Default(t *testing.T) {
	client := NewClient(&Provider{Name: "custom"})
	ui := client.mapUserInfo(map[string]any{
		"id":     "9",
		"name":   "Eve",
		"email":  "eve@example.com",
		"avatar": "https://example.com/eve.png",
	})
	assert.Equal(t, "9", ui.ID)
	assert.Equal(t, "Eve", ui.Name)
	assert.Equal(t, "eve@example.com", ui.Email)
	assert.Equal(t, "https://example.com/eve.png", ui.Avatar)
	assert.Equal(t, "custom", ui.Provider)
}

func TestMapUserInfo_DingTalk_OpenIDFallback(t *testing.T) {
	client := NewClient(DingTalkProvider("cid", "cs", "https://example.com/cb"))
	ui := client.mapUserInfo(map[string]any{
		"openId":    "open-2",
		"nick":      "N",
		"avatarUrl": "https://example.com/n.png",
	})
	assert.Equal(t, "open-2", ui.ID)
	assert.Equal(t, "N", ui.Name)
	assert.Equal(t, "https://example.com/n.png", ui.Avatar)
}

func TestMapUserInfo_Feishu_UserIDFallback(t *testing.T) {
	client := NewClient(FeishuProvider("cid", "cs", "https://example.com/cb"))
	ui := client.mapUserInfo(map[string]any{
		"user_id": "u-3",
		"name":    "F",
		"avatar":  "https://example.com/f.png",
	})
	assert.Equal(t, "u-3", ui.ID)
	assert.Equal(t, "F", ui.Name)
	assert.Equal(t, "https://example.com/f.png", ui.Avatar)
}

func TestMapUserInfo_NilRaw(t *testing.T) {
	client := NewClient(&Provider{Name: "custom"})
	ui := client.mapUserInfo(nil)
	assert.Equal(t, "custom", ui.Provider)
	assert.Empty(t, ui.ID)
}

func TestAsString_NonString(t *testing.T) {
	assert.Equal(t, "", asString(123))
	assert.Equal(t, "", asString(nil))
	assert.Equal(t, "x", asString("x"))
}

func TestAsInt64(t *testing.T) {
	assert.Equal(t, int64(42), asInt64(float64(42)))
	assert.Equal(t, int64(7), asInt64(int64(7)))
	assert.Equal(t, int64(3), asInt64(float32(3)))
	assert.Equal(t, int64(0), asInt64("not a number"))
	assert.Equal(t, int64(0), asInt64(nil))
}
