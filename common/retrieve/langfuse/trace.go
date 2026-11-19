package langfuse

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	genOnce   sync.Once
	genClient *Client
)

// TraceGeneration records a single LLM call as a Langfuse trace (best-effort).
// No-op unless LANGFUSE_ENABLED=true and keys are set (or Config is passed via env).
func TraceGeneration(ctx context.Context, name, model, input, output string, err error) {
	c := generationClient()
	if c == nil {
		return
	}
	if name == "" {
		name = "llm-generation"
	}
	traceID := newEventID()
	meta := map[string]any{}
	if model != "" {
		meta["model"] = model
	}
	start := time.Now().UTC()
	in := map[string]any{"text": input}
	events := []ingestionEvent{newTraceEvent(traceID, name, in, meta)}
	out := map[string]any{"text": output}
	if err != nil {
		out["error"] = err.Error()
	}
	end := time.Now().UTC()
	events = append(events, newSpanEvent(traceID, "generation", start, end, meta, err))
	events = append(events, traceUpdateEvent(traceID, out, err))
	_ = c.Ingest(ctx, events)
}

func generationClient() *Client {
	genOnce.Do(func() {
		en := strings.TrimSpace(strings.ToLower(os.Getenv("LANGFUSE_ENABLED")))
		if en != "1" && en != "true" && en != "yes" && en != "on" {
			return
		}
		c, err := NewClient(Config{
			Enabled:   true,
			BaseURL:   os.Getenv("LANGFUSE_HOST"),
			PublicKey: os.Getenv("LANGFUSE_PUBLIC_KEY"),
			SecretKey: os.Getenv("LANGFUSE_SECRET_KEY"),
		})
		if err != nil || c == nil {
			return
		}
		genClient = c
	})
	return genClient
}
