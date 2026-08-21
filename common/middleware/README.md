# middleware

HTTP middleware for the Gin framework: CORS, maintenance mode, rate limiting,
circuit breakers, request ID, logging, panic recovery, CSRF, and more.

## Structure

```
middleware/
├── api_version.go       # X-API-Version header validation and echo
├── auth_ratelimit.go    # Per-IP rate limiter for auth endpoints
├── breaker.go           # Circuit-breaker interface + factory
├── cors.go              # CORS middleware with configurable origins
├── db.go                # InjectDB: GORM DB into Gin context
├── error_handler.go     # Panic recovery -> structured AppError responses
├── ip_list.go           # IPList: parse IPs/CIDRs for allow/block checks
├── logger.go            # Access logging with slow-request detection
├── maintenance.go       # Maintenance mode middleware (env-driven)
├── oplog.go             # Operation-log dedup markers
├── recovery.go          # PanicRecovery: logs panics, returns 500
├── redis_rate.go        # Redis-backed distributed rate limiter
├── reqid.go             # RequestID: assigns/reuses X-Reqid header
├── security.go          # CSRF protection and security headers
└── timeout_circuit.go   # Per-request timeout + circuit breaker
```

## Key Types

```go
// Breaker is the circuit-breaker interface (satisfied by circuitbreaker pkg).
type Breaker interface {
    Allow() error
    MarkSuccess()
    MarkFailed()
}

// CORSConfig configures the CORS middleware.
type CORSConfig struct {
    AllowOrigins []string
    AllowMethods []string
    AllowHeaders []string
    // ...
}

// IPList holds parsed IP/CIDR entries for allow/block checks.
type IPList struct {
    Any  bool
    IPs  []net.IP
    Nets []*net.IPNet
}
```

## Quick Start

```go
import "github.com/LingByte/ling-base/middleware"

r := gin.New()

// Standard middleware chain (order matters)
r.Use(middleware.RequestIDMiddleware())
r.Use(middleware.LoggerMiddleware())
r.Use(middleware.PanicRecovery())
r.Use(middleware.ErrorHandler())
r.Use(middleware.APIVersionMiddleware())
r.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: []string{"https://example.com"},
}))
r.Use(middleware.InjectDB(db))

// Maintenance mode (driven by MAINTENANCE_MODE env var)
r.Use(middleware.MaintenanceMiddleware())

// Per-IP rate limiting for auth endpoints
auth := r.Group("/auth")
auth.Use(middleware.AuthRateLimitMiddleware())

// Timeout + circuit breaker for upstream calls
r.Use(middleware.TimeoutCircuit(middleware.TimeoutCircuitConfig{
    Timeout: 10 * time.Second,
    BreakerFactory: middleware.BreakerFactory{ ... },
}))
```
