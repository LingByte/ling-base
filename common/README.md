# common

Root shared-utilities package for ling-base, plus 49 sub-packages covering
caching, crypto, logging, rate limiting, and more.

## Structure

```
common/
├── base.go           # BaseModel: GORM entity with snowflake ID + audit fields
├── env.go            # GetEnv / GetEnvXxx with TTL+LRU cache
├── dbs.go            # InitDatabase: GORM connection factory
├── dbs_mysql.go      # MySQL driver    (build tag: mysql)
├── dbs_pg.go         # PostgreSQL driver (build tag: pg)
├── dbs_sqlite.go     # SQLite fallback  (default build)
├── content_type.go   # File extension -> MIME type map
├── file_type.go      # File category constants (image/audio/media/file)
├── files.go          # File hash, size, save helpers
├── array.go          # Generic Join[T] helper
├── strings.go        # Zero-copy string <-> []byte conversions
├── geo.go            # Haversine distance between two coordinates
├── signals.go        # Signal/event handler registry
└── snowflake.go      # Deprecated snowflake aliases (use common/idgen)
```

## Key Types

```go
// BaseModel is embedded by all GORM entities.
type BaseModel struct {
    ID        uint           `gorm:"primaryKey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
    CreateBy  string
    UpdateBy  string
    Remark    string
}

// InitDatabase creates a GORM DB from driver + DSN (falls back to env vars).
func InitDatabase(logWrite io.Writer, driver, dsn string) (*gorm.DB, error)

// GetEnv reads an env var with TTL+LRU caching.
func GetEnv(key string) string
```

## Sub-packages (49)

| Package          | Description                                                  |
|------------------|--------------------------------------------------------------|
| audioutil        | WAV/MP3 audio read, write, and decode utilities              |
| barcode          | 1D/2D barcode generation (Code128, EAN, PDF417, DataMatrix)  |
| bloom            | Unified Bloom-filter interface (in-memory + distributed)     |
| cache            | Generic Cache[K,V] interface with multiple backends          |
| captcha          | CAPTCHA generation                                           |
| circuitbreaker   | Thread-safe circuit breaker (Closed/Open/Half-Open)          |
| compress         | Gzip and Zstd compression utilities                         |
| config           | Multi-format config loader (YAML + .env with env overrides) |
| constants        | Shared constants and env-var keys                           |
| convert          | Type-safe conversions and JSON/TOML/YAML interconversion     |
| cron             | Cron expression parsing and next-fire-time calculation       |
| crypto           | AES (GCM/CBC), RSA, and JWT signing utilities                |
| eventbus         | In-memory pub/sub event bus (sync + async dispatch)          |
| geoip            | IP geolocation lookup (domestic + international APIs)        |
| hash             | MD5, SHA-1/256/512, HMAC-SHA256/512                          |
| i18n             | Internationalization helpers                                |
| idgen            | Snowflake + UUID v4 ID generation                            |
| imageutil        | Image processing utilities                                  |
| jwtutil          | Reusable JWT authentication layer on common/crypto           |
| limiter          | Unified rate-limiting and concurrency-control interface      |
| lock             | Distributed lock interface                                  |
| logger           | Structured logging (zap-based) with Gin integration          |
| mathutil         | Math helpers: Clamp, Round, MinMax, Truncate                 |
| metrics          | Metrics helpers                                             |
| migration        | Database migration sources (filesystem, GORM migrator)       |
| netutil          | Port availability and IP address helpers                    |
| nltime           | Natural-language time expression parser                     |
| notification     | Notification channel abstraction                            |
| opentelemetry    | OpenTelemetry helpers                                       |
| parser           | ASR and other parsing utilities                             |
| passkey          | WebAuthn / Passkey server-side ceremony                     |
| password         | Password hashing helpers                                    |
| phone            | Phone number location lookup                                |
| pinyin           | Chinese pinyin conversion                                   |
| pool             | Connection pool utilities                                   |
| qrcode           | QR code encode/decode                                       |
| queue            | Capacity scheduler and queue utilities                      |
| random           | Cryptographically secure random (numbers, strings, bytes)   |
| response         | AppError and unified response helpers                       |
| retry            | Retry framework with backoff strategies                     |
| scheduler        | Distributed task scheduler with locking                     |
| search           | Search engine abstraction                                   |
| stats            | Statistics and archiving utilities                          |
| system           | System info (disk cache, etc.)                              |
| timeutil         | Time formatting, parsing, and range helpers                 |
| totp             | TOTP / HOTP for 2FA (RFC 6238 / 4226)                       |
| tracing          | Tracing attributes and helpers                              |
| validate         | Input validation helpers                                    |
| videoutil        | Video processing (ffmpeg command builders)                  |

## Quick Start

```go
import "github.com/LingByte/ling-base/common"

// Environment variables (cached)
dbHost := common.GetEnv("DB_HOST")
port := common.GetEnvInt("DB_PORT", 3306)

// GORM database
db, err := common.InitDatabase(nil, "mysql", "user:pass@tcp(127.0.0.1:3306)/db")

// Content type lookup
ct := common.GetContentType(".json") // "application/json"

// Snowflake ID (via idgen)
id := common.NextSnowflakeUint()
```
