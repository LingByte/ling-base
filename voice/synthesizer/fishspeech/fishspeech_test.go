package synthesizer

import (
	"context"
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/synthesizer"
)

func TestNewFishSpeechConfigDefaults(t *testing.T) {
	c := NewFishSpeechConfig("fs-key", "ref123")
	if c.APIKey != "fs-key" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "fs-key")
	}
	if c.ReferenceID != "ref123" {
		t.Errorf("ReferenceID = %q, want %q", c.ReferenceID, "ref123")
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
	if c.Codec != "wav" {
		t.Errorf("Codec = %q, want %q", c.Codec, "wav")
	}
	if c.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", c.FrameDuration, "20ms")
	}
	if c.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", c.Timeout)
	}
	if c.Latency != "normal" {
		t.Errorf("Latency = %q, want %q", c.Latency, "normal")
	}
	if c.Version != "s1" {
		t.Errorf("Version = %q, want %q", c.Version, "s1")
	}
}

func TestNewFishSpeechConfigGetProvider(t *testing.T) {
	c := NewFishSpeechConfig("fs-key", "ref")
	if got := c.GetProvider(); got != base.ProviderFishSpeech {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderFishSpeech)
	}
}

func TestFishSpeechServiceProvider(t *testing.T) {
	svc := NewFishSpeechService(NewFishSpeechConfig("fs-key", "ref"))
	if got := svc.Provider(); got != base.ProviderFishSpeech {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderFishSpeech)
	}
}

func TestFishSpeechServiceFormat(t *testing.T) {
	svc := NewFishSpeechService(NewFishSpeechConfig("fs-key", "ref"))
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
	if f.FrameDuration != 20*time.Millisecond {
		t.Errorf("FrameDuration = %v, want 20ms", f.FrameDuration)
	}
}

func TestFishSpeechServiceCacheKey(t *testing.T) {
	svc := NewFishSpeechService(NewFishSpeechConfig("fs-key", "ref"))
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
	if !strings.HasPrefix(k1, "fishspeech.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "fishspeech.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".wav") {
		t.Errorf("CacheKey should end with %q, got %q", ".wav", k1)
	}
}

func TestFishSpeechServiceCacheKeyReflectsConfig(t *testing.T) {
	svc1 := NewFishSpeechService(NewFishSpeechConfig("fs-key", "ref1"))
	svc2 := NewFishSpeechService(NewFishSpeechConfig("fs-key", "ref2"))
	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when reference ID differs")
	}
}

func TestFishSpeechServiceClose(t *testing.T) {
	svc := NewFishSpeechService(NewFishSpeechConfig("fs-key", "ref"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestFishSpeechServiceSynthesizeEmptyAPIKey(t *testing.T) {
	svc := NewFishSpeechService(NewFishSpeechConfig("", "ref"))
	err := svc.Synthesize(context.Background(), base.HandlerFunc{}, "hello")
	if err == nil {
		t.Fatal("Synthesize with empty API key should return error")
	}
	if !strings.Contains(err.Error(), "FISHSPEECH_API_KEY") {
		t.Errorf("expected error mentioning FISHSPEECH_API_KEY, got %v", err)
	}
}

func TestFishSpeechServiceSynthesizeEmptyReferenceID(t *testing.T) {
	svc := NewFishSpeechService(NewFishSpeechConfig("fs-key", ""))
	err := svc.Synthesize(context.Background(), base.HandlerFunc{}, "hello")
	if err == nil {
		t.Fatal("Synthesize with empty reference ID should return error")
	}
	if !strings.Contains(err.Error(), "FISHSPEECH_REFERENCE_ID") {
		t.Errorf("expected error mentioning FISHSPEECH_REFERENCE_ID, got %v", err)
	}
}
