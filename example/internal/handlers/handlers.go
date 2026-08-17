// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package handlers contains HTTP request handlers wired onto a gin engine.
//
// The Handlers struct holds shared dependencies (DB, Redis cache, search engine,
// config, limiters) and implements bootstrap.Lifecycle so it can be registered
// directly with the application. A single Register method attaches all routes —
// following the same pattern as LingEchoX's internal/handlers.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/LingByte/ling-base/cache"
	redisCache "github.com/LingByte/ling-base/cache/redis"
	"github.com/LingByte/ling-base/circuitbreaker"
	"github.com/LingByte/ling-base/common/config"
	"github.com/LingByte/ling-base/common/convert"
	"github.com/LingByte/ling-base/common/idgen"
	"github.com/LingByte/ling-base/common/response"
	respgin "github.com/LingByte/ling-base/common/response/gin"
	"github.com/LingByte/ling-base/common/validate"
	"github.com/LingByte/ling-base/eventbus"
	"github.com/LingByte/ling-base/example/internal/listeners"
	"github.com/LingByte/ling-base/example/internal/models"
	"github.com/LingByte/ling-base/limiter"
	"github.com/LingByte/ling-base/limiter/count"
	"github.com/LingByte/ling-base/limiter/tokenbucket"
	"github.com/LingByte/ling-base/logger"
	"github.com/LingByte/ling-base/notification"
	"github.com/LingByte/ling-base/notification/inbox"
	"github.com/LingByte/ling-base/pool"
	"github.com/LingByte/ling-base/queue"
	memoryQueue "github.com/LingByte/ling-base/queue/memory"
	"github.com/LingByte/ling-base/common/retry"
	"github.com/LingByte/ling-base/search"
	"github.com/LingByte/ling-base/search/bleve"
	"github.com/LingByte/ling-base/version"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handlers holds shared dependencies for all HTTP handlers and manages the
// gin HTTP server lifecycle as a bootstrap.Lifecycle component.
type Handlers struct {
	db              *gorm.DB                       // optional, nil in env-only mode
	cache           cache.Cache[string, []byte]                    // optional, nil in no-redis mode
	search          search.Engine                  // optional, nil in no-search mode
	cb              *circuitbreaker.CircuitBreaker // circuit breaker for demo endpoint
	taskQueue       queue.Queue                    // task queue backend
	scheduler       *queue.CapacityScheduler       // capacity-aware task scheduler
	workerPool      *pool.WorkerPool               // goroutine worker pool
	cfg             *config.Store                  // config store
	events          eventbus.Bus                   // event bus for lifecycle events
	rl              limiter.Limiter                // request rate limiter (token bucket)
	connRL          limiter.Limiter                // connection count limiter
	notifDispatcher *notification.Dispatcher       // notification dispatcher
	inboxStore      *inbox.MemoryStore             // inbox storage backend

	port   int
	host   string
	server *http.Server

	startTime time.Time
}

// requestCounter is shared across all HTTP handlers for sequence numbering.
var requestCounter atomic.Int64

// NewHandlers creates a new Handlers instance with the given dependencies.
// db, redis cache, and search engine may be nil — the server degrades gracefully.
// Limiters are initialized from the config store values.
func NewHandlers(db *gorm.DB, cache cache.Cache[string, []byte], searchEng search.Engine, cfg *config.Store, events eventbus.Bus) *Handlers {
	rps := cfg.GetFloatValue("RATE_LIMIT_RPS", 100)
	maxConns := cfg.GetIntValue("RATE_LIMIT_MAX_CONNS", 1000)

	// Create in-memory task queue + capacity scheduler with rich handler.
	taskQueue := memoryQueue.New("demo-tasks")
	workerPool := pool.NewWorkerPool(4, 100)

	scheduler, err := queue.NewCapacityScheduler(queue.CapacitySchedulerConfig{
		Queue:       taskQueue,
		WorkerCount: 4,
		Capacity:    10,
		Strategy:    queue.StrategyPreemptive,
		// RichHandler receives a TaskContext for progress + logging.
		RichHandler: func(tctx queue.TaskContext, task *queue.Task) error {
			var payload struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(task.Payload, &payload); err != nil {
				_ = tctx.Log(queue.LogLevelError, "payload unmarshal failed: "+err.Error())
				return err
			}
			_ = tctx.Log(queue.LogLevelInfo, "starting task: "+payload.Message)
			_ = tctx.SetProgress(10)

			// Simulate work with progress updates and cancellation checks.
			steps := 5
			for i := 0; i < steps; i++ {
				if tctx.Err() != nil {
					_ = tctx.Log(queue.LogLevelWarn, "task canceled at step "+itoa(i))
					return tctx.Err()
				}
				time.Sleep(100 * time.Millisecond)
				_ = tctx.SetProgress(10 + (i+1)*18)
			}
			_ = tctx.Log(queue.LogLevelInfo, "task completed")
			_ = tctx.SetProgress(100)
			return nil
		},
		OnTaskComplete: func(task *queue.Task, err error) {
			if err != nil {
				logger.Warn("task failed",
					zap.String("task_id", task.ID),
					zap.Error(err),
				)
			} else {
				logger.Info("task completed",
					zap.String("task_id", task.ID),
				)
			}
		},
		OnPreempt: func(task *queue.Task) {
			logger.Info("task preempted",
				zap.String("task_id", task.ID),
				zap.Int("priority", task.Priority),
			)
		},
		OnRecover: func(count int) {
			logger.Info("task queue recovered", zap.Int("count", count))
		},
		EnablePreemption: true,
		DequeueTimeout:   500 * time.Millisecond,
	})
	if err != nil {
		logger.Error("failed to create task scheduler", zap.Error(err))
	}

	h := &Handlers{
		db:         db,
		cache:      cache,
		search:     searchEng,
		taskQueue:  taskQueue,
		scheduler:  scheduler,
		workerPool: workerPool,
		cb: circuitbreaker.New(circuitbreaker.Config{
			Name:              "demo",
			MaxRequests:       3,
			FailureThreshold:  0.5,
			MinRequests:       4,
			RecoveryTimeout:   10 * time.Second,
			SlidingWindowSize: 20,
			OnStateChange: func(name string, from, to circuitbreaker.State) {
				logger.Info("circuit breaker state changed",
					zap.String("breaker", name),
					zap.String("from", from.String()),
					zap.String("to", to.String()),
				)
			},
		}),
		cfg:    cfg,
		events: events,
		rl:     tokenbucket.New(int(rps), int(rps)),
		connRL: count.New(maxConns),
	}

	// Auto-migrate the request_logs table if DB is available.
	if db != nil {
		if err := db.AutoMigrate(&models.RequestLog{}); err != nil {
			logger.Warn("auto-migrate request_logs table failed", zap.Error(err))
		}
	}

	return h
}

