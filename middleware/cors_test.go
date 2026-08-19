// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_Default(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	// 带 Origin 的请求
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://example.com")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected ACAO=*, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_NoOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	// 不带 Origin 的请求（非浏览器），不设置 CORS 头
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no ACAO for non-browser request, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_Preflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://example.com")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected ACAO=*, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected non-empty ACAM")
	}
}

func TestCORS_SpecificOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSWithConfig(CORSConfig{
		AllowOrigins:     []string{"https://app.example.com"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	// 允许的 Origin
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Errorf("expected ACAO=https://app.example.com, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("expected ACAC=true, got %s", w.Header().Get("Access-Control-Allow-Credentials"))
	}

	// 不允许的 Origin
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req2.Header.Set("Origin", "https://evil.com")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed origin, got %d", w2.Code)
	}
}
