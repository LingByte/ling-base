package synthesizer

import (
	"strings"
	"testing"

	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	base "github.com/LingByte/ling-base/synthesizer"
)

func TestNewGoogleTTSOption(t *testing.T) {
	opt := NewGoogleTTSOption("en-US")
	if opt.LanguageCode != "en-US" {
		t.Errorf("LanguageCode = %q, want %q", opt.LanguageCode, "en-US")
	}
	if opt.SsmlGender != texttospeechpb.SsmlVoiceGender_NEUTRAL {
		t.Errorf("SsmlGender = %v, want NEUTRAL", opt.SsmlGender)
	}
	if opt.AudioEncoding != texttospeechpb.AudioEncoding_LINEAR16 {
		t.Errorf("AudioEncoding = %v, want LINEAR16", opt.AudioEncoding)
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

func TestGoogleTTSOptionGetProvider(t *testing.T) {
	opt := NewGoogleTTSOption("en-US")
	if opt.GetProvider() != base.ProviderGoogle {
		t.Errorf("GetProvider() = %q, want %q", opt.GetProvider(), base.ProviderGoogle)
	}
}

func TestGoogleTTSOptionString(t *testing.T) {
	opt := NewGoogleTTSOption("en-US")
	s := opt.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
	if !strings.Contains(s, "en-US") {
		t.Errorf("String() should contain language code, got %q", s)
	}
}

func TestNewGoogleService(t *testing.T) {
	opt := NewGoogleTTSOption("en-US")
	svc := NewGoogleService(opt)
	if svc == nil {
		t.Fatal("NewGoogleService returned nil")
	}
}

func TestGoogleServiceProvider(t *testing.T) {
	svc := NewGoogleService(NewGoogleTTSOption("en-US"))
	if svc.Provider() != base.ProviderGoogle {
		t.Errorf("Provider() = %q, want %q", svc.Provider(), base.ProviderGoogle)
	}
}

func TestGoogleServiceFormat(t *testing.T) {
	svc := NewGoogleService(NewGoogleTTSOption("en-US"))
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

func TestGoogleServiceCacheKey(t *testing.T) {
	svc := NewGoogleService(NewGoogleTTSOption("en-US"))
	key1 := svc.CacheKey("hello")
	key2 := svc.CacheKey("hello")
	key3 := svc.CacheKey("world")

	if key1 != key2 {
		t.Error("CacheKey should be deterministic for same input")
	}
	if key1 == key3 {
		t.Error("CacheKey should differ for different input")
	}
	if !strings.Contains(key1, "google.tts-en-US-") {
		t.Errorf("CacheKey should have google.tts prefix, got %q", key1)
	}
	if !strings.Contains(key1, ".pcm") {
		t.Errorf("CacheKey should end with .pcm, got %q", key1)
	}
}

func TestGoogleServiceClose(t *testing.T) {
	svc := NewGoogleService(NewGoogleTTSOption("en-US"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestGoogleServiceSynthesizeEmpty(t *testing.T) {
	svc := NewGoogleService(NewGoogleTTSOption("en-US"))
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
