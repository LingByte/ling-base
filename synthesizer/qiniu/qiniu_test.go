package synthesizer

import (
	"context"
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/synthesizer"
)

func TestNewQiniuTTSConfigDefaults(t *testing.T) {
	c := NewQiniuTTSConfig("qk-key", "https://api.qiniu.com")
	if c.APIKey != "qk-key" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "qk-key")
	}
	if c.BaseURL != "https://api.qiniu.com" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, "https://api.qiniu.com")
	}
	if c.VoiceType != "qiniu_zh_female_tmjxxy" {
		t.Errorf("VoiceType = %q, want %q", c.VoiceType, "qiniu_zh_female_tmjxxy")
	}
	if c.SampleRate != 30000 {
		t.Errorf("SampleRate = %d, want 30000", c.SampleRate)
	}
	if c.Channels != 1 {
		t.Errorf("Channels = %d, want 1", c.Channels)
	}
	if c.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", c.BitDepth)
	}
	if c.Codec != "pcm" {
		t.Errorf("Codec = %q, want %q", c.Codec, "pcm")
	}
	if c.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", c.FrameDuration, "20ms")
	}
	if c.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", c.Timeout)
	}
	if c.Retries != 0 {
		t.Errorf("Retries = %d, want 0", c.Retries)
	}
}

func TestNewQiniuTTSConfigGetProvider(t *testing.T) {
	c := NewQiniuTTSConfig("qk-key", "https://api.qiniu.com")
	if got := c.GetProvider(); got != base.ProviderQiniu {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderQiniu)
	}
}

func TestQiniuServiceProvider(t *testing.T) {
	svc := NewQiniuService(NewQiniuTTSConfig("qk-key", "https://api.qiniu.com"))
	if got := svc.Provider(); got != base.ProviderQiniu {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderQiniu)
	}
}

func TestQiniuServiceFormat(t *testing.T) {
	svc := NewQiniuService(NewQiniuTTSConfig("qk-key", "https://api.qiniu.com"))
	f := svc.Format()
	if f.SampleRate != 30000 {
		t.Errorf("SampleRate = %d, want 30000", f.SampleRate)
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

func TestQiniuServiceCacheKey(t *testing.T) {
	svc := NewQiniuService(NewQiniuTTSConfig("qk-key", "https://api.qiniu.com"))
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
	if !strings.HasPrefix(k1, "qiniu.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "qiniu.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".pcm") {
		t.Errorf("CacheKey should end with %q, got %q", ".pcm", k1)
	}
}

func TestQiniuServiceCacheKeyReflectsConfig(t *testing.T) {
	c1 := NewQiniuTTSConfig("qk-key", "https://api.qiniu.com")
	c1.VoiceType = "voice1"
	svc1 := NewQiniuService(c1)

	c2 := NewQiniuTTSConfig("qk-key", "https://api.qiniu.com")
	c2.VoiceType = "voice2"
	svc2 := NewQiniuService(c2)

	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when voice type differs")
	}
}

func TestQiniuServiceClose(t *testing.T) {
	svc := NewQiniuService(NewQiniuTTSConfig("qk-key", "https://api.qiniu.com"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestQiniuServiceSynthesizeEmptyAPIKey(t *testing.T) {
	svc := NewQiniuService(NewQiniuTTSConfig("", "https://api.qiniu.com"))
	err := svc.Synthesize(context.Background(), base.HandlerFunc{}, "hello")
	if err == nil {
		t.Fatal("Synthesize with empty API key should return error")
	}
	if !strings.Contains(err.Error(), "QINIU_TTS_API_KEY") {
		t.Errorf("expected error mentioning QINIU_TTS_API_KEY, got %v", err)
	}
}