// ──────────────────────────────────────────────
// bootstrap.Lifecycle implementation
// ──────────────────────────────────────────────

// Name returns the component name.
func (h *Handlers) Name() string { return "api" }

// Start creates the gin engine, registers routes, and launches the HTTP server.
func (h *Handlers) Start(ctx context.Context) error {
	h.port = h.cfg.GetIntValue("SERVER_PORT", 8080)
	h.host = h.cfg.GetStringWithDefault("SERVER_HOST", "0.0.0.0")
	h.startTime = time.Now()

	// Set gin mode based on profile.
	profile := h.cfg.GetStringWithDefault("APP_ENV", "dev")
	if profile == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(respgin.Recovery())

	// Initialize notification subsystem.
	h.inboxStore = inbox.NewMemoryStore()
	h.notifDispatcher = notification.NewDispatcher()
	// Register inbox as a notification channel.
	h.notifDispatcher.AddChannel(inbox.NewChannel("inbox", h.inboxStore))
	// Register a mock email channel for demo purposes.
	h.notifDispatcher.AddChannel(notification.NewChannelFunc("email-mock", notification.TypeEmail, true,
		func(ctx context.Context, msg notification.Message) error {
			logger.Info("mock email sent",
				zap.String("to", msg.To),
				zap.String("subject", msg.Subject),
			)
			return nil
		},
	))

	// Register all routes.
	h.Register(engine)

	addr := fmt.Sprintf("%s:%d", h.host, h.port)
	h.server = &http.Server{
		Addr:    addr,
		Handler: engine,
	}

	go func() {
		logger.Info("api server started",
			zap.String("addr", addr),
			zap.Bool("db", h.db != nil),
			zap.Bool("redis", h.cache != nil),
			zap.Bool("search", h.search != nil),
		)
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("api server error", zap.Error(err))
		}
	}()

	// Start the task scheduler and worker pool.
	if h.scheduler != nil {
		if err := h.scheduler.Start(); err != nil {
			logger.Error("task scheduler start error", zap.Error(err))
		}
	}
	if h.workerPool != nil {
		h.workerPool.Start()
	}

	return nil
}

// Stop gracefully shuts down the HTTP server.
func (h *Handlers) Stop(ctx context.Context) error {
	if h.scheduler != nil {
		_ = h.scheduler.Stop()
	}
	if h.workerPool != nil {
		h.workerPool.Stop()
	}
	if h.taskQueue != nil {
		_ = h.taskQueue.Close()
	}
	if h.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := h.server.Shutdown(shutdownCtx); err != nil {
			logger.Warn("api server shutdown error", zap.Error(err))
		}
	}
	if h.search != nil {
		_ = h.search.Close()
	}
	if h.cache != nil {
		_ = h.cache.Clear(ctx)
	}
	logger.Info("api server stopped")
	return nil
}

// IsRunning reports whether the component is active.
func (h *Handlers) IsRunning() bool { return h.server != nil }

// Port returns the configured server port.
func (h *Handlers) Port() int { return h.port }

// Host returns the configured server host.
func (h *Handlers) Host() string { return h.host }

// ──────────────────────────────────────────────
// Route registration
// ──────────────────────────────────────────────

// Register wires the full HTTP router onto the gin engine.
// This is the single entry point for route registration.
func (h *Handlers) Register(engine *gin.Engine) {
	engine.GET("/health", h.HealthHandler)
	engine.GET("/info", h.InfoHandler)

	api := engine.Group("/api/v1")
	h.registerEchoRoutes(api)
	h.registerRequestRoutes(api)
	h.registerSearchRoutes(api)
	h.registerRetryRoutes(api)
	h.registerCircuitBreakerRoutes(api)
	h.registerQueueRoutes(api)
	h.registerConvertRoutes(api)
	h.registerValidateRoutes(api)
	h.registerResponseRoutes(api)
	h.registerNotificationRoutes(api)
}

// registerEchoRoutes registers the echo endpoint group.
func (h *Handlers) registerEchoRoutes(r *gin.RouterGroup) {
	r.GET("/echo", h.EchoHandler)
	r.POST("/echo", h.EchoHandler)
}

// registerRequestRoutes registers the request log query endpoints.
func (h *Handlers) registerRequestRoutes(r *gin.RouterGroup) {
	r.GET("/requests", h.ListRequestsHandler)
}

// registerSearchRoutes registers the full-text search endpoints.
func (h *Handlers) registerSearchRoutes(r *gin.RouterGroup) {
	r.POST("/search", h.SearchHandler)
	r.POST("/search/index", h.SearchIndexHandler)
}

// registerRetryRoutes registers the retry demo endpoint.
func (h *Handlers) registerRetryRoutes(r *gin.RouterGroup) {
	r.POST("/retry", h.RetryHandler)
}

// registerCircuitBreakerRoutes registers the circuit breaker demo endpoints.
func (h *Handlers) registerCircuitBreakerRoutes(r *gin.RouterGroup) {
	r.POST("/cb", h.CircuitBreakerHandler)
	r.GET("/cb/state", h.CircuitBreakerStateHandler)
}

// registerQueueRoutes registers the task queue demo endpoints.
func (h *Handlers) registerQueueRoutes(r *gin.RouterGroup) {
	r.POST("/tasks", h.EnqueueTaskHandler)
	r.GET("/tasks", h.ListTasksHandler)
	r.GET("/tasks/stats", h.TaskStatsHandler)
	r.POST("/tasks/:id/cancel", h.CancelTaskHandler)
	r.GET("/tasks/:id", h.GetTaskHandler)
	r.GET("/tasks/:id/logs", h.TaskLogsHandler)
}

// ──────────────────────────────────────────────
// Public handlers — no rate limiting
// ──────────────────────────────────────────────

