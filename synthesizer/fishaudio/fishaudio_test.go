package synthesizer

import (
	"context"
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/synthesizer"
)

func TestNewFishAudioConfigDefaults(t *testing.T) {
	c := NewFishAudioConfig("fa-key", "ref123")
	if c.APIKey != "fa-key" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "fa-key")
	}
	if c.ReferenceID != "ref123" {
		t.Errorf("ReferenceID = %q, want %q", c.ReferenceID, "ref123")
	}
	if c.Model != "s1" {
		t.Errorf("Model = %q, want %q", c.Model, "s1")
	}
	if c.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", c.SampleRate)
	}
	if c.Channels != 1 {
		t.Errorf("Channels = %d, want 1", c.Channels)
	}
	if c.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", c.BitDepth)
	}
	if c.Format != "mp3" {
		t.Errorf("Format = %q, want %q", c.Format, "mp3")
	}
	if c.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", c.Temperature)
	}
	if c.TopP != 0.7 {
		t.Errorf("TopP = %v, want 0.7", c.TopP)
	}
	if c.Latency != "normal" {
		t.Errorf("Latency = %q, want %q", c.Latency, "normal")
	}
	if c.ChunkLength != 300 {
		t.Errorf("ChunkLength = %d, want 300", c.ChunkLength)
	}
	if !c.Normalize {
		t.Errorf("Normalize = %v, want true", c.Normalize)
	}
	if c.MPEGBitrate != 128 {
		t.Errorf("MPEGBitrate = %d, want 128", c.MPEGBitrate)
	}
	if c.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", c.Timeout)
	}
}

func TestNewFishAudioConfigGetProvider(t *testing.T) {
	c := NewFishAudioConfig("fa-key", "ref")
	if got := c.GetProvider(); got != base.ProviderFishAudio {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderFishAudio)
	}
}

func TestFishAudioServiceProvider(t *testing.T) {
	svc := NewFishAudioService(NewFishAudioConfig("fa-key", "ref"))
	if got := svc.Provider(); got != base.ProviderFishAudio {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderFishAudio)
	}
}

func TestFishAudioServiceFormat(t *testing.T) {
	svc := NewFishAudioService(NewFishAudioConfig("fa-key", "ref"))
	f := svc.Format()
	if f.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", f.SampleRate)
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

func TestFishAudioServiceFormatOpus(t *testing.T) {
	cfg := NewFishAudioConfig("fa-key", "ref")
	cfg.Format = "opus"
	svc := NewFishAudioService(cfg)
	f := svc.Format()
	if f.SampleRate != 48000 {
		t.Errorf("SampleRate for opus = %d, want 48000", f.SampleRate)
	}
}

func TestFishAudioServiceCacheKey(t *testing.T) {
	svc := NewFishAudioService(NewFishAudioConfig("fa-key", "ref"))
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
	if !strings.HasPrefix(k1, "fishaudio.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "fishaudio.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".mp3") {
		t.Errorf("CacheKey should end with %q, got %q", ".mp3", k1)
	}
}

func TestFishAudioServiceCacheKeyReflectsConfig(t *testing.T) {
	svc1 := NewFishAudioService(NewFishAudioConfig("fa-key", "ref1"))
	svc2 := NewFishAudioService(NewFishAudioConfig("fa-key", "ref2"))
	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when reference ID differs")
	}
}

func TestFishAudioServiceClose(t *testing.T) {
	svc := NewFishAudioService(NewFishAudioConfig("fa-key", "ref"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestFishAudioServiceSynthesizeEmptyAPIKey(t *testing.T) {
	svc := NewFishAudioService(NewFishAudioConfig("", "ref"))
	err := svc.Synthesize(context.Background(), base.HandlerFunc{}, "hello")
	if err == nil {
		t.Fatal("Synthesize with empty API key should return error")
	}
	if !strings.Contains(err.Error(), "FISHAUDIO_API_KEY") {
		t.Errorf("expected error mentioning FISHAUDIO_API_KEY, got %v", err)
	}
}
