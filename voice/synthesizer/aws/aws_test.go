package synthesizer

import (
	"strings"
	"testing"

	base "github.com/LingByte/ling-base/voice/synthesizer"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
)

func TestNewAmazonTTSOption(t *testing.T) {
	opt := NewAmazonTTSOption("us-east-1", types.OutputFormatPcm, types.VoiceIdJoanna)
	if opt.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", opt.Region, "us-east-1")
	}
	if opt.OutputFormat != types.OutputFormatPcm {
		t.Errorf("OutputFormat = %v, want pcm", opt.OutputFormat)
	}
	if opt.VoiceId != types.VoiceIdJoanna {
		t.Errorf("VoiceId = %v, want Joanna", opt.VoiceId)
	}
	if opt.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", opt.SampleRate)
	}
	if opt.Channels != 1 {
		t.Errorf("Channels = %d, want 1", opt.Channels)
	}
	if opt.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", opt.BitDepth)
	}
	if opt.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", opt.FrameDuration, "20ms")
	}
}

func TestAmazonTTSConfigGetProvider(t *testing.T) {
	opt := NewAmazonTTSOption("us-east-1", types.OutputFormatPcm, types.VoiceIdJoanna)
	if opt.GetProvider() != base.ProviderAWS {
		t.Errorf("GetProvider() = %q, want %q", opt.GetProvider(), base.ProviderAWS)
	}
}

func TestAmazonTTSConfigString(t *testing.T) {
	opt := NewAmazonTTSOption("us-east-1", types.OutputFormatPcm, types.VoiceIdJoanna)
	s := opt.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
	if !strings.Contains(s, "us-east-1") {
		t.Errorf("String() should contain region, got %q", s)
	}
}

func TestNewAmazonService(t *testing.T) {
	opt := NewAmazonTTSOption("us-east-1", types.OutputFormatPcm, types.VoiceIdJoanna)
	svc := NewAmazonService(opt)
	if svc == nil {
		t.Fatal("NewAmazonService returned nil")
	}
}

func TestAmazonServiceProvider(t *testing.T) {
	svc := NewAmazonService(NewAmazonTTSOption("us-east-1", types.OutputFormatPcm, types.VoiceIdJoanna))
	if svc.Provider() != base.ProviderAWS {
		t.Errorf("Provider() = %q, want %q", svc.Provider(), base.ProviderAWS)
	}
}

func TestAmazonServiceFormat(t *testing.T) {
	svc := NewAmazonService(NewAmazonTTSOption("us-east-1", types.OutputFormatPcm, types.VoiceIdJoanna))
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
}

func TestAmazonServiceCacheKey(t *testing.T) {
	svc := NewAmazonService(NewAmazonTTSOption("us-east-1", types.OutputFormatPcm, types.VoiceIdJoanna))
	key1 := svc.CacheKey("hello")
	key2 := svc.CacheKey("hello")
	key3 := svc.CacheKey("world")

	if key1 != key2 {
		t.Error("CacheKey should be deterministic for same input")
	}
	if key1 == key3 {
		t.Error("CacheKey should differ for different input")
	}
	if !strings.Contains(key1, "amazon.tts-") {
		t.Errorf("CacheKey should have amazon.tts prefix, got %q", key1)
	}
}

func TestAmazonServiceClose(t *testing.T) {
	svc := NewAmazonService(NewAmazonTTSOption("us-east-1", types.OutputFormatPcm, types.VoiceIdJoanna))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestAmazonServiceSynthesizeEmpty(t *testing.T) {
	svc := NewAmazonService(NewAmazonTTSOption("us-east-1", types.OutputFormatPcm, types.VoiceIdJoanna))
	var called bool
	h := base.HandlerFunc{
		OnMessageFn: func(data []byte) {
			called = true
		},
	}
	// Empty text should call OnMessage(nil) and return nil without network
	if err := svc.Synthesize(nil, h, ""); err != nil {
		t.Errorf("Synthesize(empty) should not error: %v", err)
	}
	if !called {
		t.Error("OnMessage should be called with nil for empty text")
	}
}
