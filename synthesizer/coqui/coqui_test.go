package synthesizer

import (
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/synthesizer"
)

func TestNewCoquiTTSOptionDefaults(t *testing.T) {
	c := NewCoquiTTSOption("http://localhost:5002")
	if c.Url != "http://localhost:5002" {
		t.Errorf("Url = %q, want %q", c.Url, "http://localhost:5002")
	}
	if c.Language != "en-US" {
		t.Errorf("Language = %q, want %q", c.Language, "en-US")
	}
	if c.Speaker != "p226" {
		t.Errorf("Speaker = %q, want %q", c.Speaker, "p226")
	}
	if c.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", c.SampleRate)
	}
	if c.Channels != 1 {
		t.Errorf("Channels = %d, want 1", c.Channels)
	}
	if c.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", c.BitDepth)
	}
}

func TestNewCoquiTTSOptionGetProvider(t *testing.T) {
	c := NewCoquiTTSOption("http://localhost:5002")
	if got := c.GetProvider(); got != base.ProviderCoqui {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderCoqui)
	}
}

func TestCoquiServiceProvider(t *testing.T) {
	svc := NewCoquiService(NewCoquiTTSOption("http://localhost:5002"))
	if got := svc.Provider(); got != base.ProviderCoqui {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderCoqui)
	}
}

func TestCoquiServiceFormat(t *testing.T) {
	svc := NewCoquiService(NewCoquiTTSOption("http://localhost:5002"))
	f := svc.Format()
	if f.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", f.SampleRate)
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

func TestCoquiServiceCacheKey(t *testing.T) {
	svc := NewCoquiService(NewCoquiTTSOption("http://localhost:5002"))
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
	if !strings.HasPrefix(k1, "coqui.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "coqui.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".pcm") {
		t.Errorf("CacheKey should end with %q, got %q", ".pcm", k1)
	}
}

// TestCoquiServiceCacheKeyPrefixNotQcloud guards against a regression where
// the Coqui cache key accidentally used the qcloud prefix.
func TestCoquiServiceCacheKeyPrefixNotQcloud(t *testing.T) {
	svc := NewCoquiService(NewCoquiTTSOption("http://localhost:5002"))
	k := svc.CacheKey("hello")
	if strings.HasPrefix(k, "qcloud.tts") {
		t.Errorf("CacheKey must not use qcloud.tts prefix, got %q", k)
	}
	if !strings.HasPrefix(k, "coqui.tts") {
		t.Errorf("CacheKey must use coqui.tts prefix, got %q", k)
	}
}

func TestCoquiServiceCacheKeyReflectsConfig(t *testing.T) {
	c1 := NewCoquiTTSOption("http://localhost:5002")
	c1.Speaker = "spk1"
	svc1 := NewCoquiService(c1)

	c2 := NewCoquiTTSOption("http://localhost:5002")
	c2.Speaker = "spk2"
	svc2 := NewCoquiService(c2)

	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when speaker differs")
	}
}

func TestCoquiServiceClose(t *testing.T) {
	svc := NewCoquiService(NewCoquiTTSOption("http://localhost:5002"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
