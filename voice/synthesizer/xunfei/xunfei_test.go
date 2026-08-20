package synthesizer

import (
	"context"
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/synthesizer"
)

func TestNewXunfeiTTSConfigDefaults(t *testing.T) {
	c := NewXunfeiTTSConfig("app123", "key456", "secret789")
	if c.AppID != "app123" {
		t.Errorf("AppID = %q, want %q", c.AppID, "app123")
	}
	if c.APIKey != "key456" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "key456")
	}
	if c.APISecret != "secret789" {
		t.Errorf("APISecret = %q, want %q", c.APISecret, "secret789")
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
	if c.Codec != "raw" {
		t.Errorf("Codec = %q, want %q", c.Codec, "raw")
	}
	if c.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", c.FrameDuration, "20ms")
	}
	if c.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", c.Timeout)
	}
}

func TestNewXunfeiTTSConfigGetProvider(t *testing.T) {
	c := NewXunfeiTTSConfig("app", "key", "secret")
	if got := c.GetProvider(); got != base.ProviderXunfei {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderXunfei)
	}
}

func TestXunfeiServiceProvider(t *testing.T) {
	svc := NewXunfeiService(NewXunfeiTTSConfig("app", "key", "secret"))
	if got := svc.Provider(); got != base.ProviderXunfei {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderXunfei)
	}
}

func TestXunfeiServiceFormat(t *testing.T) {
	svc := NewXunfeiService(NewXunfeiTTSConfig("app", "key", "secret"))
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
	if f.Codec != "raw" {
		t.Errorf("Codec = %q, want %q", f.Codec, "raw")
	}
	if f.FrameDuration != 20*time.Millisecond {
		t.Errorf("FrameDuration = %v, want 20ms", f.FrameDuration)
	}
}

func TestXunfeiServiceCacheKey(t *testing.T) {
	svc := NewXunfeiService(NewXunfeiTTSConfig("app", "key", "secret"))
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
	if !strings.HasPrefix(k1, "xunfei.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "xunfei.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".raw") {
		t.Errorf("CacheKey should end with %q, got %q", ".raw", k1)
	}
}

func TestXunfeiServiceCacheKeyDefaultResID(t *testing.T) {
	svc := NewXunfeiService(NewXunfeiTTSConfig("app", "key", "secret"))
	// ResID is empty by default; CacheKey should use the fallback token.
	k := svc.CacheKey("hello")
	if !strings.Contains(k, "xunfei_default") {
		t.Errorf("CacheKey should contain default resID fallback, got %q", k)
	}
}

func TestXunfeiServiceCacheKeyReflectsConfig(t *testing.T) {
	c1 := NewXunfeiTTSConfig("app", "key", "secret")
	c1.ResID = "res1"
	svc1 := NewXunfeiService(c1)

	c2 := NewXunfeiTTSConfig("app", "key", "secret")
	c2.ResID = "res2"
	svc2 := NewXunfeiService(c2)

	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when res ID differs")
	}
}

func TestXunfeiServiceClose(t *testing.T) {
	svc := NewXunfeiService(NewXunfeiTTSConfig("app", "key", "secret"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestXunfeiServiceSynthesizeMissingAppID(t *testing.T) {
	svc := NewXunfeiService(NewXunfeiTTSConfig("", "key", "secret"))
	err := svc.Synthesize(context.Background(), base.HandlerFunc{}, "hello")
	if err == nil {
		t.Fatal("Synthesize with empty AppID should return error")
	}
	if !strings.Contains(err.Error(), "XUNFEI_APP_ID") {
		t.Errorf("expected error mentioning XUNFEI_APP_ID, got %v", err)
	}
}

func TestXunfeiServiceSynthesizeMissingAPIKey(t *testing.T) {
	svc := NewXunfeiService(NewXunfeiTTSConfig("app", "", "secret"))
	err := svc.Synthesize(context.Background(), base.HandlerFunc{}, "hello")
	if err == nil {
		t.Fatal("Synthesize with empty APIKey should return error")
	}
	if !strings.Contains(err.Error(), "XUNFEI_API_KEY") {
		t.Errorf("expected error mentioning XUNFEI_API_KEY, got %v", err)
	}
}

func TestXunfeiServiceSynthesizeMissingAPISecret(t *testing.T) {
	svc := NewXunfeiService(NewXunfeiTTSConfig("app", "key", ""))
	err := svc.Synthesize(context.Background(), base.HandlerFunc{}, "hello")
	if err == nil {
		t.Fatal("Synthesize with empty APISecret should return error")
	}
	if !strings.Contains(err.Error(), "XUNFEI_API_SECRET") {
		t.Errorf("expected error mentioning XUNFEI_API_SECRET, got %v", err)
	}
}
