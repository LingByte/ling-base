package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LingByte/ling-base/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// --- CircuitBreaker ---

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	assert.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_AllowClosed(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:      3,
		SuccessThreshold:      2,
		OpenTimeout:           100 * time.Millisecond,
		MaxConcurrentRequests: 10,
	})
	assert.True(t, cb.Allow())
	assert.True(t, cb.Allow())
}

func TestCircuitBreaker_MaxConcurrent(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:      5,
		SuccessThreshold:      2,
		OpenTimeout:           100 * time.Millisecond,
		MaxConcurrentRequests: 2,
	})
	assert.True(t, cb.Allow())
	assert.True(t, cb.Allow())
	assert.False(t, cb.Allow()) // exceeds max concurrent

	cb.RecordSuccess()
	assert.True(t, cb.Allow()) // one slot freed
}

func TestCircuitBreaker_OpenOnFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:      3,
		SuccessThreshold:      2,
		OpenTimeout:           100 * time.Millisecond,
		MaxConcurrentRequests: 10,
	})

	// Trigger 3 failures to open the breaker
	for i := 0; i < 3; i++ {
		assert.True(t, cb.Allow())
		cb.RecordFailure()
	}

	assert.Equal(t, StateOpen, cb.GetState())
	assert.False(t, cb.Allow()) // blocked while open
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:      2,
		SuccessThreshold:      2,
		OpenTimeout:           50 * time.Millisecond,
		MaxConcurrentRequests: 10,
	})

	// Open the breaker
	for i := 0; i < 2; i++ {
		cb.Allow()
		cb.RecordFailure()
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for open timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open on next Allow
	assert.True(t, cb.Allow())
	assert.Equal(t, StateHalfOpen, cb.GetState())
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:      1,
		SuccessThreshold:      2,
		OpenTimeout:           50 * time.Millisecond,
		MaxConcurrentRequests: 10,
	})

	// Open the breaker
	cb.Allow()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.GetState())

	time.Sleep(60 * time.Millisecond)

	// Half-open: record successes to close
	cb.Allow() // transitions to half-open
	assert.Equal(t, StateHalfOpen, cb.GetState())
	cb.RecordSuccess()

	// Need another allow+success to reach SuccessThreshold=2
	// But half-open limits concurrent to 1, so we need to record first
	// Actually after RecordSuccess in half-open, concurrent drops to 0
	cb.Allow()
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_HalfOpenBackToOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:      1,
		SuccessThreshold:      2,
		OpenTimeout:           50 * time.Millisecond,
		MaxConcurrentRequests: 10,
	})

	// Open the breaker
	cb.Allow()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.GetState())

	time.Sleep(60 * time.Millisecond)

	// Half-open: failure sends back to open
	cb.Allow()
	assert.Equal(t, StateHalfOpen, cb.GetState())
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.GetState())
}

func TestCircuitBreaker_RecordSuccessInClosed(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:      3,
		SuccessThreshold:      2,
		OpenTimeout:           100 * time.Millisecond,
		MaxConcurrentRequests: 10,
	})

	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordSuccess() // resets failure count in closed state
}

func TestCircuitBreaker_GetStats(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:      5,
		SuccessThreshold:      2,
		OpenTimeout:           100 * time.Millisecond,
		MaxConcurrentRequests: 10,
	})

	cb.Allow()
	cb.RecordFailure()

	stats := cb.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, StateClosed, stats["state"])
	assert.Equal(t, 1, stats["failure_count"])
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	assert.Equal(t, 5, cfg.FailureThreshold)
	assert.Equal(t, 3, cfg.SuccessThreshold)
	assert.Equal(t, 100, cfg.MaxConcurrentRequests)
}

// --- TimeoutCircuitManager ---

func TestNewTimeoutCircuitManager(t *testing.T) {
	mgr := NewTimeoutCircuitManager(DefaultTimeoutConfig(), DefaultCircuitBreakerConfig())
	assert.NotNil(t, mgr)
	assert.True(t, mgr.enableTimeout)
	assert.True(t, mgr.enableCircuitBreaker)
}

