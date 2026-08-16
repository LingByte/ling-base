package synthesizer

import (
	"context"
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/synthesizer"
)

func TestNewOpenAIConfigDefaults(t *testing.T) {
	c := NewOpenAIConfig("sk-test")
	if c.APIKey != "sk-test" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "sk-test")
	}
	if c.Model != "tts-1" {
		t.Errorf("Model = %q, want %q", c.Model, "tts-1")
	}
	if c.Voice != "alloy" {
		t.Errorf("Voice = %q, want %q", c.Voice, "alloy")
	}
	if c.Speed != 1.0 {
		t.Errorf("Speed = %v, want 1.0", c.Speed)
	}
	if c.SampleRate != 24000 {
		t.Errorf("SampleRate = %d, want 24000", c.SampleRate)
	}
	if c.Channels != 1 {
		t.Errorf("Channels = %d, want 1", c.Channels)
	}
	if c.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", c.BitDepth)
	}
	if c.Codec != "mp3" {
		t.Errorf("Codec = %q, want %q", c.Codec, "mp3")
	}
	if c.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", c.FrameDuration, "20ms")
	}
	if c.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", c.Timeout)
	}
	if c.BaseURL != "https://api.openai.com" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, "https://api.openai.com")
	}
}

func TestNewOpenAIConfigGetProvider(t *testing.T) {
	c := NewOpenAIConfig("sk-test")
	if got := c.GetProvider(); got != base.ProviderOpenAI {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderOpenAI)
	}
}

func TestOpenAIServiceProvider(t *testing.T) {
	svc := NewOpenAIService(NewOpenAIConfig("sk-test"))
	if got := svc.Provider(); got != base.ProviderOpenAI {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderOpenAI)
	}
}

func TestOpenAIServiceFormat(t *testing.T) {
	svc := NewOpenAIService(NewOpenAIConfig("sk-test"))
	f := svc.Format()
	if f.SampleRate != 24000 {
		t.Errorf("SampleRate = %d, want 24000", f.SampleRate)
	}
	if f.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", f.BitDepth)
	}
	if f.Channels != 1 {
		t.Errorf("Channels = %d, want 1", f.Channels)
	}
	if f.Codec != "mp3" {
		t.Errorf("Codec = %q, want %q", f.Codec, "mp3")
	}
	if f.FrameDuration != 20*time.Millisecond {
		t.Errorf("FrameDuration = %v, want 20ms", f.FrameDuration)
	}
}

func TestOpenAIServiceCacheKey(t *testing.T) {
	svc := NewOpenAIService(NewOpenAIConfig("sk-test"))
	k1 := svc.CacheKey("hello")
	k2 := svc.CacheKey("hello")
	k3 := svc.CacheKey("world")

	if k1 == "" {
		t.Fatal("CacheKey should not be empty")
	}
	if k1 != k2 {
		t.Errorf("CacheKey not deterministic: %q != %q", k1, k2)
	}
	if k1 == k3 {
		t.Errorf("CacheKey should differ for different text: %q == %q", k1, k3)
	}
	if !strings.HasPrefix(k1, "openai.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "openai.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".mp3") {
		t.Errorf("CacheKey should end with codec suffix %q, got %q", ".mp3", k1)
	}
}

func TestOpenAIServiceCacheKeyReflectsConfig(t *testing.T) {
	cfg1 := NewOpenAIConfig("sk-test")
	cfg1.Voice = "alloy"
	svc1 := NewOpenAIService(cfg1)

	cfg2 := NewOpenAIConfig("sk-test")
	cfg2.Voice = "echo"
	svc2 := NewOpenAIService(cfg2)

	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when voice differs")
	}
}

func TestOpenAIServiceClose(t *testing.T) {
	svc := NewOpenAIService(NewOpenAIConfig("sk-test"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestOpenAIServiceSynthesizeEmptyAPIKey(t *testing.T) {
	cfg := NewOpenAIConfig("")
	svc := NewOpenAIService(cfg)
	err := svc.Synthesize(context.Background(), base.HandlerFunc{}, "hello")
	if err == nil {
		t.Fatal("Synthesize with empty API key should return error")
	}
	if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("expected error mentioning OPENAI_API_KEY, got %v", err)
	}
}
