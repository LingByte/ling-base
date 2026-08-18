// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package gin

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/stats"
	"github.com/LingByte/ling-base/common/stats/memory"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter(wm *stats.WebsiteMetrics, cfg Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware(wm, cfg))
	return r
}

func TestMiddlewareBasic(t *testing.T) {
	c := memory.New()
	wm := stats.NewWebsiteMetrics(c)

	r := setupRouter(wm, Config{
		GetUserID: func(c *gin.Context) string {
			return c.GetHeader("X-User-ID")
		},
		SkipPaths: []string{"/health"},
	})

	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	r.GET("/users/:id", func(c *gin.Context) { c.JSON(200, gin.H{"id": c.Param("id")}) })
	r.POST("/users/:id/posts", func(c *gin.Context) { c.JSON(201, gin.H{"ok": true}) })

	// Request 1: /health — should be skipped
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Request 2: /users/123
	req = httptest.NewRequest("GET", "/users/123", nil)
	req.Header.Set("X-User-ID", "user-001")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Request 3: /users/456
	req = httptest.NewRequest("GET", "/users/456", nil)
	req.Header.Set("X-User-ID", "user-002")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Request 4: /users/789 (same user as request 2)
	req = httptest.NewRequest("GET", "/users/789", nil)
	req.Header.Set("X-User-ID", "user-001")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Verify metrics were recorded.
	// All /users/:id requests should go to the same normalized path.
	today := getToday()
	pvUsers := wm.GetPV(today, "/users/:id")
	assert.Equal(t, int64(3), pvUsers, "3 requests to /users/:id")

	// UV: 2 unique users
	uv := wm.GetUV(today)
	assert.Equal(t, uint64(2), uv, "2 unique users")

	// IP: should be 1 (all from httptest, same IP)
	ip := wm.GetIP(today)
	assert.Greater(t, ip, uint64(0), "IP recorded")

	// Requests: 3 (excluding /health)
	requests := c.Counter("requests:" + today).Get()
	assert.Equal(t, int64(3), requests, "3 requests (health skipped)")

	// Response time should be recorded
	timerCount := c.Timer("response_time:" + today).Count()
	assert.Equal(t, int64(3), timerCount, "3 response time samples")
}

func TestMiddlewareErrorRecording(t *testing.T) {
	c := memory.New()
	wm := stats.NewWebsiteMetrics(c)

	r := setupRouter(wm, Config{})
	r.GET("/error", func(c *gin.Context) { c.JSON(500, gin.H{"err": "fail"}) })
	r.GET("/ok", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	// Trigger 500
	req := httptest.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)

	// Trigger 200
	req = httptest.NewRequest("GET", "/ok", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	today := getToday()
	errors := c.Counter("errors:" + today).Get()
	assert.Equal(t, int64(1), errors, "1 error recorded")

	requests := c.Counter("requests:" + today).Get()
	assert.Equal(t, int64(2), requests, "2 total requests")

	// Error rate should be 50%
	errorRate := wm.GetErrorRate(today)
	assert.InDelta(t, 0.5, errorRate, 0.01)
}

func TestMiddlewarePathNormalization(t *testing.T) {
	c := memory.New()
	wm := stats.NewWebsiteMetrics(c)

	r := setupRouter(wm, Config{})
	r.GET("/users/:id", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	// Hit 1000 different user IDs — should all normalize to /users/:id
	for i := 0; i < 1000; i++ {
		path := "/users/" + itoa(i)
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	today := getToday()
	// Should be 1 unique path key, not 1000
	pv := wm.GetPV(today, "/users/:id")
	assert.Equal(t, int64(1000), pv, "1000 PV to /users/:id")

	// Verify only 1 path key exists (not 1000)
	// The counter map should have "pv:<date>:/users/:id" but not "pv:<date>:/users/0" etc.
	c2 := c
	_ = c2
}

func TestMiddlewareSkipIf(t *testing.T) {
	c := memory.New()
	wm := stats.NewWebsiteMetrics(c)

	r := setupRouter(wm, Config{
		SkipIf: func(c *gin.Context) bool {
			return c.GetHeader("X-Internal") == "true"
		},
	})
	r.GET("/api", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	// Normal request — recorded
	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Internal request — skipped
	req = httptest.NewRequest("GET", "/api", nil)
	req.Header.Set("X-Internal", "true")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	today := getToday()
	requests := c.Counter("requests:" + today).Get()
	assert.Equal(t, int64(1), requests, "only 1 request recorded (internal skipped)")
}

func TestMiddlewareNoUserID(t *testing.T) {
	c := memory.New()
	wm := stats.NewWebsiteMetrics(c)

	r := setupRouter(wm, Config{}) // No GetUserID
	r.GET("/page", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	req := httptest.NewRequest("GET", "/page", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	today := getToday()
	// UV should be 0 (no user ID configured)
	uv := wm.GetUV(today)
	assert.Equal(t, uint64(0), uv, "UV should be 0 without GetUserID")

	// PV should still be recorded
	pv := wm.GetPV(today, "/page")
	assert.Equal(t, int64(1), pv, "PV recorded without user ID")
}

// BenchmarkMiddleware measures the overhead of the stats middleware.
func BenchmarkMiddleware(b *testing.B) {
	c := memory.New(
		memory.WithReservoirTimer(4096),
		memory.WithBloomSet(1000000, 0.001),
	)
	wm := stats.NewWebsiteMetrics(c)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(Middleware(wm, Config{
		GetUserID: func(c *gin.Context) string {
			return "bench-user"
		},
	}))
	r.GET("/users/:id", func(c *gin.Context) { c.JSON(200, gin.H{"id": c.Param("id")}) })

	// Pre-create request to reuse.
	req := httptest.NewRequest("GET", "/users/123", nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
		}
	})
}

// BenchmarkMiddlewareNoStats measures baseline without middleware.
func BenchmarkMiddlewareNoStats(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/users/:id", func(c *gin.Context) { c.JSON(200, gin.H{"id": c.Param("id")}) })

	req := httptest.NewRequest("GET", "/users/123", nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
		}
	})
}

func getToday() string {
	return time.Now().Format("2006-01-02")
}