// HealthHandler returns the application health status.
func (h *Handlers) HealthHandler(c *gin.Context) {
	checks := map[string]string{
		"api": "ok",
	}
	if h.db != nil {
		checks["db"] = "ok"
	} else {
		checks["db"] = "disabled"
	}
	if h.cache != nil {
		checks["redis"] = "ok"
	} else {
		checks["redis"] = "disabled"
	}
	if h.search != nil {
		checks["search"] = "ok"
	} else {
		checks["search"] = "disabled"
	}

	respgin.Success(c, &models.HealthStatus{
		Status:    "healthy",
		App:       "app-demo",
		Version:   version.GetVersion(),
		Profile:   h.cfg.GetStringWithDefault("APP_ENV", "dev"),
		Uptime:    time.Since(h.startTime).Round(time.Second).String(),
		Checks:    checks,
		Timestamp: time.Now(),
	})
}

// InfoHandler returns application metadata.
func (h *Handlers) InfoHandler(c *gin.Context) {
	respgin.Success(c, gin.H{
		"app":       "app-demo",
		"version":   version.GetVersion(),
		"go":        version.GetGoVersion(),
		"commit":    version.GetGitCommit(),
		"buildTime": version.GetBuildTime(),
		"host":      h.host,
		"port":      h.port,
	})
}

// ──────────────────────────────────────────────
// API v1 handlers — rate limited
// ──────────────────────────────────────────────

