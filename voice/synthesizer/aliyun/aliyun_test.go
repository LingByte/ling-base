package synthesizer

import (
	"context"
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/synthesizer"
)

func TestNewAliyunTTSConfigDefaults(t *testing.T) {
	c := NewAliyunTTSConfig("sk-test")
	if c.APIKey != "sk-test" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "sk-test")
	}
	if c.BaseURL != aliyunDefaultEndpoint {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, aliyunDefaultEndpoint)
	}
	if c.Model != aliyunDefaultModel {
		t.Errorf("Model = %q, want %q", c.Model, aliyunDefaultModel)
	}
	if c.Voice != aliyunDefaultVoice {
		t.Errorf("Voice = %q, want %q", c.Voice, aliyunDefaultVoice)
	}
	if c.LanguageType != aliyunDefaultLanguage {
		t.Errorf("LanguageType = %q, want %q", c.LanguageType, aliyunDefaultLanguage)
	}
	if c.Mode != aliyunDefaultMode {
		t.Errorf("Mode = %q, want %q", c.Mode, aliyunDefaultMode)
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
	if c.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", c.FrameDuration, "20ms")
	}
	if c.DialTimeoutMs != 10000 {
		t.Errorf("DialTimeoutMs = %d, want 10000", c.DialTimeoutMs)
	}
}

func TestNewAliyunTTSConfigGetProvider(t *testing.T) {
	c := NewAliyunTTSConfig("sk-test")
	if got := c.GetProvider(); got != base.ProviderAliyun {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderAliyun)
	}
}

func TestAliyunServiceProvider(t *testing.T) {
	svc := NewAliyunService(NewAliyunTTSConfig("sk-test"))
	if got := svc.Provider(); got != base.ProviderAliyun {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderAliyun)
	}
}

func TestAliyunServiceFormat(t *testing.T) {
	svc := NewAliyunService(NewAliyunTTSConfig("sk-test"))
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
	if f.Codec != "pcm" {
		t.Errorf("Codec = %q, want %q", f.Codec, "pcm")
	}
	if f.FrameDuration != 20*time.Millisecond {
		t.Errorf("FrameDuration = %v, want 20ms", f.FrameDuration)
	}
}

func TestAliyunServiceCacheKey(t *testing.T) {
	svc := NewAliyunService(NewAliyunTTSConfig("sk-test"))
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
	if !strings.HasPrefix(k1, "aliyun.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "aliyun.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".pcm") {
		t.Errorf("CacheKey should end with %q, got %q", ".pcm", k1)
	}
}

func TestAliyunServiceCacheKeyReflectsConfig(t *testing.T) {
	cfg1 := NewAliyunTTSConfig("sk-test")
	cfg1.Voice = "Cherry"
	svc1 := NewAliyunService(cfg1)

	cfg2 := NewAliyunTTSConfig("sk-test")
	cfg2.Voice = "Ethan"
	svc2 := NewAliyunService(cfg2)

	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when voice differs")
	}
}

func TestAliyunServiceCapabilities(t *testing.T) {
	svc := NewAliyunService(NewAliyunTTSConfig("sk-test"))
	caps := svc.Capabilities()
	if !caps.StreamingTTFB {
		t.Error("Capabilities().StreamingTTFB should be true for streaming vendor")
	}
	if caps.SuggestedFirstMaxRunes <= 0 {
		t.Errorf("SuggestedFirstMaxRunes = %d, want > 0", caps.SuggestedFirstMaxRunes)
	}
}

func TestAliyunServiceSynthesizeEmptyAPIKey(t *testing.T) {
	cfg := NewAliyunTTSConfig("")
	svc := NewAliyunService(cfg)
	err := svc.Synthesize(context.Background(), base.HandlerFunc{}, "hello")
	if err == nil {
		t.Fatal("Synthesize with empty API key should return error")
	}
	if !strings.Contains(err.Error(), "DASHSCOPE_API_KEY") {
		t.Errorf("expected error mentioning DASHSCOPE_API_KEY, got %v", err)
	}
}
