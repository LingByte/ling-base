// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package authcontext_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/ling-base/common/authcontext"
)

// Identity is a convenient alias for test literals.
type Identity = authcontext.Identity

func TestIdentity_HasRole(t *testing.T) {
	id := &Identity{Roles: []string{"admin", "editor"}}
	if !id.HasRole("admin") {
		t.Error("HasRole(admin) = false")
	}
	if id.HasRole("viewer") {
		t.Error("HasRole(viewer) = true")
	}
	if (&Identity{}).HasRole("admin") {
		t.Error("nil identity HasRole should be false")
	}
}

func TestIdentity_HasPermission_Wildcard(t *testing.T) {
	id := &Identity{Permissions: []string{"user:*", "system:read"}}
	cases := []struct {
		perm string
		want bool
	}{
		{"user:read", true},
		{"user:write", true},
		{"user:delete", true},
		{"system:read", true},
		{"system:write", false},
		{"admin:all", false},
	}
	for _, c := range cases {
		if got := id.HasPermission(c.perm); got != c.want {
			t.Errorf("HasPermission(%q) = %v, want %v", c.perm, got, c.want)
		}
	}
}

func TestIdentity_HasPermission_Star(t *testing.T) {
	id := &Identity{Permissions: []string{"*"}}
	if !id.HasPermission("anything") {
		t.Error("HasPermission with '*' should match anything")
	}
}

func TestIdentity_IsAuthenticated(t *testing.T) {
	if (&Identity{}).IsAuthenticated() {
		t.Error("empty identity should not be authenticated")
	}
	id := &Identity{UserID: "user-1"}
	if !id.IsAuthenticated() {
		t.Error("identity with UserID should be authenticated")
	}
}

func TestWithIdentity_FromContext(t *testing.T) {
	ctx := context.Background()
	if id := authcontext.FromContext(ctx); id != nil {
		t.Error("FromContext on empty ctx should return nil")
	}
	id := &Identity{UserID: "u1"}
	ctx = authcontext.WithIdentity(ctx, id)
	got := authcontext.FromContext(ctx)
	if got == nil || got.UserID != "u1" {
		t.Errorf("FromContext = %v, want UserID=u1", got)
	}
}

func TestMustFromContext_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustFromContext should panic when no identity")
		}
	}()
	authcontext.MustFromContext(context.Background())
}

func TestHeaderExtractor(t *testing.T) {
	extractor := authcontext.NewHeaderExtractor()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-User-ID", "user-123")
	req.Header.Set("X-User-Name", "alice")
	req.Header.Set("X-User-Roles", "admin,editor")
	req.Header.Set("X-User-Permissions", "user:read,user:write")

	id, err := extractor.Extract(req)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if id == nil {
		t.Fatal("identity is nil")
	}
	if id.UserID != "user-123" {
		t.Errorf("UserID = %q", id.UserID)
	}
	if id.UserName != "alice" {
		t.Errorf("UserName = %q", id.UserName)
	}
	if len(id.Roles) != 2 || id.Roles[0] != "admin" {
		t.Errorf("Roles = %v", id.Roles)
	}
	if id.Source != "header" {
		t.Errorf("Source = %q", id.Source)
	}
}

func TestHeaderExtractor_NoHeaders(t *testing.T) {
	extractor := authcontext.NewHeaderExtractor()
	req := httptest.NewRequest("GET", "/", nil)
	id, err := extractor.Extract(req)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if id != nil {
		t.Error("identity should be nil when no headers")
	}
}

func TestBearerFromHeader(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"Bearer abc123", "abc123"},
		{"bearer abc123", "abc123"},
		{"BEARER abc123", "abc123"},
		{"Basic abc123", ""},
		{"", ""},
		{"Bearer", ""},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", "/", nil)
		if c.header != "" {
			req.Header.Set("Authorization", c.header)
		}
		got := authcontext.BearerFromHeader(req)
		if got != c.want {
			t.Errorf("BearerFromHeader(%q) = %q, want %q", c.header, got, c.want)
		}
	}
}

// stubVerifier implements authcontext.JWTVerifier for testing.
type stubVerifier struct {
	claims *authcontext.JWTClaims
	err    error
}

func (s *stubVerifier) Verify(token string) (*authcontext.JWTClaims, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.claims, nil
}