func TestTimeoutCircuitManager_GetTimeout_ExactMatch(t *testing.T) {
	mgr := NewTimeoutCircuitManager(TimeoutConfig{
		DefaultTimeout: 30 * time.Second,
		EndpointTimeouts: map[string]time.Duration{
			"/api/slow": 60 * time.Second,
		},
	}, DefaultCircuitBreakerConfig())

	assert.Equal(t, 60*time.Second, mgr.getTimeout("/api/slow"))
}

func TestTimeoutCircuitManager_GetTimeout_PrefixMatch(t *testing.T) {
	mgr := NewTimeoutCircuitManager(TimeoutConfig{
		DefaultTimeout: 30 * time.Second,
		EndpointTimeouts: map[string]time.Duration{
			"/api/upload/": 5 * time.Minute,
		},
	}, DefaultCircuitBreakerConfig())

	assert.Equal(t, 5*time.Minute, mgr.getTimeout("/api/upload/file"))
}

func TestTimeoutCircuitManager_GetTimeout_Default(t *testing.T) {
	mgr := NewTimeoutCircuitManager(TimeoutConfig{
		DefaultTimeout: 30 * time.Second,
	}, DefaultCircuitBreakerConfig())

	assert.Equal(t, 30*time.Second, mgr.getTimeout("/api/unknown"))
}

func TestTimeoutCircuitManager_GetCircuitBreaker(t *testing.T) {
	mgr := NewTimeoutCircuitManager(DefaultTimeoutConfig(), DefaultCircuitBreakerConfig())

	cb1 := mgr.getCircuitBreaker("/api/test")
	cb2 := mgr.getCircuitBreaker("/api/test")
	assert.Same(t, cb1, cb2) // same instance

	cb3 := mgr.getCircuitBreaker("/api/other")
	assert.NotSame(t, cb1, cb3) // different instance
}

func TestTimeoutCircuitManager_SetLongLivedPrefixes(t *testing.T) {
	mgr := NewTimeoutCircuitManager(DefaultTimeoutConfig(), DefaultCircuitBreakerConfig())
	mgr.SetLongLivedPrefixes([]string{"/api/stream/"})
	globalTimeoutCircuitManager = mgr

	assert.True(t, isLongLivedEndpoint("/api/stream/live"))
	assert.False(t, isLongLivedEndpoint("/api/normal"))

	globalTimeoutCircuitManager = nil
}

func TestDefaultTimeoutConfig(t *testing.T) {
	cfg := DefaultTimeoutConfig()
	assert.Equal(t, 30*time.Second, cfg.DefaultTimeout)
	assert.NotEmpty(t, cfg.EndpointTimeouts)
	assert.NotNil(t, cfg.FallbackResponse)
}

// --- Endpoint detection ---

func TestIsProbeEndpoint(t *testing.T) {
	probes := []string{"/healthz", "/livez", "/health", "/readyz", "/ready", "/metrics"}
	for _, p := range probes {
		assert.True(t, isProbeEndpoint(p), "%s should be probe", p)
	}
	assert.False(t, isProbeEndpoint("/api/data"))
}

func TestIsLongLivedEndpoint_WebSocket(t *testing.T) {
	assert.True(t, isLongLivedEndpoint("/api/ws"))
	assert.True(t, isLongLivedEndpoint("/api/v1/ws/chat"))
	assert.True(t, isLongLivedEndpoint("/ws"))
}

func TestIsLongLivedEndpoint_Stream(t *testing.T) {
	assert.True(t, isLongLivedEndpoint("/api/stream"))
	assert.True(t, isLongLivedEndpoint("/api/incoming/stream/live"))
}

func TestIsLongLivedEndpoint_Normal(t *testing.T) {
	// Reset global manager to avoid prefix interference
	globalTimeoutCircuitManager = nil
	assert.False(t, isLongLivedEndpoint("/api/users"))
	assert.False(t, isLongLivedEndpoint("/api/data"))
}

