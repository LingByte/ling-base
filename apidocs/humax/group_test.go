// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package humax

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/ling-base/apidocs"
	"github.com/gin-gonic/gin"
)

func TestGroup_BasicRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := apidocs.Mount(r, apidocs.Options{Title: "test", Version: "0.0.0"})

	g := NewGroup(api, r, "/api/v1")
	g.GET("/users", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	g.POST("/users", func(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"ok": true}) })
	g.GET("/users/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"id": c.Param("id")}) })

	// Gin 路由能正常执行
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// OpenAPI 文档包含路由
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w2, req2)

	var spec map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &spec); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("no paths in OpenAPI spec")
	}

	expected := []string{"/api/v1/users", "/api/v1/users/{id}"}
	for _, p := range expected {
		if _, ok := paths[p]; !ok {
			t.Errorf("OpenAPI spec missing path: %s", p)
		}
	}
}

func TestGroup_SubGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := apidocs.Mount(r, apidocs.Options{Title: "test", Version: "0.0.0"})

	g := NewGroup(api, r, "/api/v1")
	auth := g.Group("/auth")
	auth.POST("/login", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// OpenAPI 文档包含子组路由
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w2, req2)

	var spec map[string]any
	json.Unmarshal(w2.Body.Bytes(), &spec)
	paths := spec["paths"].(map[string]any)
	if _, ok := paths["/api/v1/auth/login"]; !ok {
		t.Error("OpenAPI spec missing /api/v1/auth/login")
	}
}

func TestGinPathToOpenAPI(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		{"/api/v1/users", "/api/v1/users"},
		{"/api/v1/users/:id", "/api/v1/users/{id}"},
		{"/api/v1/users/*path", "/api/v1/users/{path}"},
	}
	for _, tt := range tests {
		got := GinPathToOpenAPI(tt.in)
		if got != tt.out {
			t.Errorf("GinPathToOpenAPI(%s) = %s, want %s", tt.in, got, tt.out)
		}
	}
}

func TestOperationID(t *testing.T) {
	id := OperationID("GET", "/api/v1/users/{id}")
	if id != "get-api-v1-users-id" {
		t.Errorf("unexpected operationId: %s", id)
	}
}