// EchoHandler echoes back the request info, applies rate limiting, and
// optionally persists the request log to the database.
func (h *Handlers) EchoHandler(c *gin.Context) {
	ctx := c.Request.Context()
	seq := int(requestCounter.Add(1))

	// Apply rate limiting.
	reqID, duration, err := h.handleWithLimit(ctx)
	if err != nil {
		logger.Warn("request rejected",
			zap.Int("seq", seq),
			zap.String("path", c.Request.URL.Path),
			zap.Error(err),
		)
		respgin.FailWithCode(c, response.CodeRateLimited, "rate limited", nil)
		return
	}

	// Emit request.completed signal — the listeners package handles
	// async DB persistence. This decouples the handler from storage I/O.
	rec := models.NewRequestLog(seq, c.Request.Method, c.Request.URL.Path, "ok", c.ClientIP(), duration)
	_ = h.events.Publish(ctx, eventbus.New(listeners.SigRequestCompleted, rec))

	logger.Info("request handled",
		zap.Int("seq", seq),
		zap.String("requestId", reqID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("clientIP", c.ClientIP()),
		zap.Duration("duration", duration),
	)

	respgin.Success(c, gin.H{
		"requestId": reqID,
		"seq":       seq,
		"method":    c.Request.Method,
		"path":      c.Request.URL.Path,
		"clientIP":  c.ClientIP(),
		"duration":  duration.String(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ListRequestsHandler returns recent request logs from the DB.
// If DB is not configured, returns an empty list with a notice.
func (h *Handlers) ListRequestsHandler(c *gin.Context) {
	if h.db == nil {
		respgin.Success(c, gin.H{
			"requests": []interface{}{},
			"message":  "DB persistence is disabled",
		})
		return
	}

	var records []models.RequestLog
	limit := 50
	result := h.db.WithContext(c.Request.Context()).
		Order("seq DESC").
		Limit(limit).
		Find(&records)
	if result.Error != nil {
		respgin.FailWithCode(c, response.CodeInternal, "query failed", nil)
		return
	}

	respgin.Success(c, gin.H{
		"requests": records,
		"count":    len(records),
	})
}

// ──────────────────────────────────────────────
// Search handlers — uses search engine + redis cache
// ──────────────────────────────────────────────

// SearchHandler executes a full-text search. Results are cached in Redis
// when available, with a 5-minute TTL.
func (h *Handlers) SearchHandler(c *gin.Context) {
	var req models.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid request body", nil)
		return
	}

	if req.Size <= 0 {
		req.Size = 10
	}

	if h.search == nil {
		respgin.FailWithCode(c, response.CodeServiceUnavail, "search engine is disabled", nil)
		return
	}

	ctx := c.Request.Context()
	cacheKey := fmt.Sprintf("search:%s:%d", req.Keyword, req.Size)

	// Try Redis cache first.
	if h.cache != nil {
		if data, err := h.cache.Get(ctx, cacheKey); err == nil {
			var cached models.SearchResponse
			if json.Unmarshal(data, &cached) == nil {
				cached.Cached = true
				respgin.Success(c, cached)
				return
			}
		}
	}

	// Execute search.
	res, err := h.search.Search(ctx, search.NewKeywordSearch(req.Keyword, req.Size))
	if err != nil {
		logger.Warn("search failed", zap.String("keyword", req.Keyword), zap.Error(err))
		respgin.FailWithCode(c, response.CodeInternal, "search failed", nil)
		return
	}

	// Build response.
	hits := make([]models.Hit, 0, len(res.Hits))
	for _, hit := range res.Hits {
		hits = append(hits, models.Hit{
			ID:     hit.ID,
			Score:  hit.Score,
			Fields: hit.Fields,
		})
	}

	resp := models.SearchResponse{
		Keyword: req.Keyword,
		Total:   res.Total,
		Took:    res.Took.String(),
		Hits:    hits,
		Cached:  false,
	}

	// Cache in Redis (5 min TTL).
	if h.cache != nil {
		if data, err := json.Marshal(resp); err == nil {
			if err := h.cache.Set(ctx, cacheKey, data, 5*time.Minute); err != nil {
				logger.Warn("cache set failed", zap.String("key", cacheKey), zap.Error(err))
			}
		}
	}

	logger.Info("search executed",
		zap.String("keyword", req.Keyword),
		zap.Uint64("total", res.Total),
		zap.Int("hits", len(hits)),
		zap.Duration("took", res.Took),
		zap.Bool("cached", false),
	)

	respgin.Success(c, resp)
}

// SearchIndexHandler indexes a document into the search engine.
func (h *Handlers) SearchIndexHandler(c *gin.Context) {
	if h.search == nil {
		respgin.FailWithCode(c, response.CodeServiceUnavail, "search engine is disabled", nil)
		return
	}

	var doc search.Doc
	if err := c.ShouldBindJSON(&doc); err != nil {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid document body", nil)
		return
	}

	if doc.ID == "" {
		doc.ID = idgen.ShortID()
	}

	ctx := c.Request.Context()
	if err := h.search.Index(ctx, doc); err != nil {
		logger.Warn("index failed", zap.String("id", doc.ID), zap.Error(err))
		respgin.FailWithCode(c, response.CodeInternal, "index failed", nil)
		return
	}

	// Invalidate search cache on new index.
	if h.cache != nil {
		_ = h.cache.Clear(ctx)
	}

	logger.Info("document indexed", zap.String("id", doc.ID), zap.String("type", doc.Type))

	respgin.Success(c, gin.H{
		"id":      doc.ID,
		"type":    doc.Type,
		"indexed": true,
	})
}

// ──────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────

// handleWithLimit applies connection + rate limiting and generates a request ID.
func (h *Handlers) handleWithLimit(ctx context.Context) (string, time.Duration, error) {
	start := time.Now()
	connKey := []byte("conn")

	if err := h.connRL.Acquire(ctx, connKey); err != nil {
		return "", 0, err
	}
	defer h.connRL.Release(connKey)

	if err := h.rl.Acquire(ctx, []byte("req")); err != nil {
		return "", 0, err
	}

	return idgen.ShortID(), time.Since(start), nil
}

// ──────────────────────────────────────────────
// Constructors for optional dependencies
// ──────────────────────────────────────────────

// NewRedisCache creates a Redis cache from the given address.
// Returns nil if addr is empty (Redis disabled).
func NewRedisCache(addr, password string, db int) cache.Cache[string, []byte] {
	if addr == "" {
		return nil
	}

	c, err := redisCache.New(&goredis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err != nil {
		logger.Warn("redis cache init failed, continuing without cache", zap.String("addr", addr), zap.Error(err))
		return nil
	}

	logger.Info("redis cache connected", zap.String("addr", addr), zap.Int("db", db))
	return c
}

// NewSearchEngine creates a search engine from the given config.
// If indexPath is empty, uses an in-memory index.
// Returns nil if search is disabled (indexPath is "disabled").
func NewSearchEngine(indexPath string, batchSize int, queryTimeout time.Duration) search.Engine {
	if indexPath == "disabled" {
		return nil
	}

	var eng search.Engine
	var err error
	if indexPath == "" || indexPath == "memory" {
		eng, err = bleve.NewMemory()
	} else {
		eng, err = bleve.NewDefault(indexPath)
	}
	if err != nil {
		logger.Warn("search engine init failed, continuing without search", zap.String("path", indexPath), zap.Error(err))
		return nil
	}

	logger.Info("search engine started", zap.String("path", indexPath))
	return eng
}

// ──────────────────────────────────────────────
// Retry handlers — demonstrates the retry framework
// ──────────────────────────────────────────────

// ErrSimulatedTransient is a simulated transient error for the retry demo.
var ErrSimulatedTransient = errors.New("simulated transient error")

// RetryHandler demonstrates the retry framework by simulating a flaky
// operation that fails a configurable number of times before succeeding.
//
// Request body:
//
//	{"fail_times": 3, "strategy": "exponential"}
//
// strategy can be "exponential", "fixed", or "none" (default: "exponential").
func (h *Handlers) RetryHandler(c *gin.Context) {
	var req struct {
		FailTimes int    `json:"fail_times"`
		Strategy  string `json:"strategy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid request body", nil)
		return
	}

	if req.FailTimes < 0 {
		req.FailTimes = 0
	}

	var attempts atomic.Int32
	var retryLog []map[string]any

	ctx := c.Request.Context()

	// Build retry options based on strategy.
	var opts []retry.Option
	opts = append(opts, retry.WithMaxAttempts(req.FailTimes+1))

	switch req.Strategy {
	case "fixed":
		opts = append(opts, retry.WithFixedInterval(200*time.Millisecond))
	case "none":
		opts = append(opts, retry.WithNoBackoff())
	default: // "exponential" or empty
		opts = append(opts, retry.WithExponentialBackoff(100*time.Millisecond, 5*time.Second, 2.0, true))
	}

	opts = append(opts,
		retry.WithRetryIf(func(err error) bool {
			return errors.Is(err, ErrSimulatedTransient)
		}),
		retry.WithOnRetry(func(attempt int, err error) {
			retryLog = append(retryLog, map[string]any{
				"attempt": attempt,
				"error":   err.Error(),
				"delay":   "backoff",
			})
		}),
		retry.WithOnSuccess(func(n int) {
			logger.Info("retry demo succeeded", zap.Int("attempts", n))
		}),
		retry.WithOnError(func(n int, err error) {
			logger.Warn("retry demo exhausted", zap.Int("attempts", n), zap.Error(err))
		}),
	)

	start := time.Now()
	err := retry.Do(ctx, func(ctx context.Context) error {
		n := attempts.Add(1)
		if int(n) <= req.FailTimes {
			return fmt.Errorf("attempt %d: %w", n, ErrSimulatedTransient)
		}
		return nil
	}, opts...)
	elapsed := time.Since(start)

	if err != nil {
		respgin.Success(c, gin.H{
			"success":     false,
			"attempts":    attempts.Load(),
			"strategy":    req.Strategy,
			"fail_times":  req.FailTimes,
			"elapsed_ms":  elapsed.Milliseconds(),
			"retry_log":   retryLog,
			"final_error": err.Error(),
		})
		return
	}

	respgin.Success(c, gin.H{
		"success":    true,
		"attempts":   attempts.Load(),
		"strategy":   req.Strategy,
		"fail_times": req.FailTimes,
		"elapsed_ms": elapsed.Milliseconds(),
		"retry_log":  retryLog,
	})
}

// ──────────────────────────────────────────────
// Circuit breaker handlers — demonstrates CB + retry integration
// ──────────────────────────────────────────────

// CircuitBreakerHandler demonstrates the circuit breaker with optional
// retry integration. The operation fails a configurable number of times
// before succeeding, and the circuit breaker tracks the failure rate.
//
// Request body:
//
//	{"fail_times": 3, "use_retry": true}
func (h *Handlers) CircuitBreakerHandler(c *gin.Context) {
	var req struct {
		FailTimes int  `json:"fail_times"`
		UseRetry  bool `json:"use_retry"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid request body", nil)
		return
	}

	var attempts atomic.Int32
	ctx := c.Request.Context()
	start := time.Now()

	op := func(ctx context.Context) error {
		n := attempts.Add(1)
		if int(n) <= req.FailTimes {
			return fmt.Errorf("attempt %d: simulated failure", n)
		}
		return nil
	}

	var err error
	if req.UseRetry {
		// Retry + circuit breaker: retry handles transient failures,
		// circuit breaker prevents cascading failures.
		err = h.cb.Execute(ctx, func(ctx context.Context) error {
			return retry.Do(ctx, op,
				retry.WithMaxAttempts(2),
				retry.WithNoBackoff(),
			)
		})
	} else {
		err = h.cb.Execute(ctx, op)
	}
	elapsed := time.Since(start)

	cbMetrics := h.cb.Metrics()

	respgin.Success(c, gin.H{
		"success":         err == nil,
		"attempts":        attempts.Load(),
		"fail_times":      req.FailTimes,
		"use_retry":       req.UseRetry,
		"elapsed_ms":      elapsed.Milliseconds(),
		"cb_state":        cbMetrics.State,
		"cb_failures":     cbMetrics.Failures,
		"cb_successes":    cbMetrics.Successes,
		"cb_failure_rate": cbMetrics.FailureRate,
		"cb_window":       cbMetrics.WindowLen,
		"error":           errString(err),
	})
}

// CircuitBreakerStateHandler returns the current circuit breaker state and metrics.
func (h *Handlers) CircuitBreakerStateHandler(c *gin.Context) {
	m := h.cb.Metrics()
	respgin.Success(c, m)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ──────────────────────────────────────────────
// Task queue handlers — demonstrates queue + scheduler
// ──────────────────────────────────────────────

// EnqueueTaskHandler submits a task to the queue.
//
// Request body:
//
//	{"message": "hello", "priority": 5, "max_retries": 3}
func (h *Handlers) EnqueueTaskHandler(c *gin.Context) {
	var req struct {
		Message     string `json:"message"`
		Priority    int    `json:"priority"`
		MaxRetries  int    `json:"max_retries"`
		Weight      int    `json:"weight"`      // resource cost (capacity units)
		JobID       string `json:"job_id"`      // group tasks by job
		Preemptible bool   `json:"preemptible"` // allow preemption
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid request body", nil)
		return
	}
	if req.MaxRetries <= 0 {
		req.MaxRetries = 3
	}
	if req.Weight <= 0 {
		req.Weight = 1
	}

	taskID := idgen.ShortID()
	payload, _ := json.Marshal(map[string]string{"message": req.Message})

	task := &queue.Task{
		ID:          taskID,
		Queue:       "demo-tasks",
		Priority:    req.Priority,
		Payload:     payload,
		MaxRetries:  req.MaxRetries,
		Weight:      req.Weight,
		JobID:       req.JobID,
		Preemptible: req.Preemptible,
	}

	if err := h.taskQueue.Enqueue(c.Request.Context(), task); err != nil {
		respgin.FailWithCode(c, response.CodeInternal, err.Error(), nil)
		return
	}

	// Query queue position (may be -1 if already dispatched).
	pos, _ := h.scheduler.Position(c.Request.Context(), taskID)

	respgin.Success(c, gin.H{
		"task_id":        taskID,
		"status":         "queued",
		"priority":       req.Priority,
		"weight":         req.Weight,
		"job_id":         req.JobID,
		"preemptible":    req.Preemptible,
		"queue_position": pos,
		"message":        req.Message,
	})
}

// ListTasksHandler returns current queue stats and scheduler stats.
func (h *Handlers) ListTasksHandler(c *gin.Context) {
	qStats, _ := h.taskQueue.Stats(c.Request.Context())
	sStats := h.scheduler.Stats()

	respgin.Success(c, gin.H{
		"queue":     qStats,
		"scheduler": sStats,
	})
}

// TaskStatsHandler returns detailed scheduler statistics.
func (h *Handlers) TaskStatsHandler(c *gin.Context) {
	respgin.Success(c, h.scheduler.Stats())
}

// CancelTaskHandler cancels a pending task.
func (h *Handlers) CancelTaskHandler(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		respgin.FailWithCode(c, response.CodeBadRequest, "task id is required", nil)
		return
	}

	if err := h.taskQueue.Cancel(c.Request.Context(), taskID); err != nil {
		respgin.FailWithCode(c, response.CodeNotFound, err.Error(), nil)
		return
	}

	respgin.Success(c, gin.H{
		"task_id": taskID,
		"status":  "canceled",
	})
}

// GetTaskHandler returns a task's details including progress and queue position.
func (h *Handlers) GetTaskHandler(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		respgin.FailWithCode(c, response.CodeBadRequest, "task id is required", nil)
		return
	}

	task, err := h.taskQueue.Get(c.Request.Context(), taskID)
	if err != nil {
		respgin.FailWithCode(c, response.CodeNotFound, err.Error(), nil)
		return
	}

	pos, _ := h.scheduler.Position(c.Request.Context(), taskID)

	respgin.Success(c, gin.H{
		"task":           task,
		"queue_position": pos,
	})
}

// TaskLogsHandler returns execution logs for a task.
func (h *Handlers) TaskLogsHandler(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		respgin.FailWithCode(c, response.CodeBadRequest, "task id is required", nil)
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	logs, err := h.scheduler.ListLogs(c.Request.Context(), taskID, limit)
	if err != nil {
		respgin.FailWithCode(c, response.CodeInternal, err.Error(), nil)
		return
	}

	respgin.Success(c, gin.H{
		"task_id": taskID,
		"logs":    logs,
		"count":   len(logs),
	})
}

// itoa is a small helper to avoid strconv import churn in hot paths.
func itoa(i int) string { return strconv.Itoa(i) }

// ──────────────────────────────────────────────
// Convert routes — type conversion + format interconversion
// ──────────────────────────────────────────────

func (h *Handlers) registerConvertRoutes(r *gin.RouterGroup) {
	r.POST("/convert/type", h.ConvertTypeHandler)
	r.POST("/convert/format", h.ConvertFormatHandler)
}

// ConvertTypeHandler demonstrates common/convert scalar type conversion.
//
// POST /api/v1/convert/type
// {"value": "42", "target": "int"}
// → {"ok": true, "result": 42}
//
// Supported targets: int, int64, uint, float64, bool, string, duration
func (h *Handlers) ConvertTypeHandler(c *gin.Context) {
	var req struct {
		Value  any    `json:"value"`
		Target string `json:"target"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid request body", nil)
		return
	}

	var result any
	var err error

	switch strings.ToLower(req.Target) {
	case "int":
		result, err = convert.ToInt(req.Value)
	case "int64":
		result, err = convert.ToInt64(req.Value)
	case "uint":
		result, err = convert.ToUint(req.Value)
	case "float64":
		result, err = convert.ToFloat64(req.Value)
	case "bool":
		result, err = convert.ToBool(req.Value)
	case "string":
		result, err = convert.ToString(req.Value)
	case "duration":
		result, err = convert.ToDuration(req.Value)
	default:
		respgin.FailWithCode(c, response.CodeBadRequest, "unsupported target type", nil)
		return
	}

	if err != nil {
		respgin.Success(c, gin.H{
			"ok":     false,
			"error":  err.Error(),
			"target": req.Target,
		})
		return
	}

	respgin.Success(c, gin.H{
		"ok":     true,
		"result": result,
		"target": req.Target,
	})
}

// ConvertFormatHandler demonstrates JSON/YAML/TOML interconversion.
//
// POST /api/v1/convert/format
// {"data": "{\"key\":\"value\"}", "from": "json", "to": "yaml"}
// → {"ok": true, "result": "key: value\n"}
//
// Supported formats: json, yaml, toml
func (h *Handlers) ConvertFormatHandler(c *gin.Context) {
	var req struct {
		Data string `json:"data"`
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid request body", nil)
		return
	}

	from := convert.Format(strings.ToLower(req.From))
	to := convert.Format(strings.ToLower(req.To))

	result, err := convert.Convert(from, to, []byte(req.Data))
	if err != nil {
		respgin.Success(c, gin.H{
			"ok":    false,
			"error": err.Error(),
			"from":  req.From,
			"to":    req.To,
		})
		return
	}

	respgin.Success(c, gin.H{
		"ok":     true,
		"result": string(result),
		"from":   req.From,
		"to":     req.To,
	})
}

// ──────────────────────────────────────────────
// Validate routes — struct-tag-driven validation
// ──────────────────────────────────────────────

func (h *Handlers) registerValidateRoutes(r *gin.RouterGroup) {
	r.POST("/validate", h.ValidateHandler)
	r.POST("/validate/tag", h.ValidateTagHandler)
}

// validateUserForm is a demo struct showing various validation rules.
type validateUserForm struct {
	Name     string `json:"name"     validate:"required,min=2,max=50"`
	Email    string `json:"email"    validate:"required,email"`
	Age      int    `json:"age"      validate:"gte=18,lte=120"`
	Role     string `json:"role"     validate:"oneof=admin user guest"`
	Password string `json:"password" validate:"required,min=8"`
	Confirm  string `json:"confirm"  validate:"eqfield=Password"`
	Website  string `json:"website"  validate:"url"`
} // validate:"required,email" — demonstrates struct tag validation

// ValidateHandler validates a user registration form using common/validate.
//
// POST /api/v1/validate
// {"name":"Alice","email":"alice@example.com","age":25,"role":"admin",
//
//	"password":"secure123","confirm":"secure123","website":"https://alice.dev"}
//
// → {"valid": true}
//
// Invalid input → {"valid": false, "errors": [...]}
func (h *Handlers) ValidateHandler(c *gin.Context) {
	var form validateUserForm
	if err := c.ShouldBindJSON(&form); err != nil {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid request body: "+err.Error(), nil)
		return
	}

	err := validate.Validate(form)
	if err == nil {
		respgin.Success(c, gin.H{
			"valid": true,
			"form":  form,
		})
		return
	}

	errs, ok := err.(validate.Errors)
	if !ok {
		respgin.Success(c, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	// Build a clean error list for the API response.
	type fieldErr struct {
		Field   string `json:"field"`
		Rule    string `json:"rule"`
		Message string `json:"message"`
	}
	var feList []fieldErr
	for _, fe := range errs {
		feList = append(feList, fieldErr{
			Field:   fe.Field,
			Rule:    fe.Rule,
			Message: fe.Message,
		})
	}

	respgin.Success(c, gin.H{
		"valid":  false,
		"errors": feList,
		"count":  len(feList),
	})
}

// ValidateTagHandler validates a single value against a tag string.
//
// POST /api/v1/validate/tag
// {"value": "test@example.com", "tag": "required,email"}
// → {"valid": true}
func (h *Handlers) ValidateTagHandler(c *gin.Context) {
	var req struct {
		Value any    `json:"value"`
		Tag   string `json:"tag"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid request body", nil)
		return
	}

	err := validate.ValidateWithTag(req.Value, req.Tag)
	if err == nil {
		respgin.Success(c, gin.H{
			"valid": true,
		})
		return
	}

	if fe, ok := err.(validate.FieldError); ok {
		respgin.Success(c, gin.H{
			"valid":   false,
			"rule":    fe.Rule,
			"message": fe.Message,
		})
		return
	}

	respgin.Success(c, gin.H{
		"valid": false,
		"error": err.Error(),
	})
}

// ──────────────────────────────────────────────
// Response routes — unified response envelope + error handling demo
// ──────────────────────────────────────────────

func (h *Handlers) registerResponseRoutes(r *gin.RouterGroup) {
	r.GET("/response/success", h.ResponseSuccessHandler)
	r.GET("/response/error/:code", h.ResponseErrorHandler)
	r.POST("/response/user", h.ResponseCreateUserHandler)
	r.GET("/response/page", h.ResponsePageHandler)
	r.GET("/response/panic", h.ResponsePanicHandler)
}

// ResponseSuccessHandler demonstrates the unified success envelope.
//
// GET /api/v1/response/success
// → {"code":200,"msg":"success","data":{"app":"ling-base-demo","version":"..."}}
func (h *Handlers) ResponseSuccessHandler(c *gin.Context) {
	respgin.Success(c, gin.H{
		"app":     "ling-base-demo",
		"version": version.Version,
		"time":    time.Now().Format(time.RFC3339),
	})
}

// ResponseErrorHandler demonstrates unified error envelopes with i18n keys.
//
// GET /api/v1/response/error/not_found   → 404 {"code":1001,"msg":"common.not_found","error":"NOT_FOUND",...}
// GET /api/v1/response/error/bad_request → 400 {"code":1000,"msg":"common.invalid_params","error":"BAD_REQUEST",...}
// GET /api/v1/response/error/forbidden   → 403 {"code":1003,"msg":"common.forbidden","error":"FORBIDDEN",...}
// GET /api/v1/response/error/unauthorized→ 401 {"code":1002,"msg":"common.unauthorized","error":"UNAUTHORIZED",...}
// GET /api/v1/response/error/conflict    → 409 {"code":1004,"msg":"common.conflict","error":"CONFLICT",...}
// GET /api/v1/response/error/rate_limited→ 429 {"code":1005,"msg":"common.rate_limited","error":"RATE_LIMITED",...}
// GET /api/v1/response/error/internal    → 500 {"code":2000,"msg":"common.internal_error","error":"INTERNAL",...}
func (h *Handlers) ResponseErrorHandler(c *gin.Context) {
	codeStr := c.Param("code")

	var ae *response.AppError
	switch strings.ToLower(codeStr) {
	case "not_found", "404":
		ae = response.Err(response.CodeNotFound)
	case "bad_request", "400":
		ae = response.Err(response.CodeBadRequest)
	case "forbidden", "403":
		ae = response.Err(response.CodeForbidden)
	case "unauthorized", "401":
		ae = response.Err(response.CodeUnauthorized)
	case "conflict", "409":
		ae = response.Err(response.CodeConflict)
	case "rate_limited", "429":
		ae = response.Err(response.CodeRateLimited)
	case "validation":
		ae = response.NewI18n(response.CodeValidation, response.KeyInvalidParams)
	case "quota":
		ae = response.Err(response.CodeQuotaExceeded).
			WithDetails(map[string]any{"limit": 100, "used": 100})
	case "timeout":
		ae = response.WrapErr(response.CodeUpstreamTimeout, errors.New("upstream took 30s"))
	case "service_unavailable":
		ae = response.Err(response.CodeServiceUnavail)
	case "internal", "500":
		ae = response.Wrap(response.CodeInternal, "simulated internal error", errors.New("db connection lost"))
	default:
		ae = response.Newf(response.CodeBadRequest, "unknown error code %q", codeStr)
	}

	respgin.FailAppError(c, ae)
}

// responseDemoUser is a demo user struct for the create-user endpoint.
type responseDemoUser struct {
	Name     string `json:"name"     validate:"required,min=2,max=50"`
	Email    string `json:"email"    validate:"required,email"`
	Age      int    `json:"age"      validate:"gte=0,lte=150"`
	Password string `json:"password" validate:"required,min=8"`
} // validate:"required,email" — demonstrates response + validate integration

// ResponseCreateUserHandler demonstrates response + validate integration.
// Uses common/validate for input validation and common/response for output.
//
// POST /api/v1/response/user
// {"name":"Alice","email":"alice@example.com","age":25,"password":"secure123"}
// → {"code":200,"msg":"common.created","data":{"id":"...","name":"Alice",...}}
//
// Invalid input → 400 error envelope with validation details
func (h *Handlers) ResponseCreateUserHandler(c *gin.Context) {
	var user responseDemoUser
	if err := c.ShouldBindJSON(&user); err != nil {
		respgin.FailAppError(c, response.NewI18n(response.CodeBadRequest, response.KeyInvalidBody))
		return
	}

	// Validate using common/validate
	if err := validate.Validate(user); err != nil {
		// Convert validation errors to response details
		details := make(map[string]any)
		if errs, ok := err.(validate.Errors); ok {
			fields := make([]map[string]string, 0, len(errs))
			for _, fe := range errs {
				fields = append(fields, map[string]string{
					"field":   fe.Field,
					"rule":    fe.Rule,
					"message": fe.Message,
				})
			}
			details["validation_errors"] = ""
			details["fields"] = fields
		}

		ae := response.Err(response.CodeValidation).
			WithDetails(details)
		respgin.FailAppError(c, ae)
		return
	}

	// Simulate user creation
	userID := idgen.ShortID()

	respgin.Created(c, gin.H{
		"id":         userID,
		"name":       user.Name,
		"email":      user.Email,
		"age":        user.Age,
		"created_at": time.Now().Format(time.RFC3339),
	})
}

// ResponsePageHandler demonstrates paginated responses.
//
// GET /api/v1/response/page?page=1&size=5
// → {"code":200,"msg":"success","data":{"list":[...],"total":42,"page":1,"size":5,"total_page":9}}
func (h *Handlers) ResponsePageHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	// Simulate a dataset
	total := int64(42)
	all := make([]gin.H, 0, total)
	for i := 1; i <= int(total); i++ {
		all = append(all, gin.H{
			"id":   i,
			"name": fmt.Sprintf("item-%d", i),
		})
	}

	// Slice for the requested page
	start := (page - 1) * size
	end := start + size
	if start > len(all) {
		start = len(all)
	}
	if end > len(all) {
		end = len(all)
	}
	list := all[start:end]

	pageData := response.NewPage(list, total, page, size)
	respgin.Success(c, pageData)
}

// ResponsePanicHandler demonstrates the panic recovery middleware.
// This handler always panics; the Recovery middleware catches it and
// returns a 500 error envelope.
//
// GET /api/v1/response/panic
// → 500 {"code":2000,"msg":"panic: simulated panic","error":"INTERNAL",...}
func (h *Handlers) ResponsePanicHandler(c *gin.Context) {
	panic("simulated panic for demo purposes")
}

// ──────────────────────────────────────────────
// Notification routes — unified notification dispatch + inbox
// ──────────────────────────────────────────────

func (h *Handlers) registerNotificationRoutes(r *gin.RouterGroup) {
	r.POST("/notify", h.NotifyHandler)
	r.POST("/notify/inbox", h.NotifyInboxHandler)
	r.GET("/notify/inbox/list", h.NotifyInboxListHandler)
	r.GET("/notify/inbox/unread", h.NotifyInboxUnreadHandler)
	r.POST("/notify/inbox/:id/read", h.NotifyInboxMarkReadHandler)
	r.POST("/notify/inbox/read-all", h.NotifyInboxMarkAllReadHandler)
	r.DELETE("/notify/inbox/:id", h.NotifyInboxDeleteHandler)
	r.GET("/notify/channels", h.NotifyChannelsHandler)
}

// NotifyHandler dispatches a notification through all enabled channels
// of the requested type. This demonstrates the unified Dispatcher with
// multi-channel failover.
//
// POST /api/v1/notify
// {"type":"inbox","to":"user-1","subject":"Welcome","body":"Hello!"}
// → {"code":200,"msg":"common.success","data":{"type":"inbox","to":"user-1","status":"sent"}}
func (h *Handlers) NotifyHandler(c *gin.Context) {
	var req struct {
		Type    string `json:"type"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid request body", nil)
		return
	}

	msgType := notification.MessageType(req.Type)
	switch msgType {
	case notification.TypeEmail, notification.TypeSMS, notification.TypeIM,
		notification.TypeWebhook, notification.TypeInbox:
		// valid type
	default:
		respgin.FailWithCode(c, response.CodeBadRequest,
			"unsupported type: "+req.Type+" (email/sms/im/webhook/inbox)", nil)
		return
	}

	msg := notification.Message{
		Type:    msgType,
		To:      req.To,
		Subject: req.Subject,
		Body:    req.Body,
		Title:   req.Subject,
		Content: req.Body,
		UserID:  req.To,
	}

	if err := h.notifDispatcher.Send(c.Request.Context(), msg); err != nil {
		respgin.FailWithCode(c, response.CodeInternal, err.Error(), nil)
		return
	}

	respgin.Success(c, gin.H{
		"type":   req.Type,
		"to":     req.To,
		"status": "sent",
	})
}

// NotifyInboxHandler sends an in-app inbox notification to a user.
//
// POST /api/v1/notify/inbox
// {"user_id":"user-1","title":"Welcome","content":"Hello!","action_url":"https://..."}
// → {"code":200,"msg":"common.success","data":{"status":"sent"}}
func (h *Handlers) NotifyInboxHandler(c *gin.Context) {
	var req struct {
		UserID      string `json:"user_id"`
		Title       string `json:"title"`
		Content     string `json:"content"`
		ActionURL   string `json:"action_url"`
		ActionLabel string `json:"action_label"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid request body", nil)
		return
	}

	if req.UserID == "" || req.Title == "" {
		respgin.FailWithCode(c, response.CodeBadRequest, "user_id and title are required", nil)
		return
	}

	err := h.inboxStore.Create(inbox.Message{
		UserID:      req.UserID,
		Title:       req.Title,
		Content:     req.Content,
		ActionURL:   req.ActionURL,
		ActionLabel: req.ActionLabel,
	})
	if err != nil {
		respgin.FailWithCode(c, response.CodeInternal, err.Error(), nil)
		return
	}

	respgin.Success(c, gin.H{"status": "sent"})
}

// NotifyInboxListHandler lists inbox messages for a user with pagination.
//
// GET /api/v1/notify/inbox/list?user_id=user-1&page=1&size=10&filter=unread
func (h *Handlers) NotifyInboxListHandler(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		respgin.FailWithCode(c, response.CodeBadRequest, "user_id is required", nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	filter := c.DefaultQuery("filter", "all")
	titleKeyword := c.Query("title")
	contentKeyword := c.Query("content")

	if !inbox.IsValidFilter(filter) {
		respgin.FailWithCode(c, response.CodeBadRequest, "invalid filter (all/unread/read)", nil)
		return
	}

	result, err := h.inboxStore.List(userID, page, size, filter, titleKeyword, contentKeyword, time.Time{}, time.Time{})
	if err != nil {
		respgin.FailWithCode(c, response.CodeInternal, err.Error(), nil)
		return
	}

	respgin.Success(c, result)
}

// NotifyInboxUnreadHandler returns the unread message count for a user.
//
// GET /api/v1/notify/inbox/unread?user_id=user-1
func (h *Handlers) NotifyInboxUnreadHandler(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		respgin.FailWithCode(c, response.CodeBadRequest, "user_id is required", nil)
		return
	}

	count, err := h.inboxStore.UnreadCount(userID)
	if err != nil {
		respgin.FailWithCode(c, response.CodeInternal, err.Error(), nil)
		return
	}

	respgin.Success(c, gin.H{"unread": count})
}

// NotifyInboxMarkReadHandler marks a specific inbox message as read.
//
// POST /api/v1/notify/inbox/:id/read?user_id=user-1
func (h *Handlers) NotifyInboxMarkReadHandler(c *gin.Context) {
	userID := c.Query("user_id")
	msgID := c.Param("id")
	if userID == "" || msgID == "" {
		respgin.FailWithCode(c, response.CodeBadRequest, "user_id and id are required", nil)
		return
	}

	if err := h.inboxStore.MarkRead(userID, msgID); err != nil {
		respgin.FailWithCode(c, response.CodeNotFound, err.Error(), nil)
		return
	}

	respgin.Success(c, gin.H{"id": msgID, "status": "read"})
}

// NotifyInboxMarkAllReadHandler marks all inbox messages as read for a user.
//
// POST /api/v1/notify/inbox/read-all?user_id=user-1
func (h *Handlers) NotifyInboxMarkAllReadHandler(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		respgin.FailWithCode(c, response.CodeBadRequest, "user_id is required", nil)
		return
	}

	if err := h.inboxStore.MarkAllRead(userID); err != nil {
		respgin.FailWithCode(c, response.CodeInternal, err.Error(), nil)
		return
	}

	respgin.Success(c, gin.H{"status": "all_read"})
}

// NotifyInboxDeleteHandler deletes a specific inbox message.
//
// DELETE /api/v1/notify/inbox/:id?user_id=user-1
func (h *Handlers) NotifyInboxDeleteHandler(c *gin.Context) {
	userID := c.Query("user_id")
	msgID := c.Param("id")
	if userID == "" || msgID == "" {
		respgin.FailWithCode(c, response.CodeBadRequest, "user_id and id are required", nil)
		return
	}

	if err := h.inboxStore.Delete(userID, msgID); err != nil {
		respgin.FailWithCode(c, response.CodeNotFound, err.Error(), nil)
		return
	}

	respgin.Success(c, gin.H{"id": msgID, "status": "deleted"})
}

// NotifyChannelsHandler returns the list of registered notification channels.
//
// GET /api/v1/notify/channels
func (h *Handlers) NotifyChannelsHandler(c *gin.Context) {
	channels := h.notifDispatcher.Channels()
	result := make([]gin.H, 0, len(channels))
	for _, ch := range channels {
		result = append(result, gin.H{
			"name":    ch.Name(),
			"type":    string(ch.Type()),
			"enabled": ch.Enabled(),
		})
	}
	respgin.Success(c, gin.H{"channels": result})
}