// --- Middleware integration ---

func TestTimeoutMiddleware_NormalRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	// Create a fresh manager with long timeout
	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 5 * time.Second},
		DefaultCircuitBreakerConfig(),
	)

	router := gin.New()
	router.Use(TimeoutMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTimeoutMiddleware_ProbeSkipped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 1 * time.Millisecond},
		DefaultCircuitBreakerConfig(),
	)

	router := gin.New()
	router.Use(TimeoutMiddleware())
	router.GET("/healthz", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond) // would timeout if not skipped
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTimeoutMiddleware_LongLivedSkipped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 1 * time.Millisecond},
		DefaultCircuitBreakerConfig(),
	)

	router := gin.New()
	router.Use(TimeoutMiddleware())
	router.GET("/api/ws", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond) // would timeout if not skipped
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/ws", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTimeoutMiddleware_Timeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 10 * time.Millisecond},
		DefaultCircuitBreakerConfig(),
	)

	router := gin.New()
	router.Use(TimeoutMiddleware())
	router.GET("/", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond) // exceeds timeout
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestTimeout, w.Code)
}

func TestCircuitBreakerMiddleware_AllowsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		DefaultTimeoutConfig(),
		DefaultCircuitBreakerConfig(),
	)

	router := gin.New()
	router.Use(CircuitBreakerMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCircuitBreakerMiddleware_BlockedWhenOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		DefaultTimeoutConfig(),
		CircuitBreakerConfig{
			FailureThreshold:      1,
			SuccessThreshold:      1,
			OpenTimeout:           10 * time.Second,
			MaxConcurrentRequests: 10,
		},
	)

	// Force the breaker open for "/" endpoint
	cb := globalTimeoutCircuitManager.getCircuitBreaker("/").(*legacyBreakerAdapter).cb
	cb.Allow()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.GetState())

	router := gin.New()
	router.Use(CircuitBreakerMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCombinedTimeoutCircuitMiddleware_BothDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	mgr := NewTimeoutCircuitManager(DefaultTimeoutConfig(), DefaultCircuitBreakerConfig())
	mgr.enableTimeout = false
	mgr.enableCircuitBreaker = false
	globalTimeoutCircuitManager = mgr

	router := gin.New()
	router.Use(CombinedTimeoutCircuitMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCombinedTimeoutCircuitMiddleware_NormalRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 5 * time.Second},
		DefaultCircuitBreakerConfig(),
	)

	router := gin.New()
	router.Use(CombinedTimeoutCircuitMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCombinedTimeoutCircuitMiddleware_ProbeSkipped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 1 * time.Millisecond},
		DefaultCircuitBreakerConfig(),
	)

	router := gin.New()
	router.Use(CombinedTimeoutCircuitMiddleware())
	router.GET("/healthz", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCombinedTimeoutCircuitMiddleware_CircuitBreakerOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	mgr := NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 5 * time.Second},
		DefaultCircuitBreakerConfig(),
	)
	mgr.enableTimeout = false
	mgr.enableCircuitBreaker = true
	globalTimeoutCircuitManager = mgr

	router := gin.New()
	router.Use(CombinedTimeoutCircuitMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCombinedTimeoutCircuitMiddleware_CircuitBreakerBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	mgr := NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 5 * time.Second},
		CircuitBreakerConfig{
			FailureThreshold:      1,
			SuccessThreshold:      1,
			OpenTimeout:           10 * time.Second,
			MaxConcurrentRequests: 10,
		},
	)
	mgr.enableTimeout = false
	mgr.enableCircuitBreaker = true
	globalTimeoutCircuitManager = mgr

	cb := mgr.getCircuitBreaker("/").(*legacyBreakerAdapter).cb
	cb.Allow()
	cb.RecordFailure()

	router := gin.New()
	router.Use(CombinedTimeoutCircuitMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCombinedTimeoutCircuitMiddleware_Timeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 10 * time.Millisecond},
		DefaultCircuitBreakerConfig(),
	)

	router := gin.New()
	router.Use(CombinedTimeoutCircuitMiddleware())
	router.GET("/", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestTimeout, w.Code)
}

func TestGetCircuitBreakerStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		DefaultTimeoutConfig(),
		DefaultCircuitBreakerConfig(),
	)

	// Touch a breaker to create it
	globalTimeoutCircuitManager.getCircuitBreaker("/api/test")

	stats := GetCircuitBreakerStats()
	assert.NotNil(t, stats)
	_, ok := stats["/api/test"]
	assert.True(t, ok)
}

func TestInitTimeoutCircuitManager(t *testing.T) {
	// sync.Once means this only runs once per process; verify it doesn't panic
	// and the global manager is usable afterward
	InitTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 10 * time.Second},
		DefaultCircuitBreakerConfig(),
		false, false,
	)
	mgr := GetTimeoutCircuitManager()
	assert.NotNil(t, mgr)
}

func TestCircuitBreakerMiddleware_5xxRecordsFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		DefaultTimeoutConfig(),
		CircuitBreakerConfig{
			FailureThreshold:      2,
			SuccessThreshold:      1,
			OpenTimeout:           10 * time.Second,
			MaxConcurrentRequests: 10,
		},
	)

	router := gin.New()
	router.Use(CircuitBreakerMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	cb := globalTimeoutCircuitManager.getCircuitBreaker("/").(*legacyBreakerAdapter).cb
	assert.Equal(t, 1, cb.GetStats()["failure_count"])
}

func TestCircuitBreakerMiddleware_4xxNoRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		DefaultTimeoutConfig(),
		CircuitBreakerConfig{
			FailureThreshold:      2,
			SuccessThreshold:      1,
			OpenTimeout:           10 * time.Second,
			MaxConcurrentRequests: 10,
		},
	)

	router := gin.New()
	router.Use(CircuitBreakerMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusBadRequest) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	cb := globalTimeoutCircuitManager.getCircuitBreaker("/").(*legacyBreakerAdapter).cb
	assert.Equal(t, 0, cb.GetStats()["failure_count"])
}

func TestCombinedTimeoutCircuitMiddleware_5xxRecordsFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 5 * time.Second},
		CircuitBreakerConfig{
			FailureThreshold:      2,
			SuccessThreshold:      1,
			OpenTimeout:           10 * time.Second,
			MaxConcurrentRequests: 10,
		},
	)

	router := gin.New()
	router.Use(CombinedTimeoutCircuitMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	cb := globalTimeoutCircuitManager.getCircuitBreaker("/").(*legacyBreakerAdapter).cb
	assert.Equal(t, 1, cb.GetStats()["failure_count"])
}

func TestCombinedTimeoutCircuitMiddleware_2xxRecordsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 5 * time.Second},
		CircuitBreakerConfig{
			FailureThreshold:      2,
			SuccessThreshold:      1,
			OpenTimeout:           10 * time.Second,
			MaxConcurrentRequests: 10,
		},
	)

	// First allow + failure to get a non-zero failure count
	cb := globalTimeoutCircuitManager.getCircuitBreaker("/").(*legacyBreakerAdapter).cb
	cb.Allow()
	cb.RecordFailure()

	router := gin.New()
	router.Use(CombinedTimeoutCircuitMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Success should have reset failure count in closed state
	assert.Equal(t, 0, cb.GetStats()["failure_count"])
}

func TestCombinedTimeoutCircuitMiddleware_PanicInGoroutine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Lg = zap.NewNop()
	defer func() { logger.Lg = nil }()

	globalTimeoutCircuitManager = NewTimeoutCircuitManager(
		TimeoutConfig{DefaultTimeout: 5 * time.Second},
		CircuitBreakerConfig{
			FailureThreshold:      5,
			SuccessThreshold:      1,
			OpenTimeout:           10 * time.Second,
			MaxConcurrentRequests: 10,
		},
	)

	router := gin.New()
	router.Use(CombinedTimeoutCircuitMiddleware())
	router.GET("/", func(c *gin.Context) { panic("test panic in goroutine") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
