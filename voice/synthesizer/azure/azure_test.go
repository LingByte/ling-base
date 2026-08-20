package synthesizer

import (
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/synthesizer"
)

func TestNewAzureConfigDefaults(t *testing.T) {
	c := NewAzureConfig("key123", "eastus")
	if c.SubscriptionKey != "key123" {
		t.Errorf("SubscriptionKey = %q, want %q", c.SubscriptionKey, "key123")
	}
	if c.Region != "eastus" {
		t.Errorf("Region = %q, want %q", c.Region, "eastus")
	}
	if c.Voice != "zh-CN-XiaoxiaoNeural" {
		t.Errorf("Voice = %q, want %q", c.Voice, "zh-CN-XiaoxiaoNeural")
	}
	if c.SampleRate != 22050 {
		t.Errorf("SampleRate = %d, want 22050", c.SampleRate)
	}
	if c.Channels != 1 {
		t.Errorf("Channels = %d, want 1", c.Channels)
	}
	if c.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", c.BitDepth)
	}
	if c.Codec != "audio-24khz-48kbitrate-mono-mp3" {
		t.Errorf("Codec = %q, want %q", c.Codec, "audio-24khz-48kbitrate-mono-mp3")
	}
	if c.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", c.FrameDuration, "20ms")
	}
	if c.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", c.Timeout)
	}
}

func TestNewAzureConfigGetProvider(t *testing.T) {
	c := NewAzureConfig("key", "eastus")
	if got := c.GetProvider(); got != base.ProviderAzure {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderAzure)
	}
}

func TestAzureServiceProvider(t *testing.T) {
	svc := NewAzureService(NewAzureConfig("key", "eastus"))
	if got := svc.Provider(); got != base.ProviderAzure {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderAzure)
	}
}

func TestAzureServiceFormat(t *testing.T) {
	svc := NewAzureService(NewAzureConfig("key", "eastus"))
	f := svc.Format()
	if f.SampleRate != 22050 {
		t.Errorf("SampleRate = %d, want 22050", f.SampleRate)
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

func TestAzureServiceCacheKey(t *testing.T) {
	svc := NewAzureService(NewAzureConfig("key", "eastus"))
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
	if !strings.HasPrefix(k1, "azure.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "azure.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".mp3") {
		t.Errorf("CacheKey should end with %q, got %q", ".mp3", k1)
	}
}

func TestAzureServiceCacheKeyReflectsConfig(t *testing.T) {
	svc1 := NewAzureService(NewAzureConfig("key", "eastus"))
	svc2 := NewAzureService(NewAzureConfig("key", "westus"))
	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when region differs")
	}
}

func TestAzureServiceClose(t *testing.T) {
	svc := NewAzureService(NewAzureConfig("key", "eastus"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestGetAzureVoices(t *testing.T) {
	voices := GetAzureVoices()
	if len(voices) == 0 {
		t.Fatal("GetAzureVoices() returned empty map")
	}
	if _, ok := voices["zh-CN-XiaoxiaoNeural"]; !ok {
		t.Error("expected zh-CN-XiaoxiaoNeural in voices map")
	}
}
