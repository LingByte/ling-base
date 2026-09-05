// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package gin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	rbacgin "github.com/LingByte/ling-base/common/rbac/gin"
	"github.com/LingByte/ling-base/common/rbac"
	"github.com/gin-gonic/gin"
)

func TestMiddleware_Allow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()
	mgr.AddPolicy("admin", "/api/users", "GET")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		rbacgin.SetSubject(c, "admin")
		c.Next()
	})
	r.Use(rbacgin.Middleware(mgr))
	r.GET("/api/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestMiddleware_Deny(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()
	mgr.AddPolicy("admin", "/api/users", "GET")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		rbacgin.SetSubject(c, "viewer") // viewer has no policy
		c.Next()
	})
	r.Use(rbacgin.Middleware(mgr))
	r.GET("/api/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want 403; body: %s", w.Code, w.Body.String())
	}
}

func TestMiddleware_PathPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()
	// Policy without prefix; request path has prefix.
	mgr.AddPolicy("admin", "/users", "GET")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		rbacgin.SetSubject(c, "admin")
		c.Next()
	})
	r.Use(rbacgin.Middleware(mgr, rbacgin.WithPathPrefix("/api")))
	r.GET("/api/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestMiddleware_CustomDenyHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	denyCalled := false
	r := gin.New()
	r.Use(func(c *gin.Context) {
		rbacgin.SetSubject(c, "viewer")
		c.Next()
	})
	r.Use(rbacgin.Middleware(mgr, rbacgin.WithDenyHandler(func(c *gin.Context, sub, obj, act string) {
		denyCalled = true
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"custom": true})
	})))
	r.GET("/api/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	r.ServeHTTP(w, req)

	if !denyCalled {
		t.Error("Custom deny handler was not called")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want 403", w.Code)
	}
}

func TestMiddleware_RoleInheritance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()
	mgr.AddPolicy("admin", "/api/users", "GET")
	mgr.AssignRole("alice", "admin")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		rbacgin.SetSubject(c, "alice")
		c.Next()
	})
	r.Use(rbacgin.Middleware(mgr))
	r.GET("/api/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want 200 (alice inherits admin); body: %s", w.Code, w.Body.String())
	}
}
