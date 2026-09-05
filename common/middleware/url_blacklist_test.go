// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestURLBlacklist_Exact(t *testing.T) {
	bl := NewURLBlacklist()
	bl.Add("/admin/secret")
	if !bl.IsBlocked("/admin/secret") {
		t.Error("exact match should block")
	}
	if bl.IsBlocked("/admin/secret/extra") {
		t.Error("exact match should not block sub-paths")
	}
	if bl.IsBlocked("/admin") {
		t.Error("exact match should not block parent")
	}
}

func TestURLBlacklist_Wildcard(t *testing.T) {
	bl := NewURLBlacklist()
	bl.Add("/api/*")
	if !bl.IsBlocked("/api/users") {
		t.Error("wildcard should block /api/users")
	}
	if !bl.IsBlocked("/api/users/123") {
		t.Error("wildcard should block /api/users/123")
	}
	if !bl.IsBlocked("/api") {
		t.Error("wildcard /api/* should also block /api itself")
	}
	if bl.IsBlocked("/api2") {
		t.Error("wildcard should not block /api2")
	}
}

func TestURLBlacklist_Prefix(t *testing.T) {
	bl := NewURLBlacklist()
	bl.Add("/private/")
	if !bl.IsBlocked("/private/x") {
		t.Error("prefix should block /private/x")
	}
	if !bl.IsBlocked("/private/a/b") {
		t.Error("prefix should block /private/a/b")
	}
	if bl.IsBlocked("/private") {
		t.Error("prefix with trailing / should NOT block bare /private")
	}
}

func TestURLBlacklist_Remove(t *testing.T) {
	bl := NewURLBlacklist()
	bl.Add("/secret")
	bl.Add("/admin/*")
	if bl.Count() != 2 {
		t.Errorf("count = %d, want 2", bl.Count())
	}
	bl.Remove("/secret")
	if bl.IsBlocked("/secret") {
		t.Error("removed pattern should not block")
	}
	if !bl.IsBlocked("/admin/x") {
		t.Error("non-removed pattern should still block")
	}
}

func TestURLBlacklist_CleanPath(t *testing.T) {
	bl := NewURLBlacklist()
	bl.Add("/admin/secret")
	// Trailing slash should be cleaned.
	if !bl.IsBlocked("/admin/secret/") {
		t.Error("trailing slash should still match exact pattern")
	}
	// Dot segments should be resolved.
	if !bl.IsBlocked("/admin/../admin/secret") {
		t.Error("dot segments should be resolved before matching")
	}
}

func TestURLBlacklist_Empty(t *testing.T) {
	bl := NewURLBlacklist()
	if bl.IsBlocked("/anything") {
		t.Error("empty blacklist should not block")
	}
	if bl.IsBlocked("") {
		t.Error("empty path should not block")
	}
}

func TestURLBlacklist_AddAll(t *testing.T) {
	bl := NewURLBlacklist()
	bl.AddAll("/a", "/b/*", "/c/")
	if bl.Count() != 3 {
		t.Errorf("count = %d, want 3", bl.Count())
	}
}

func TestURLBlacklistMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bl := NewURLBlacklist()
	bl.Add("/blocked")

	r := gin.New()
	r.Use(URLBlacklistMiddleware(bl))
	r.GET("/ok", func(c *gin.Context) { c.String(200, "ok") })
	r.GET("/blocked", func(c *gin.Context) { c.String(200, "should not reach") })

	// Non-blocked path.
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ok", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("/ok status = %d, want 200", w.Code)
	}

	// Blocked path.
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/blocked", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("/blocked status = %d, want 403", w.Code)
	}
}

func TestURLBlacklistMiddleware_NilBlacklist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(URLBlacklistMiddleware(nil))
	r.GET("/anything", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/anything", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("nil blacklist should allow all, got %d", w.Code)
	}
}

func TestURLBlacklistMiddlewareWithHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bl := NewURLBlacklist()
	bl.Add("/secret")

	r := gin.New()
	customHandler := func(c *gin.Context) {
		c.JSON(http.StatusTeapot, gin.H{"error": "custom"})
	}
	r.Use(URLBlacklistMiddlewareWithHandler(bl, customHandler))
	r.GET("/secret", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/secret", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTeapot {
		t.Errorf("custom handler status = %d, want 418", w.Code)
	}
}
