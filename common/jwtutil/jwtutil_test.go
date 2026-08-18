// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package jwtutil

import (
	"context"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/common/crypto"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, crypto.JWTAlgHS256, cfg.Algorithm)
	assert.Equal(t, 15*time.Minute, cfg.AccessTTL)
	assert.Equal(t, 168*time.Hour, cfg.RefreshTTL)
}

func TestNew_HS256(t *testing.T) {
	auth, err := New(Config{
		Secret:     []byte("test-secret-key-1234567890123456"),
		Algorithm:  crypto.JWTAlgHS256,
		Issuer:     "test-issuer",
		AccessTTL:  5 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	})
	require.NoError(t, err)
	require.NotNil(t, auth)
}

func TestNew_HS256_NoSecret(t *testing.T) {
	_, err := New(Config{
		Algorithm: crypto.JWTAlgHS256,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Secret is required")
}

func TestNew_UnsupportedAlgorithm(t *testing.T) {
	_, err := New(Config{
		Algorithm: crypto.JWTAlgorithm("ES256"),
		Secret:    []byte("key"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported algorithm")
}

func TestNew_RS256(t *testing.T) {
	priv, pub := generateTestRSAKey(t)
	auth, err := New(Config{
		Algorithm: crypto.JWTAlgRS256,
		RSAPriv:   priv,
		RSAPub:    pub,
		Issuer:    "test-issuer",
	})
	require.NoError(t, err)
	require.NotNil(t, auth)

	pair, err := auth.Login("user-1")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
	assert.Equal(t, int64(900), pair.ExpiresIn) // 15 min default

	claims, err := auth.Verify(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.Subject)
	assert.True(t, claims.IsAccess())
}

func TestNew_RS256_NoPrivateKey(t *testing.T) {
	_, err := New(Config{
		Algorithm: crypto.JWTAlgRS256,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RSAPriv is required")
}

func TestLogin(t *testing.T) {
	auth := newTestAuth(t)
	pair, err := auth.Login("user-123",
		WithRoles("admin", "user"),
		WithPermissions("read", "write"),
		WithExtra("company", "acme"),
	)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
	assert.Positive(t, pair.ExpiresIn)
}

func TestLogin_EmptySubject(t *testing.T) {
	auth := newTestAuth(t)
	_, err := auth.Login("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subject is required")
}

func TestVerify(t *testing.T) {
	auth := newTestAuth(t)
	pair, err := auth.Login("user-1", WithRoles("admin"))
	require.NoError(t, err)

	claims, err := auth.Verify(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.Subject)
	assert.True(t, claims.IsAccess())
	assert.Contains(t, claims.Roles, "admin")
	assert.NotEmpty(t, claims.ID)
}

func TestVerify_InvalidToken(t *testing.T) {
	auth := newTestAuth(t)
	_, err := auth.Verify("invalid.token.here")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestVerify_WrongTokenType(t *testing.T) {
	auth := newTestAuth(t)
	pair, err := auth.Login("user-1")
	require.NoError(t, err)

	// Try to verify refresh token as access token.
	_, err = auth.Verify(pair.RefreshToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWrongTokenType)
}

func TestVerifyRefresh(t *testing.T) {
	auth := newTestAuth(t)
	pair, err := auth.Login("user-1")
	require.NoError(t, err)

	claims, err := auth.VerifyRefresh(pair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.Subject)
	assert.True(t, claims.IsRefresh())
}

func TestVerifyRefresh_WrongTokenType(t *testing.T) {
	auth := newTestAuth(t)
	pair, err := auth.Login("user-1")
	require.NoError(t, err)

	_, err = auth.VerifyRefresh(pair.AccessToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWrongTokenType)
}

func TestRefresh(t *testing.T) {
	auth := newTestAuth(t)
	pair, err := auth.Login("user-1", WithRoles("admin"))
	require.NoError(t, err)

	newPair, err := auth.Refresh(pair.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newPair.AccessToken)
	assert.NotEmpty(t, newPair.RefreshToken)

	// New tokens should be different from old ones.
	assert.NotEqual(t, pair.AccessToken, newPair.AccessToken)
	assert.NotEqual(t, pair.RefreshToken, newPair.RefreshToken)

	// New access token should be valid.
	claims, err := auth.Verify(newPair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.Subject)
	assert.Contains(t, claims.Roles, "admin")
}

func TestRefresh_WithStore_ReplayPrevention(t *testing.T) {
	store := NewMemoryTokenStore()
	auth := newTestAuthWithStore(t, store)
	pair, err := auth.Login("user-1")
	require.NoError(t, err)

	// First refresh should succeed.
	newPair, err := auth.Refresh(pair.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newPair.AccessToken)

	// Second refresh with the same token should fail (replay).
	_, err = auth.Refresh(pair.RefreshToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTokenRevoked)
}

func TestRevoke(t *testing.T) {
	store := NewMemoryTokenStore()
	auth := newTestAuthWithStore(t, store)

	pair, err := auth.Login("user-1")
	require.NoError(t, err)

	// Verify works before revocation.
	_, err = auth.Verify(pair.AccessToken)
	require.NoError(t, err)

	// Revoke.
	err = auth.Revoke(context.Background(), pair.AccessToken)
	require.NoError(t, err)

	// Verify should now fail.
	_, err = auth.Verify(pair.AccessToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTokenRevoked)
}

func TestRevoke_NoStore(t *testing.T) {
	auth := newTestAuth(t)
	pair, err := auth.Login("user-1")
	require.NoError(t, err)

	err = auth.Revoke(context.Background(), pair.AccessToken)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no token store")
}

func TestRevokeRefresh(t *testing.T) {
	store := NewMemoryTokenStore()
	auth := newTestAuthWithStore(t, store)

	pair, err := auth.Login("user-1")
	require.NoError(t, err)

	err = auth.RevokeRefresh(context.Background(), pair.RefreshToken)
	require.NoError(t, err)

	// Refresh should fail.
	_, err = auth.Refresh(pair.RefreshToken)
	require.Error(t, err)
}

func TestIssueAccess(t *testing.T) {
	auth := newTestAuth(t)
	token, err := auth.IssueAccess("user-1", WithRoles("admin"))
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := auth.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.Subject)
	assert.Contains(t, claims.Roles, "admin")
}

func TestClaims_HasRole(t *testing.T) {
	c := &Claims{Roles: []string{"admin", "user"}}
	assert.True(t, c.HasRole("admin"))
	assert.True(t, c.HasRole("user"))
	assert.False(t, c.HasRole("superadmin"))
}

func TestClaims_HasPermission(t *testing.T) {
	c := &Claims{Permissions: []string{"read", "write"}}
	assert.True(t, c.HasPermission("read"))
	assert.False(t, c.HasPermission("delete"))
}

func TestClaims_IsAccess_IsRefresh(t *testing.T) {
	c := &Claims{TokenType: TokenTypeAccess}
	assert.True(t, c.IsAccess())
	assert.False(t, c.IsRefresh())

	c = &Claims{TokenType: TokenTypeRefresh}
	assert.False(t, c.IsAccess())
	assert.True(t, c.IsRefresh())
}

func TestExtractBearer(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		wantErr bool
		want    string
	}{
		{"valid", "Bearer abc123", false, "abc123"},
		{"empty", "", true, ""},
		{"wrong prefix", "Token abc123", true, ""},
		{"empty token", "Bearer ", true, ""},
		{"no space", "Bearer", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ExtractBearer(tt.header)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, token)
			}
		})
	}
}

func TestContextWithClaims_ClaimsFromContext(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, ClaimsFromContext(ctx))

	claims := &Claims{Subject: "user-1"}
	ctx = ContextWithClaims(ctx, claims)
	got := ClaimsFromContext(ctx)
	require.NotNil(t, got)
	assert.Equal(t, "user-1", got.Subject)
}

func TestMemoryTokenStore(t *testing.T) {
	store := NewMemoryTokenStore()
	ctx := context.Background()

	// Initially not revoked.
	revoked, err := store.IsRevoked(ctx, "jti-1")
	require.NoError(t, err)
	assert.False(t, revoked)

	// Revoke.
	err = store.Revoke(ctx, "jti-1", time.Now().Add(time.Hour))
	require.NoError(t, err)

	// Now revoked.
	revoked, err = store.IsRevoked(ctx, "jti-1")
	require.NoError(t, err)
	assert.True(t, revoked)

	// Different jti not revoked.
	revoked, err = store.IsRevoked(ctx, "jti-2")
	require.NoError(t, err)
	assert.False(t, revoked)
}

func TestMemoryTokenStore_ExpiredRevocation(t *testing.T) {
	store := NewMemoryTokenStore()
	ctx := context.Background()

	// Revoke with past expiry.
	err := store.Revoke(ctx, "jti-1", time.Now().Add(-time.Hour))
	require.NoError(t, err)

	// Should not be considered revoked (revocation expired).
	revoked, err := store.IsRevoked(ctx, "jti-1")
	require.NoError(t, err)
	assert.False(t, revoked)
}

func TestMemoryTokenStore_MarkUsed(t *testing.T) {
	store := NewMemoryTokenStore()
	ctx := context.Background()

	// First use should succeed.
	fresh, err := store.MarkUsed(ctx, "jti-1", time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.True(t, fresh)

	// Second use should fail (already used).
	fresh, err = store.MarkUsed(ctx, "jti-1", time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, fresh)
}

func TestMemoryTokenStore_Cleanup(t *testing.T) {
	store := NewMemoryTokenStore()
	ctx := context.Background()

	// Add expired and non-expired entries.
	_ = store.Revoke(ctx, "expired", time.Now().Add(-time.Hour))
	_ = store.Revoke(ctx, "active", time.Now().Add(time.Hour))
	_, _ = store.MarkUsed(ctx, "expired-used", time.Now().Add(-time.Hour))
	_, _ = store.MarkUsed(ctx, "active-used", time.Now().Add(time.Hour))

	store.Cleanup()

	revoked, used := store.Len()
	assert.Equal(t, 1, revoked) // only "active" remains
	assert.Equal(t, 1, used)    // only "active-used" remains
}

func TestCounterIDGenerator(t *testing.T) {
	gen := NewCounterIDGenerator()
	id1, err := gen.NewID()
	require.NoError(t, err)
	assert.Equal(t, "jti-1", id1)

	id2, err := gen.NewID()
	require.NoError(t, err)
	assert.Equal(t, "jti-2", id2)
}

func TestSetStore(t *testing.T) {
	auth := newTestAuth(t)
	store := NewMemoryTokenStore()
	auth.SetStore(store)

	pair, err := auth.Login("user-1")
	require.NoError(t, err)

	// Revoke should work now.
	err = auth.Revoke(context.Background(), pair.AccessToken)
	require.NoError(t, err)

	_, err = auth.Verify(pair.AccessToken)
	assert.ErrorIs(t, err, ErrTokenRevoked)
}

func TestSetIDGenerator(t *testing.T) {
	auth := newTestAuth(t)
	auth.SetIDGenerator(NewCounterIDGenerator())

	pair, err := auth.Login("user-1")
	require.NoError(t, err)

	claims, err := auth.Verify(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "jti-1", claims.ID)
}

func TestTokenPair_JSON(t *testing.T) {
	pair := &TokenPair{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresIn:    900,
		TokenType:    "Bearer",
	}
	// Just verify the fields are accessible.
	assert.Equal(t, "access", pair.AccessToken)
	assert.Equal(t, "refresh", pair.RefreshToken)
	assert.Equal(t, int64(900), pair.ExpiresIn)
	assert.Equal(t, "Bearer", pair.TokenType)
}

func TestVerify_ExpiredToken(t *testing.T) {
	// Use a signer with very short TTL, then sleep to let it expire.
	auth, err := New(Config{
		Secret:     []byte("test-secret-key-1234567890123456"),
		AccessTTL:  1 * time.Second,
		RefreshTTL: 24 * time.Hour,
	})
	require.NoError(t, err)

	pair, err := auth.Login("user-1")
	require.NoError(t, err)

	// Wait for the token to expire.
	time.Sleep(2 * time.Second)

	_, err = auth.Verify(pair.AccessToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func newTestAuth(t *testing.T) *Auth {
	t.Helper()
	auth, err := New(Config{
		Secret:     []byte("test-secret-key-1234567890123456"),
		Algorithm:  crypto.JWTAlgHS256,
		Issuer:     "test-issuer",
		AccessTTL:  5 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	})
	require.NoError(t, err)
	return auth
}

func newTestAuthWithStore(t *testing.T, store TokenStore) *Auth {
	t.Helper()
	auth, err := New(Config{
		Secret:     []byte("test-secret-key-1234567890123456"),
		Algorithm:  crypto.JWTAlgHS256,
		Issuer:     "test-issuer",
		AccessTTL:  5 * time.Minute,
		RefreshTTL: 24 * time.Hour,
		Store:      store,
	})
	require.NoError(t, err)
	return auth
}

func generateTestRSAKey(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, pub, err := crypto.GenerateRSAKeyPair(2048)
	require.NoError(t, err)
	return priv, pub
}
