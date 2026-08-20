# i18n

Unified internationalization module for Go: locale detection, translation
lookup with fallback, locale-specific formatting, and pluggable
machine-translation backends.

## Structure

```
i18n/
├── i18n.go              # Manager, Locale, Config, translation storage
├── context.go           # context.Context locale helpers
├── locale.go            # Accept-Language parsing, locale resolution
├── formatter.go         # number / currency / date / relative-time formatting
├── translator.go        # Translator interface (implementations in sub-modules)
├── translations/        # bundled JSON translation files
│   ├── messages.en.json
│   ├── messages.zh-CN.json
│   └── messages.zh-TW.json
├── gin/                 # optional Gin middleware (separate module)
│   ├── go.mod
│   └── middleware.go
└── mymemory/            # optional MyMemory MT backend (separate module)
    ├── go.mod
    └── translator.go
```

## Design: Core + Optional Sub-Modules

The core `i18n` package has **zero external dependencies** — it only needs
the Go standard library. Optional integrations live in separate Go modules:

```bash
# Core only (no HTTP framework, no external MT API)
go get github.com/LingByte/ling-base/i18n

# With Gin middleware
go get github.com/LingByte/ling-base/i18n/gin

# With MyMemory machine translation
go get github.com/LingByte/ling-base/i18n/mymemory
```

## Quick Start

### Basic Manager

```go
import "github.com/LingByte/ling-base/i18n"

manager := i18n.NewManager(&i18n.Config{
    DefaultLocale:    i18n.LocaleEn,
    SupportedLocales: []i18n.Locale{i18n.LocaleEn, i18n.LocaleZhCN, i18n.LocaleZhTW},
    FallbackLocale:   i18n.LocaleEn,
    TranslationsPath: "./translations", // load JSON files from directory
})

// Translate
msg := manager.T(i18n.LocaleZhCN, "common.success")
// → "成功"

// With printf-style args
msg = manager.T(i18n.LocaleEn, "hello", "World")
// → "Hello, World!"
```

### Loading Translation Files

Translation files are JSON `key → value` maps. The locale is extracted from
the filename suffix:

```
translations/
├── messages.en.json      → locale "en"
├── messages.zh-CN.json   → locale "zh-CN"
└── messages.zh-TW.json   → locale "zh-TW"
```

```go
// Load all files from a directory
manager.LoadTranslations("./translations")

// Load a single file
manager.LoadTranslationFile(i18n.LocaleJaJP, "./messages.ja-JP.json")

// Set a translation programmatically
manager.SetTranslation(i18n.LocaleEn, "custom.key", "Custom Value")
```

### Locale Detection

```go
// From Accept-Language header
locale := manager.ParseAcceptLanguage("zh-CN,zh;q=0.9,en;q=0.8")
// → i18n.LocaleZhCN

// From a raw locale string
locale := manager.ResolveLocale("zh_CN")
// → i18n.LocaleZhCN (underscore normalised)

// From a context.Context
ctx := i18n.WithLocale(context.Background(), i18n.LocaleZhCN)
locale := i18n.GetLocaleFromContext(ctx)
```

### Formatting

```go
f := i18n.NewFormatter(i18n.LocaleEn)

f.FormatNumber(1234.56, 2)           // "1,234.56"
f.FormatCurrency(1234.56, "USD")     // "$1,234.56"
f.FormatDate(time.Now(), "YYYY-MM-DD") // "2024-01-15"
f.FormatRelativeTime(pastTime)       // "2 hours ago"

fCN := i18n.NewFormatter(i18n.LocaleZhCN)
fCN.FormatCurrency(1234.56, "CNY")   // "¥1,234.56"
fCN.FormatRelativeTime(pastTime)     // "2 小时前"
```

### Gin Middleware

```go
import (
    "github.com/LingByte/ling-base/i18n"
    i18ngin "github.com/LingByte/ling-base/i18n/gin"
    "github.com/gin-gonic/gin"
)

manager := i18n.NewManager(&i18n.Config{
    SupportedLocales: []i18n.Locale{i18n.LocaleEn, i18n.LocaleZhCN},
    DefaultLocale:    i18n.LocaleEn,
    TranslationsPath: "./translations",
})

r := gin.New()
r.Use(i18ngin.Middleware(manager))

r.GET("/hello", func(c *gin.Context) {
    // Locale auto-detected from ?locale=, Accept-Language header, or cookie
    i18ngin.ResponseJSON(c, 200, "common.success", gin.H{"locale": i18ngin.GetLocale(c)})
})
```

Detection order: `?locale=` query param → `Accept-Language` header → `locale` cookie → default.

### Machine Translation (MyMemory)

```go
import (
    "github.com/LingByte/ling-base/i18n"
    "github.com/LingByte/ling-base/i18n/mymemory"
)

translator := mymemory.New("your-email@example.com") // email optional
manager := i18n.NewManager(&i18n.Config{
    Translator: translator,
})

result, err := manager.Translate("Hello", "en", "zh-CN")
// → "你好"
```

## API Reference

### Manager

| Method | Description |
|---|---|
| `NewManager(*Config) *Manager` | Create a manager |
| `T(locale, key, args...) string` | Translate with printf-style args |
| `GetTranslation(locale, key) string` | Lookup with fallback |
| `SetTranslation(locale, key, value)` | Set a single translation |
| `LoadTranslations(path) error` | Load JSON files from directory |
| `LoadTranslationFile(locale, path) error` | Load a single JSON file |
| `Translate(text, from, to) (string, error)` | Machine translation |
| `ParseAcceptLanguage(header) Locale` | Parse Accept-Language header |
| `ResolveLocale(raw) Locale` | Normalise a locale string |
| `GetDefaultLocale() Locale` | Get default locale |
| `GetSupportedLocales() []Locale` | Get supported locales |
| `IsSupportedLocale(locale) bool` | Check if supported |

### Formatter

| Method | Description |
|---|---|
| `NewFormatter(locale) *Formatter` | Create a formatter |
| `FormatNumber(n, decimals) string` | Locale-specific number |
| `FormatCurrency(amount, currency) string` | With currency symbol |
| `FormatDate(time, format) string` | Token-based date format |
| `FormatRelativeTime(time) string` | "2 hours ago" style |

### Context Helpers

| Function | Description |
|---|---|
| `WithLocale(ctx, locale) context.Context` | Attach locale to context |
| `GetLocaleFromContext(ctx) Locale` | Extract locale from context |

## Supported Locales

Built-in constants: `en`, `en-US`, `en-GB`, `zh-CN`, `zh-TW`, `ja-JP`,
`ko-KR`, `fr-FR`, `de-DE`, `es-ES`. Custom locales can be added via
`Config.SupportedLocales`.

## Testing

```bash
# Core (no external deps)
cd i18n && go test -cover

# Gin middleware
cd i18n/gin && go test -cover

# MyMemory translator
cd i18n/mymemory && go test -cover
```

Coverage:
- `i18n` (core): ~84%
- `i18n/gin`: ~94%
- `i18n/mymemory`: ~86%
