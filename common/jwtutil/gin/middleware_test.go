// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/jwtutil"
	gin "github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestAuth(t *testing.T) *jwtutil.Auth {
	t.Helper()
	auth, err := jwtutil.New(jwtutil.Config{
		Secret:     []byte("test-secret-key-32-bytes-long!!!"),
		Issuer:     "test",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("jwtutil.New: %v", err)
	}
	return auth
}

func TestMiddleware_PublicPath(t *testing.T) {
	auth := newTestAuth(t)
	r := gin.New()
	r.Use(Middleware(auth, WithPublicPaths("/health", "/api/v1/auth/login")))
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("public path should pass, got %d", w.Code)
	}
}

func TestMiddleware_NoToken(t *testing.T) {
	auth := newTestAuth(t)
	r := gin.New()
	r.Use(Middleware(auth))
	r.GET("/api/v1/me", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}
}

func TestMiddleware_ValidToken(t *testing.T) {
	auth := newTestAuth(t)
	r := gin.New()
	r.Use(Middleware(auth))
	r.GET("/api/v1/me", func(c *gin.Context) {
		claims := ClaimsFromGin(c)
		if claims == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no claims"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": claims.Subject})
	})

	pair, err := auth.Login("user-123", jwtutil.WithRoles("admin"))
	if err != nil {
		t.Fatalf("auth.Login: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	auth := newTestAuth(t)
	r := gin.New()
	r.Use(Middleware(auth))
	r.GET("/api/v1/me", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got %d", w.Code)
	}
}

func TestRequireRole(t *testing.T) {
	auth := newTestAuth(t)
	r := gin.New()
	r.Use(Middleware(auth))
	r.GET("/admin", RequireRole(auth, "admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// 有 admin 角色
	pair, _ := auth.Login("user-1", jwtutil.WithRoles("admin"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin should pass, got %d", w.Code)
	}

	// 无 admin 角色
	pair2, _ := auth.Login("user-2", jwtutil.WithRoles("user"))
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req2.Header.Set("Authorization", "Bearer "+pair2.AccessToken)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("non-admin should get 403, got %d", w2.Code)
	}
}

func TestRequirePermission(t *testing.T) {
	auth := newTestAuth(t)
	r := gin.New()
	r.Use(Middleware(auth))
	r.GET("/reports", RequirePermission(auth, "reports:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	pair, _ := auth.Login("user-1", jwtutil.WithPermissions("reports:read"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reports", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("should pass with permission, got %d", w.Code)
	}
}

func TestUserIDFromGin(t *testing.T) {
	auth := newTestAuth(t)
	r := gin.New()
	r.Use(Middleware(auth))
	r.GET("/me", func(c *gin.Context) {
		uid := UserIDFromGin(c)
		c.JSON(http.StatusOK, gin.H{"user_id": uid})
	})

	pair, _ := auth.Login("user-456")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), "user-456") {
		t.Errorf("expected user_id=user-456 in body, got %s", w.Body.String())
	}
}

func TestCustomErrorHandler(t *testing.T) {
	auth := newTestAuth(t)
	r := gin.New()
	r.Use(Middleware(auth, WithErrorHandler(func(c *gin.Context, code int, msg string) {
		c.JSON(code, gin.H{"custom_error": msg})
		c.Abort()
	})))
	r.GET("/api/v1/me", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	r.ServeHTTP(w, req)
	if !contains(w.Body.String(), "custom_error") {
		t.Errorf("expected custom_error in body, got %s", w.Body.String())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
