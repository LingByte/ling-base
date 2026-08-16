package synthesizer

import (
	"net/url"
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/synthesizer"
)

func TestNewBaiduTTSOptionDefaults(t *testing.T) {
	c := NewBaiduTTSOption("tok123")
	if c.Tok != "tok123" {
		t.Errorf("Tok = %q, want %q", c.Tok, "tok123")
	}
	if c.Ctp != "1" {
		t.Errorf("Ctp = %q, want %q", c.Ctp, "1")
	}
	if c.Lan != "zh" {
		t.Errorf("Lan = %q, want %q", c.Lan, "zh")
	}
	if c.Spd != "5" {
		t.Errorf("Spd = %q, want %q", c.Spd, "5")
	}
	if c.Pit != "5" {
		t.Errorf("Pit = %q, want %q", c.Pit, "5")
	}
	if c.Vol != "5" {
		t.Errorf("Vol = %q, want %q", c.Vol, "5")
	}
	if c.Aue != "3" {
		t.Errorf("Aue = %q, want %q", c.Aue, "3")
	}
	if c.Channels != 1 {
		t.Errorf("Channels = %d, want 1", c.Channels)
	}
	if c.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", c.SampleRate)
	}
	if c.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", c.BitDepth)
	}
	if c.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", c.FrameDuration, "20ms")
	}
}

func TestNewBaiduTTSOptionGetProvider(t *testing.T) {
	c := NewBaiduTTSOption("tok")
	if got := c.GetProvider(); got != base.ProviderBaidu {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderBaidu)
	}
}

func TestBaiduTTSServiceProvider(t *testing.T) {
	svc := NewBaiduService(NewBaiduTTSOption("tok"))
	if got := svc.Provider(); got != base.ProviderBaidu {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderBaidu)
	}
}

func TestBaiduTTSServiceFormat(t *testing.T) {
	svc := NewBaiduService(NewBaiduTTSOption("tok"))
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

func TestBaiduTTSServiceCacheKey(t *testing.T) {
	svc := NewBaiduService(NewBaiduTTSOption("tok"))
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
	if !strings.HasPrefix(k1, "baidu.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "baidu.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".pcm") {
		t.Errorf("CacheKey should end with %q, got %q", ".pcm", k1)
	}
}

func TestBaiduTTSServiceCacheKeyReflectsConfig(t *testing.T) {
	c1 := NewBaiduTTSOption("tok")
	c1.Lan = "zh"
	svc1 := NewBaiduService(c1)

	c2 := NewBaiduTTSOption("tok")
	c2.Lan = "en"
	svc2 := NewBaiduService(c2)

	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when language differs")
	}
}

func TestBaiduTTSServiceClose(t *testing.T) {
	svc := NewBaiduService(NewBaiduTTSOption("tok"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestDoubleURLEncode(t *testing.T) {
	svc := NewBaiduService(NewBaiduTTSOption("tok"))
	input := "你好 world"
	encoded := svc.DoubleURLEncode(input)

	// Double encoding means the result, when decoded once, equals a single
	// QueryEscape of the input.
	once, err := url.QueryUnescape(encoded)
	if err != nil {
		t.Fatalf("first unescape failed: %v", err)
	}
	if once != url.QueryEscape(input) {
		t.Errorf("DoubleURLEncode mismatch: once-decoded = %q, want %q", once, url.QueryEscape(input))
	}

	// Decoding twice should yield the original input.
	twice, err := url.QueryUnescape(once)
	if err != nil {
		t.Fatalf("second unescape failed: %v", err)
	}
	if twice != input {
		t.Errorf("DoubleURLEncode fully decoded = %q, want %q", twice, input)
	}
}

func TestDoubleURLEncodeEmpty(t *testing.T) {
	svc := NewBaiduService(NewBaiduTTSOption("tok"))
	if got := svc.DoubleURLEncode(""); got != "" {
		t.Errorf("DoubleURLEncode(\"\") = %q, want empty", got)
	}
}
