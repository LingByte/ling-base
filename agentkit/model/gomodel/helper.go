package gomodel

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// MIME type lookup tables for fast access
var (
	mimeExtMap = map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".bmp":  "image/bmp",
		".svg":  "image/svg+xml",
		".heic": "image/heic",
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
		".webm": "video/webm",
		".mkv":  "video/x-matroska",
		".avi":  "video/x-msvideo",
		".txt":  "text/plain",
		".log":  "text/plain",
		".md":   "text/markdown",
		".json": "application/json",
		".yaml": "application/x-yaml",
		".yml":  "application/x-yaml",
		".xml":  "application/xml",
	}

	mimeAliasMap = map[string]string{
		"image/jpg":   "image/jpeg",
		"image/pjpeg": "image/jpeg",
		"image/x-png": "image/png",
		"video/mov":   "video/quicktime",
	}

	mimeCache   = make(map[string]string, 100)
	mimeCacheMu sync.RWMutex
)

func sanitizeForGemini(mt string) string {
	mt = strings.ToLower(strings.TrimSpace(mt))
	mt = stripMIMEParameters(mt)
	switch mt {
	case "image/jpeg", "image/png", "image/webp", "image/gif",
		"video/mp4", "video/quicktime", "video/webm":
		return mt
	case "image/jpg":
		return "image/jpeg"
	case "image/x-png":
		return "image/png"
	case "video/mov":
		return "video/quicktime"
	}
	return ""
}

func normalizeMIME(name, m string) string {
	mimeCacheMu.RLock()
	cached, ok := mimeCache[m]
	mimeCacheMu.RUnlock()
	if ok {
		return cached
	}
	result := normalizeMIMEUncached(name, m)
	mimeCacheMu.Lock()
	mimeCache[m] = result
	mimeCacheMu.Unlock()
	return result
}

func normalizeMIMEUncached(name, m string) string {
	m = strings.ToLower(strings.TrimSpace(m))
	m = stripMIMEParameters(m)
	if m == "" {
		return mimeFromExtension(name)
	}
	if canonical, ok := mimeAliasMap[m]; ok {
		return canonical
	}
	if _, _, err := mime.ParseMediaType(m); err == nil {
		return m
	}
	return mimeFromExtension(name)
}

func stripMIMEParameters(value string) string {
	if i := strings.IndexByte(value, ';'); i >= 0 {
		return strings.TrimSpace(value[:i])
	}
	return value
}

func mimeFromExtension(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return ""
	}
	if m, ok := mimeExtMap[ext]; ok {
		return m
	}
	if m := mime.TypeByExtension(ext); m != "" {
		return stripMIMEParameters(m)
	}
	return ""
}

func isTextMIME(m string) bool {
	m = strings.ToLower(strings.TrimSpace(m))
	m = stripMIMEParameters(m)
	return strings.HasPrefix(m, "text/")
}

func combinePromptWithFiles(base string, files []File) string {
	var b strings.Builder
	b.WriteString(base)
	for _, f := range files {
		mt := normalizeMIME(f.Name, f.MIME)
		if isTextMIME(mt) {
			b.WriteString("\n\n--- ")
			b.WriteString(f.Name)
			b.WriteString(" ---\n")
			b.Write(f.Data)
		}
	}
	return b.String()
}

func isImageOrVideoMIME(m string) bool {
	m = strings.ToLower(strings.TrimSpace(m))
	m = stripMIMEParameters(m)
	return strings.HasPrefix(m, "image/") || strings.HasPrefix(m, "video/")
}

// Ensure fmt is used.
var _ = fmt.Sprintf

// vertexProviderFactory is the factory signature for creating Vertex LLMs.
type vertexProviderFactory func(context.Context, string, string, string, string) (Agent, error)

// newLLMProvider is the testable core of NewLLMProvider. It accepts a
// vertexProviderFactory so tests can inject a fake Vertex backend.
func newLLMProvider(
	ctx context.Context,
	provider string,
	model string,
	promptPrefix string,
	newVertex vertexProviderFactory,
) (Agent, error) {
	var agent Agent
	var err error

	switch provider {
	case "openai":
		agent = NewOpenAILLM(model, promptPrefix)
	case "gemini", "google":
		agent, err = NewGeminiLLM(ctx, model, promptPrefix)
	case "vertex", "vertexai", "vertex-ai":
		project := strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
		location := strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_LOCATION"))
		if location == "" {
			location = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_REGION"))
		}
		agent, err = newVertex(ctx, model, promptPrefix, project, location)
	case "ollama":
		agent, err = NewOllamaLLM(model, promptPrefix)
	case "anthropic", "claude":
		agent = NewAnthropicLLM(model, promptPrefix)
	case "openrouter":
		agent = NewOpenRouterLLM(model, promptPrefix)
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	if err != nil {
		return nil, err
	}

	return TryCreateCachedLLM(agent), nil
}