func TestJWTExtractor(t *testing.T) {
	verifier := &stubVerifier{
		claims: &authcontext.JWTClaims{
			Subject: "user-456",
			Extra: map[string]any{
				"user_name":    "bob",
				"nick_name":    "Bobby",
				"roles":        []any{"admin", "viewer"},
				"permissions":  []any{"user:*", "system:read"},
			},
		},
	}
	extractor := authcontext.NewJWTExtractor(verifier)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	id, err := extractor.Extract(req)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if id.UserID != "user-456" {
		t.Errorf("UserID = %q", id.UserID)
	}
	if id.UserName != "bob" {
		t.Errorf("UserName = %q", id.UserName)
	}
	if len(id.Roles) != 2 || id.Roles[0] != "admin" {
		t.Errorf("Roles = %v", id.Roles)
	}
	if id.Token != "test-token" {
		t.Errorf("Token = %q", id.Token)
	}
	if id.Source != "jwt" {
		t.Errorf("Source = %q", id.Source)
	}
}

func TestJWTExtractor_InvalidToken(t *testing.T) {
	verifier := &stubVerifier{err: errors.New("bad signature")}
	extractor := authcontext.NewJWTExtractor(verifier)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer bad-token")

	_, err := extractor.Extract(req)
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestJWTExtractor_NoToken(t *testing.T) {
	verifier := &stubVerifier{}
	extractor := authcontext.NewJWTExtractor(verifier)
	req := httptest.NewRequest("GET", "/", nil)
	id, err := extractor.Extract(req)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if id != nil {
		t.Error("identity should be nil when no token")
	}
}

func TestPropagatingTransport(t *testing.T) {
	// Use a test server to capture forwarded headers.
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(200)
	}))
	defer server.Close()

	id := &Identity{
		UserID:      "u-789",
		UserName:    "carol",
		Roles:       []string{"admin"},
		Permissions: []string{"user:read"},
		Token:       "forwarded-token",
	}
	ctx := authcontext.WithIdentity(context.Background(), id)

	client := &http.Client{
		Transport: authcontext.NewPropagatingTransport(nil),
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	if capturedHeaders.Get("X-User-ID") != "u-789" {
		t.Errorf("X-User-ID = %q", capturedHeaders.Get("X-User-ID"))
	}
	if capturedHeaders.Get("X-User-Name") != "carol" {
		t.Errorf("X-User-Name = %q", capturedHeaders.Get("X-User-Name"))
	}
	if capturedHeaders.Get("X-User-Roles") != "admin" {
		t.Errorf("X-User-Roles = %q", capturedHeaders.Get("X-User-Roles"))
	}
	if capturedHeaders.Get("X-Forwarded-Token") != "forwarded-token" {
		t.Errorf("X-Forwarded-Token = %q", capturedHeaders.Get("X-Forwarded-Token"))
	}
}

func TestPropagatingTransport_NoIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-User-ID") != "" {
			t.Error("X-User-ID should not be set when no identity")
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: authcontext.NewPropagatingTransport(nil),
	}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
}

func TestInjectHeaders(t *testing.T) {
	id := &Identity{UserID: "u1", Roles: []string{"admin"}}
	req := httptest.NewRequest("GET", "/", nil)
	authcontext.InjectHeaders(req, id, authcontext.DefaultPropagationHeaders())
	if req.Header.Get("X-User-ID") != "u1" {
		t.Errorf("X-User-ID = %q", req.Header.Get("X-User-ID"))
	}
	if req.Header.Get("X-User-Roles") != "admin" {
		t.Errorf("X-User-Roles = %q", req.Header.Get("X-User-Roles"))
	}
}

func TestMarshalUnmarshalIdentity(t *testing.T) {
	id := &Identity{
		UserID:      "u1",
		UserName:    "alice",
		Roles:       []string{"admin"},
		Permissions: []string{"user:read"},
		Source:      "jwt",
	}
	data, err := authcontext.MarshalIdentity(id)
	if err != nil {
		t.Fatalf("MarshalIdentity: %v", err)
	}
	got, err := authcontext.UnmarshalIdentity(data)
	if err != nil {
		t.Fatalf("UnmarshalIdentity: %v", err)
	}
	if got.UserID != "u1" || got.UserName != "alice" {
		t.Errorf("got = %+v", got)
	}
	if len(got.Roles) != 1 || got.Roles[0] != "admin" {
		t.Errorf("Roles = %v", got.Roles)
	}
	// Token has json:"-" so it should not be serialized.
	_ = json.Unmarshal(data, &struct{}{})
}
