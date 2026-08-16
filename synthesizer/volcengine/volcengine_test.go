package synthesizer

import (
	"context"
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/synthesizer"
)

func TestNewVolcengineTTSOptionDefaults(t *testing.T) {
	o := NewVolcengineTTSOption("app123", "token456", "cluster789")
	if o.AppID != "app123" {
		t.Errorf("AppID = %q, want %q", o.AppID, "app123")
	}
	if o.AccessToken != "token456" {
		t.Errorf("AccessToken = %q, want %q", o.AccessToken, "token456")
	}
	if o.Cluster != "cluster789" {
		t.Errorf("Cluster = %q, want %q", o.Cluster, "cluster789")
	}
	if o.VoiceType != "BV700_streaming" {
		t.Errorf("VoiceType = %q, want %q", o.VoiceType, "BV700_streaming")
	}
	if o.Rate != 16000 {
		t.Errorf("Rate = %d, want 16000", o.Rate)
	}
	if o.Encoding != "pcm" {
		t.Errorf("Encoding = %q, want %q", o.Encoding, "pcm")
	}
	if o.SpeedRatio != 1.0 {
		t.Errorf("SpeedRatio = %v, want 1.0", o.SpeedRatio)
	}
	if o.VolumeRatio != 1.0 {
		t.Errorf("VolumeRatio = %v, want 1.0", o.VolumeRatio)
	}
	if o.PitchRatio != 1.0 {
		t.Errorf("PitchRatio = %v, want 1.0", o.PitchRatio)
	}
	if o.Channels != 1 {
		t.Errorf("Channels = %d, want 1", o.Channels)
	}
	if o.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", o.BitDepth)
	}
	if o.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", o.FrameDuration, "20ms")
	}
	if o.TextType != "plain" {
		t.Errorf("TextType = %q, want %q", o.TextType, "plain")
	}
	if o.Ssml != false {
		t.Errorf("Ssml = %v, want false", o.Ssml)
	}
	if o.Streaming != true {
		t.Errorf("Streaming = %v, want true", o.Streaming)
	}
}

func TestNewVolcengineTTSOptionGetProvider(t *testing.T) {
	o := NewVolcengineTTSOption("app", "token", "cluster")
	if got := o.GetProvider(); got != base.ProviderVolcengine {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderVolcengine)
	}
}

func TestVolcengineServiceProvider(t *testing.T) {
	svc := NewVolcengineService(NewVolcengineTTSOption("app", "token", "cluster"))
	if got := svc.Provider(); got != base.ProviderVolcengine {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderVolcengine)
	}
}

func TestVolcengineServiceFormat(t *testing.T) {
	svc := NewVolcengineService(NewVolcengineTTSOption("app", "token", "cluster"))
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
	if f.Codec != "pcm" {
		t.Errorf("Codec = %q, want %q", f.Codec, "pcm")
	}
	if f.FrameDuration != 20*time.Millisecond {
		t.Errorf("FrameDuration = %v, want 20ms", f.FrameDuration)
	}
}

func TestVolcengineServiceCacheKey(t *testing.T) {
	svc := NewVolcengineService(NewVolcengineTTSOption("app", "token", "cluster"))
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
	if !strings.HasPrefix(k1, "volcengine.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "volcengine.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".pcm") {
		t.Errorf("CacheKey should end with %q, got %q", ".pcm", k1)
	}
}

func TestVolcengineServiceCacheKeyReflectsConfig(t *testing.T) {
	o1 := NewVolcengineTTSOption("app", "token", "cluster")
	o1.VoiceType = "BV700_streaming"
	svc1 := NewVolcengineService(o1)

	o2 := NewVolcengineTTSOption("app", "token", "cluster")
	o2.VoiceType = "BV001_streaming"
	svc2 := NewVolcengineService(o2)

	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when voice type differs")
	}
}

func TestVolcengineServiceCapabilitiesStreaming(t *testing.T) {
	o := NewVolcengineTTSOption("app", "token", "cluster")
	o.Streaming = true
	svc := NewVolcengineService(o)
	caps := svc.Capabilities()
	if !caps.StreamingTTFB {
		t.Error("Capabilities().StreamingTTFB should be true when streaming=true")
	}
}

func TestVolcengineServiceCapabilitiesNonStreaming(t *testing.T) {
	o := NewVolcengineTTSOption("app", "token", "cluster")
	o.Streaming = false
	svc := NewVolcengineService(o)
	caps := svc.Capabilities()
	if caps.StreamingTTFB {
		t.Error("Capabilities().StreamingTTFB should be false when streaming=false")
	}
}

func TestVolcengineServiceClose(t *testing.T) {
	svc := NewVolcengineService(NewVolcengineTTSOption("app", "token", "cluster"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestVolcengineServicePrewarmNilPool(t *testing.T) {
	svc := NewVolcengineService(NewVolcengineTTSOption("app", "token", "cluster"))
	// Force a nil pool to verify Prewarm does not panic.
	svc.pool = nil
	// Should not panic.
	svc.Prewarm(context.Background())
}

func TestVolcengineServiceSynthesizeEmptyText(t *testing.T) {
	svc := NewVolcengineService(NewVolcengineTTSOption("app", "token", "cluster"))
	svc.pool = nil
	var got []byte
	h := base.HandlerFunc{OnMessageFn: func(data []byte) { got = data }}
	if err := svc.Synthesize(context.Background(), h, ""); err != nil {
		t.Errorf("Synthesize with empty text should not error: %v", err)
	}
	if got == nil {
		t.Error("expected OnMessage to be called with empty slice for empty text")
	}
}
