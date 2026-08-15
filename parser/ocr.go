package parser

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// OCRProvider is the interface that cloud OCR backends implement.
// Each provider calls a real cloud API to extract text from image bytes.
type OCRProvider interface {
	// Name returns the provider identifier (e.g. "aliyun", "qcloud", "google").
	Name() string
	// Recognize sends image bytes to the cloud OCR API and returns extracted text.
	Recognize(ctx context.Context, imageBytes []byte, opts *OCROptions) (string, error)
}

// OCROptions controls provider-specific behavior (language hints, etc.).
type OCROptions struct {
	// Language is a BCP-47 or provider-specific language hint (e.g. "zh", "en", "auto").
	Language string
	// Extra allows passing provider-specific parameters not covered by the struct.
	Extra map[string]any
}

// ocrProviderRegistry holds the global OCR provider.
var (
	ocrProviderMu       sync.RWMutex
	ocrProvider         OCRProvider
	ocrProviderByDriver = make(map[string]OCRProvider)
)

// RegisterOCRProvider registers an OCR provider under the given driver name.
// The last registration wins. Passing nil is a no-op.
func RegisterOCRProvider(driver string, p OCRProvider) {
	if p == nil {
		return
	}
	ocrProviderMu.Lock()
	defer ocrProviderMu.Unlock()
	ocrProviderByDriver[strings.ToLower(strings.TrimSpace(driver))] = p
}

// SetOCRProvider sets the active global OCR provider.
// Passing nil clears the active provider (OCR will return ErrUnsupportedFileType).
func SetOCRProvider(p OCRProvider) {
	ocrProviderMu.Lock()
	defer ocrProviderMu.Unlock()
	ocrProvider = p
}

// SetOCRProviderByDriver selects an already-registered provider by driver name
// as the active OCR provider. Returns an error if no provider is registered
// under that name.
func SetOCRProviderByDriver(driver string) error {
	ocrProviderMu.RLock()
	p, ok := ocrProviderByDriver[strings.ToLower(strings.TrimSpace(driver))]
	ocrProviderMu.RUnlock()
	if !ok {
		return fmt.Errorf("ocr provider %q not registered", driver)
	}
	SetOCRProvider(p)
	return nil
}

// GetOCRProvider returns the currently active OCR provider, or nil if none is set.
func GetOCRProvider() OCRProvider {
	ocrProviderMu.RLock()
	defer ocrProviderMu.RUnlock()
	return ocrProvider
}

// RegisteredOCRDrivers returns the names of all registered OCR providers.
func RegisteredOCRDrivers() []string {
	ocrProviderMu.RLock()
	defer ocrProviderMu.RUnlock()
	out := make([]string, 0, len(ocrProviderByDriver))
	for k := range ocrProviderByDriver {
		out = append(out, k)
	}
	return out
}

// OCRParser implements the Parser interface for image files using a cloud OCR provider.
// If no OCR provider is registered, Parse returns ErrUnsupportedFileType.
type OCRParser struct {
	// Language is a language hint passed to the OCR provider.
	Language string
	// Driver overrides the global provider for this parser instance.
	// If nil, the global provider (set via SetOCRProvider) is used.
	Driver OCRProvider
}

func (p *OCRParser) Provider() string { return "ocr" }

func (p *OCRParser) SupportedTypes() []string {
	return []string{FileTypePNG, FileTypeJPG, FileTypeJPEG, FileTypeWEBP, FileTypeGIF, FileTypeBMP, FileTypeTIFF, FileTypeTIF}
}

func (p *OCRParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	if req == nil {
		return nil, ErrEmptyInput
	}
	data, fileName, err := readRequestBytes(req)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, ErrEmptyInput
	}

	// Decode to ensure it is a supported image and to guard against invalid data.
	_, _, err = image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Select provider: instance-level > global.
	provider := p.Driver
	if provider == nil {
		provider = GetOCRProvider()
	}
	if provider == nil {
		return nil, fmt.Errorf("no OCR provider registered: %w", ErrUnsupportedFileType)
	}

	ocrOpts := &OCROptions{
		Language: strings.TrimSpace(p.Language),
	}
	if ocrOpts.Language == "" {
		ocrOpts.Language = "auto"
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	text, err := provider.Recognize(ctx, data, ocrOpts)
	if err != nil {
		return nil, fmt.Errorf("ocr provider %s: %w", provider.Name(), err)
	}
	text = strings.TrimSpace(text)
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	ft := req.FileType
	if ft == "" {
		ft = DetectFileType(req)
	}

	return &ParseResult{
		FileType: ft,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}
