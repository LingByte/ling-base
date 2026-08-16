package synthesizer

import (
	"context"
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/synthesizer"
)

func TestNewElevenLabsConfigDefaults(t *testing.T) {
	c := NewElevenLabsConfig("el-key", "")
	if c.APIKey != "el-key" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "el-key")
	}
	if c.VoiceID != "21m00Tcm4TlvDq8ikWAM" {
		t.Errorf("VoiceID = %q, want default Rachel voice", c.VoiceID)
	}
	if c.ModelID != "eleven_monolingual_v1" {
		t.Errorf("ModelID = %q, want %q", c.ModelID, "eleven_monolingual_v1")
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
	if c.Codec != "mp3" {
		t.Errorf("Codec = %q, want %q", c.Codec, "mp3")
	}
	if c.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", c.FrameDuration, "20ms")
	}
	if c.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", c.Timeout)
	}
	if c.Stability != 0.5 {
		t.Errorf("Stability = %v, want 0.5", c.Stability)
	}
	if c.SimilarityBoost != 0.75 {
		t.Errorf("SimilarityBoost = %v, want 0.75", c.SimilarityBoost)
	}
	if c.Style != 0.0 {
		t.Errorf("Style = %v, want 0.0", c.Style)
	}
	if !c.UseSpeakerBoost {
		t.Errorf("UseSpeakerBoost = %v, want true", c.UseSpeakerBoost)
	}
}

func TestNewElevenLabsConfigExplicitVoiceID(t *testing.T) {
	c := NewElevenLabsConfig("el-key", "custom-voice")
	if c.VoiceID != "custom-voice" {
		t.Errorf("VoiceID = %q, want %q", c.VoiceID, "custom-voice")
	}
}

func TestNewElevenLabsConfigGetProvider(t *testing.T) {
	c := NewElevenLabsConfig("el-key", "")
	if got := c.GetProvider(); got != base.ProviderElevenLabs {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderElevenLabs)
	}
}

func TestElevenLabsServiceProvider(t *testing.T) {
	svc := NewElevenLabsService(NewElevenLabsConfig("el-key", ""))
	if got := svc.Provider(); got != base.ProviderElevenLabs {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderElevenLabs)
	}
}

func TestElevenLabsServiceFormat(t *testing.T) {
	svc := NewElevenLabsService(NewElevenLabsConfig("el-key", ""))
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
	if f.Codec != "mp3" {
		t.Errorf("Codec = %q, want %q", f.Codec, "mp3")
	}
	if f.FrameDuration != 20*time.Millisecond {
		t.Errorf("FrameDuration = %v, want 20ms", f.FrameDuration)
	}
}

func TestElevenLabsServiceCacheKey(t *testing.T) {
	svc := NewElevenLabsService(NewElevenLabsConfig("el-key", ""))
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
	if !strings.HasPrefix(k1, "elevenlabs.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "elevenlabs.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".mp3") {
		t.Errorf("CacheKey should end with %q, got %q", ".mp3", k1)
	}
}

func TestElevenLabsServiceCacheKeyReflectsConfig(t *testing.T) {
	svc1 := NewElevenLabsService(NewElevenLabsConfig("el-key", "voice1"))
	svc2 := NewElevenLabsService(NewElevenLabsConfig("el-key", "voice2"))
	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when voice ID differs")
	}
}

func TestElevenLabsServiceClose(t *testing.T) {
	svc := NewElevenLabsService(NewElevenLabsConfig("el-key", ""))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestElevenLabsServiceSynthesizeEmptyAPIKey(t *testing.T) {
	svc := NewElevenLabsService(NewElevenLabsConfig("", ""))
	err := svc.Synthesize(context.Background(), base.HandlerFunc{}, "hello")
	if err == nil {
		t.Fatal("Synthesize with empty API key should return error")
	}
	if !strings.Contains(err.Error(), "ELEVENLABS_API_KEY") {
		t.Errorf("expected error mentioning ELEVENLABS_API_KEY, got %v", err)
	}
}
