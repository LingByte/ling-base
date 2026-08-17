package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/circuitbreaker"
	"github.com/LingByte/ling-base/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestDefaultBreakerFactory(t *testing.T) {
	f := DefaultBreakerFactory(circuitbreaker.Config{
		MaxRequests:      3,
		FailureThreshold: 0.5,
		MinRequests:      2,
	})
	b := f("/api/test")
	assert.NotNil(t, b)
	assert.NoError(t, b.Allow())
}

func TestSREBreakerFactory(t *testing.T) {
	f := SREBreakerFactory(
		circuitbreaker.WithSRERequest(1),
		circuitbreaker.WithSREWindow(time.Second),
	)
	b := f("/api/test")
	assert.NotNil(t, b)
	assert.NoError(t, b.Allow())
}

func TestNoopBreakerFactory(t *testing.T) {
	f := NoopBreakerFactory()
	b := f("/api/test")
	assert.NotNil(t, b)
	assert.NoError(t, b.Allow())
	b.MarkSuccess()
	b.MarkFailed()
}

func TestSetBreakerFactory_SRE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	mgr := NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 5 * time.Second},
		DefaultCircuitBreakerConfig(),
	)
	mgr.enableTimeout = false
	mgr.enableCircuitBreaker = true
	mgr.SetBreakerFactory(SREBreakerFactory(
		circuitbreaker.WithSRERequest(1),
		circuitbreaker.WithSREWindow(time.Second),
		circuitbreaker.WithSREBucket(2),
	))
	globalTimeoutCircuitManager = mgr

	router := gin.New()
	router.Use(CircuitBreakerMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetBreakerFactory_Noop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	mgr := NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 5 * time.Second},
		DefaultCircuitBreakerConfig(),
	)
	mgr.enableTimeout = false
	mgr.enableCircuitBreaker = true
	mgr.SetBreakerFactory(NoopBreakerFactory())
	globalTimeoutCircuitManager = mgr

	router := gin.New()
	router.Use(CircuitBreakerMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	// Noop breaker never blocks, even after 5xx
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	}
}

func TestLegacyBreakerAdapter(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:      1,
		SuccessThreshold:      1,
		OpenTimeout:           10 * time.Second,
		MaxConcurrentRequests: 10,
	})
	adapter := &legacyBreakerAdapter{cb: cb}

	assert.NoError(t, adapter.Allow())
	adapter.MarkFailed()

	// Should be open now
	err := adapter.Allow()
	assert.Error(t, err)
	assert.Equal(t, "middleware: circuit breaker open", err.Error())
}

func TestErrCircuitOpenSentinel(t *testing.T) {
	e := errCircuitOpenSentinel{}
	assert.Equal(t, "middleware: circuit breaker open", e.Error())
}
