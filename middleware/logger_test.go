package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LingByte/ling-base/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLoggerMiddleware_GETSkipped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	router := gin.New()
	router.Use(LoggerMiddleware(zap.NewNop()))
	router.GET("/api/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/data", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoggerMiddleware_POSTLogged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	router := gin.New()
	router.Use(LoggerMiddleware(zap.NewNop()))
	router.POST("/api/create", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"id": 1})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/create", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestLoggerMiddleware_NilLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = nil

	router := gin.New()
	router.Use(LoggerMiddleware(nil))
	router.POST("/api/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoggerMiddleware_SkipAccessLogPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	SetSkipAccessLogPaths([]string{"/metrics"})
	defer SetSkipAccessLogPaths(nil)

	router := gin.New()
	router.Use(LoggerMiddleware(zap.NewNop()))
	router.POST("/metrics", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestShouldSkipAccessLog(t *testing.T) {
	SetSkipAccessLogPaths([]string{"/static", "/metrics"})
	defer SetSkipAccessLogPaths(nil)

	assert.True(t, shouldSkipAccessLog("/static/css/app.css"))
	assert.True(t, shouldSkipAccessLog("/api/metrics/foo"))
	assert.False(t, shouldSkipAccessLog("/api/users"))
}

func TestShouldSkipAccessLog_EmptyList(t *testing.T) {
	SetSkipAccessLogPaths(nil)
	assert.False(t, shouldSkipAccessLog("/anything"))
}

func TestLoggerMiddleware_WithErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	router := gin.New()
	router.Use(LoggerMiddleware(zap.NewNop()))
	router.POST("/api/error", func(c *gin.Context) {
		_ = c.Error(errors.New("some error"))
		c.Status(http.StatusBadRequest)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoggerMiddleware_SlowRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	router := gin.New()
	router.Use(LoggerMiddleware(zap.NewNop()))
	router.POST("/api/slow", func(c *gin.Context) {
		time.Sleep(350 * time.Millisecond) // exceeds slowHTTPThreshold (300ms)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/slow", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
