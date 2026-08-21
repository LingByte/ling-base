# ocr

Vendor-agnostic OCR (Optical Character Recognition) abstraction for
ling-base. Cloud OCR providers implement the `Provider` interface and
self-register via `RegisterProvider`.

The core package has zero external dependencies. Optional integrations
live in sub-modules.

## Structure

```
ocr/
├── ocr.go          # Core: Provider interface, Options, registry
├── aliyun/         # Aliyun OCR integration
├── aws/            # AWS Textract integration
├── azure/          # Azure Computer Vision integration
├── baidu/          # Baidu OCR integration
├── google/         # Google Cloud Vision integration
└── qcloud/         # Tencent Cloud OCR integration
```

## Key Types

```go
// Provider is the interface that cloud OCR backends implement.
type Provider interface {
    // Name returns the provider identifier (e.g. "aliyun", "qcloud").
    Name() string
    // Recognize sends image bytes to the cloud OCR API and returns text.
    Recognize(ctx context.Context, imageBytes []byte, opts *Options) (string, error)
}

// Options controls provider-specific behavior.
type Options struct {
    Language string         // BCP-47 or provider-specific hint ("zh", "en", "auto")
    Extra    map[string]any // provider-specific parameters
}
```

## Registry Functions

```go
// RegisterProvider registers an OCR provider under a driver name.
func RegisterProvider(driver string, p Provider)

// SetProvider sets the active global OCR provider.
func SetProvider(p Provider)

// SetProviderByDriver selects a registered provider by name.
func SetProviderByDriver(driver string) error

// GetProvider returns the currently active provider.
func GetProvider() Provider

// RegisteredDrivers returns all registered provider names.
func RegisteredDrivers() []string
```

## Quick Start

```go
import (
    "context"
    "github.com/LingByte/ling-base/ocr"
    "github.com/LingByte/ling-base/ocr/aliyun"
)

func main() {
    // Register and activate a provider
    p, err := aliyun.New(aliyun.Config{
        AccessKeyID:     "your-key-id",
        AccessKeySecret: "your-key-secret",
    })
    if err != nil {
        panic(err)
    }
    ocr.RegisterProvider("aliyun", p)
    ocr.SetProviderByDriver("aliyun")

    // Recognize text from an image
    imgBytes, _ := os.ReadFile("document.png")
    text, err := ocr.GetProvider().Recognize(
        context.Background(),
        imgBytes,
        &ocr.Options{Language: "zh"},
    )
    if err != nil {
        panic(err)
    }
    fmt.Println(text)
}
```
