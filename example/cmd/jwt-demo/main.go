// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command jwt-demo demonstrates the ling-base common/jwtutil package.
//
// It shows:
//   - Issuing a token pair (access + refresh)
//   - Verifying an access token
//   - Refreshing tokens
//   - Revoking tokens (logout)
//   - Role-based claims
//
// Usage:
//
//	go run ./cmd/jwt-demo
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/LingByte/ling-base/common/crypto"
	"github.com/LingByte/ling-base/common/jwtutil"
)

func main() {
	fmt.Println("=== JWT Auth Demo ===")

	// ── Create the auth manager ──
	auth, err := jwtutil.New(jwtutil.Config{
		Secret:     []byte("demo-secret-key-32-bytes-long!!!"),
		Algorithm:  crypto.JWTAlgHS256,
		Issuer:     "demo-app",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
		Store:      jwtutil.NewMemoryTokenStore(),
	})
	if err != nil {
		fmt.Printf("Error creating auth: %v\n", err)
		return
	}

	// ── Login: issue a token pair ──
	fmt.Println("\n1. Login as user 'alice' with roles [admin, user]")
	pair, err := auth.Login("alice",
		jwtutil.WithRoles("admin", "user"),
		jwtutil.WithPermissions("read", "write", "delete"),
		jwtutil.WithExtra("company", "Acme Corp"),
	)
	if err != nil {
		fmt.Printf("Login error: %v\n", err)
		return
	}
	fmt.Printf("   Access Token:  %s...\n", truncate(pair.AccessToken, 50))
	fmt.Printf("   Refresh Token: %s...\n", truncate(pair.RefreshToken, 50))
	fmt.Printf("   Expires In:    %d seconds\n", pair.ExpiresIn)
	fmt.Printf("   Token Type:    %s\n", pair.TokenType)

	// ── Verify the access token ──
	fmt.Println("\n2. Verify access token")
	claims, err := auth.Verify(pair.AccessToken)
	if err != nil {
		fmt.Printf("Verify error: %v\n", err)
		return
	}
	fmt.Printf("   Subject:     %s\n", claims.Subject)
	fmt.Printf("   Issuer:      %s\n", claims.Issuer)
	fmt.Printf("   Token Type:  %s\n", claims.TokenType)
	fmt.Printf("   Roles:       %v\n", claims.Roles)
	fmt.Printf("   Permissions: %v\n", claims.Permissions)
	fmt.Printf("   Company:     %s\n", claims.Extra["company"])
	fmt.Printf("   Expires At:  %s\n", time.Unix(claims.ExpiresAt, 0).Format(time.RFC3339))
	fmt.Printf("   Has admin role: %v\n", claims.HasRole("admin"))
	fmt.Printf("   Has delete perm: %v\n", claims.HasPermission("delete"))

	// ── Refresh the token ──
	fmt.Println("\n3. Refresh token")
	newPair, err := auth.Refresh(pair.RefreshToken)
	if err != nil {
		fmt.Printf("Refresh error: %v\n", err)
		return
	}
	fmt.Printf("   New Access Token:  %s...\n", truncate(newPair.AccessToken, 50))
	fmt.Printf("   New Refresh Token: %s...\n", truncate(newPair.RefreshToken, 50))

	// ── Revoke (logout) ──
	fmt.Println("\n4. Revoke access token (logout)")
	if err := auth.Revoke(context.Background(), newPair.AccessToken); err != nil {
		fmt.Printf("Revoke error: %v\n", err)
		return
	}
	fmt.Println("   Token revoked successfully")

	// Verify that the revoked token is rejected.
	_, err = auth.Verify(newPair.AccessToken)
	if err != nil {
		fmt.Printf("   Verify after revoke: %v (expected)\n", err)
	}

	// ── RS256 demo ──
	fmt.Println("\n5. RS256 algorithm demo")
	priv, pub, err := crypto.GenerateRSAKeyPair(2048)
	if err != nil {
		fmt.Printf("RSA key gen error: %v\n", err)
		return
	}
	rsaAuth, err := jwtutil.New(jwtutil.Config{
		Algorithm:  crypto.JWTAlgRS256,
		RSAPriv:    priv,
		RSAPub:     pub,
		Issuer:     "rsa-demo",
		AccessTTL:  1 * time.Hour,
		RefreshTTL: 24 * time.Hour,
	})
	if err != nil {
		fmt.Printf("RSA auth error: %v\n", err)
		return
	}
	rsaPair, err := rsaAuth.Login("bob", jwtutil.WithRoles("user"))
	if err != nil {
		fmt.Printf("RSA login error: %v\n", err)
		return
	}
	rsaClaims, err := rsaAuth.Verify(rsaPair.AccessToken)
	if err != nil {
		fmt.Printf("RSA verify error: %v\n", err)
		return
	}
	fmt.Printf("   Subject: %s, Roles: %v\n", rsaClaims.Subject, rsaClaims.Roles)

	fmt.Println("\n=== Demo complete ===")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
