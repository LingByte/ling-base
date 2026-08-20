package synthesizer

import (
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/synthesizer"
)

func TestNewMinimaxOptionDefaults(t *testing.T) {
	c := NewMinimaxOption("mm-key")
	if c.APIKey != "mm-key" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "mm-key")
	}
	if c.Model != MinimaxSpeech25TurboPreview {
		t.Errorf("Model = %q, want %q", c.Model, MinimaxSpeech25TurboPreview)
	}
	if c.VoiceID != "male-qn-qingse" {
		t.Errorf("VoiceID = %q, want %q", c.VoiceID, "male-qn-qingse")
	}
	if c.SpeedRatio != 1.0 {
		t.Errorf("SpeedRatio = %v, want 1.0", c.SpeedRatio)
	}
	if c.Volume != 1.0 {
		t.Errorf("Volume = %v, want 1.0", c.Volume)
	}
	if c.Pitch != 0.0 {
		t.Errorf("Pitch = %v, want 0.0", c.Pitch)
	}
	if c.Emotion != "neutral" {
		t.Errorf("Emotion = %q, want %q", c.Emotion, "neutral")
	}
	if c.SampleRate != 8000 {
		t.Errorf("SampleRate = %d, want 8000", c.SampleRate)
	}
	if c.Bitrate != 16 {
		t.Errorf("Bitrate = %d, want 16", c.Bitrate)
	}
	if c.Format != "pcm" {
		t.Errorf("Format = %q, want %q", c.Format, "pcm")
	}
	if c.Channels != 1 {
		t.Errorf("Channels = %d, want 1", c.Channels)
	}
	if c.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", c.FrameDuration, "20ms")
	}
}

func TestNewMinimaxOptionGetProvider(t *testing.T) {
	c := NewMinimaxOption("mm-key")
	if got := c.GetProvider(); got != base.ProviderMinimax {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderMinimax)
	}
}

func TestMinimaxServiceProvider(t *testing.T) {
	svc := NewMinimaxService(NewMinimaxOption("mm-key"))
	if got := svc.Provider(); got != base.ProviderMinimax {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderMinimax)
	}
}

func TestMinimaxServiceFormat(t *testing.T) {
	svc := NewMinimaxService(NewMinimaxOption("mm-key"))
	f := svc.Format()
	if f.SampleRate != 8000 {
		t.Errorf("SampleRate = %d, want 8000", f.SampleRate)
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

func TestMinimaxServiceCacheKey(t *testing.T) {
	svc := NewMinimaxService(NewMinimaxOption("mm-key"))
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
	if !strings.HasPrefix(k1, "minimax.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "minimax.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".pcm") {
		t.Errorf("CacheKey should end with %q, got %q", ".pcm", k1)
	}
}

func TestMinimaxServiceCacheKeyReflectsConfig(t *testing.T) {
	c1 := NewMinimaxOption("mm-key")
	c1.VoiceID = "voice1"
	svc1 := NewMinimaxService(c1)

	c2 := NewMinimaxOption("mm-key")
	c2.VoiceID = "voice2"
	svc2 := NewMinimaxService(c2)

	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when voice ID differs")
	}
}

func TestMinimaxServiceClose(t *testing.T) {
	svc := NewMinimaxService(NewMinimaxOption("mm-key"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
