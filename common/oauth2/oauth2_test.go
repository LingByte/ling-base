// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package oauth2

import (
	"context"
	"encoding/json"
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
