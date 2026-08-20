package synthesizer

import (
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/synthesizer"
)

// -----------------------------------------------------------------------------
// LocalService
// -----------------------------------------------------------------------------

func TestNewLocalTTSConfigDefaults(t *testing.T) {
	c := NewLocalTTSConfig("say")
	if c.Command != "say" {
		t.Errorf("Command = %q, want %q", c.Command, "say")
	}
	if c.Voice != "" {
		t.Errorf("Voice = %q, want empty", c.Voice)
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
	if c.Codec != "wav" {
		t.Errorf("Codec = %q, want %q", c.Codec, "wav")
	}
	if c.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %q, want %q", c.FrameDuration, "20ms")
	}
	if c.OutputDir != "/tmp" {
		t.Errorf("OutputDir = %q, want %q", c.OutputDir, "/tmp")
	}
}

func TestNewLocalTTSConfigGetProvider(t *testing.T) {
	c := NewLocalTTSConfig("say")
	if got := c.GetProvider(); got != base.ProviderLocal {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderLocal)
	}
}

func TestLocalServiceProvider(t *testing.T) {
	svc := NewLocalService(NewLocalTTSConfig("say"))
	if got := svc.Provider(); got != base.ProviderLocal {
		t.Errorf("Provider() = %q, want %q", got, base.ProviderLocal)
	}
}

func TestLocalServiceFormat(t *testing.T) {
	svc := NewLocalService(NewLocalTTSConfig("say"))
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
	if f.Codec != "wav" {
		t.Errorf("Codec = %q, want %q", f.Codec, "wav")
	}
	if f.FrameDuration != 20*time.Millisecond {
		t.Errorf("FrameDuration = %v, want 20ms", f.FrameDuration)
	}
}

func TestLocalServiceCacheKey(t *testing.T) {
	svc := NewLocalService(NewLocalTTSConfig("say"))
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
	if !strings.HasPrefix(k1, "local.tts-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "local.tts-", k1)
	}
	if !strings.HasSuffix(k1, ".wav") {
		t.Errorf("CacheKey should end with %q, got %q", ".wav", k1)
	}
}

func TestLocalServiceCacheKeyReflectsConfig(t *testing.T) {
	svc1 := NewLocalService(NewLocalTTSConfig("say"))
	svc2 := NewLocalService(NewLocalTTSConfig("espeak"))
	if svc1.CacheKey("hi") == svc2.CacheKey("hi") {
		t.Error("CacheKey should differ when command differs")
	}
}

func TestLocalServiceClose(t *testing.T) {
	svc := NewLocalService(NewLocalTTSConfig("say"))
	if err := svc.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Detection helpers
// -----------------------------------------------------------------------------

func TestDetectLocalTTSCommandReturnsString(t *testing.T) {
	// DetectLocalTTSCommand returns a string; it may be empty when no TTS tool
	// is installed, but it must never panic and must be a string type.
	cmd := DetectLocalTTSCommand()
	// no assertion on emptiness; just ensure it runs and is a string.
	_ = cmd
}

func TestCheckLocalTTSAvailableExcludesNonExistentCommand(t *testing.T) {
	available := CheckLocalTTSAvailable()
	// A non-existent command must never be reported as available.
	const bogus = "definitely-not-a-real-tts-cmd-12345"
	for _, c := range available {
		if c == bogus {
			t.Errorf("CheckLocalTTSAvailable reported non-existent command %q", bogus)
		}
	}
	// Only known commands should ever appear.
	known := map[string]bool{"say": true, "espeak": true, "festival": true}
	for _, c := range available {
		if !known[c] {
			t.Errorf("CheckLocalTTSAvailable returned unexpected command %q", c)
		}
	}
}

// -----------------------------------------------------------------------------
// LocalGoSpeechService
// -----------------------------------------------------------------------------

func TestNewLocalGoSpeechConfigDefaults(t *testing.T) {
	c := NewLocalGoSpeechConfig(LocalGoSpeechProviderEspeak, "/path/to/model")
	if c.Provider != LocalGoSpeechProviderEspeak {
		t.Errorf("Provider = %q, want %q", c.Provider, LocalGoSpeechProviderEspeak)
	}
	if c.ModelPath != "/path/to/model" {
		t.Errorf("ModelPath = %q, want %q", c.ModelPath, "/path/to/model")
	}
	if c.Language != "zh-CN" {
		t.Errorf("Language = %q, want %q", c.Language, "zh-CN")
	}
	if c.Speaker != "default" {
		t.Errorf("Speaker = %q, want %q", c.Speaker, "default")
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
	if c.Speed != 1.0 {
		t.Errorf("Speed = %v, want 1.0", c.Speed)
	}
	if c.Pitch != 1.0 {
		t.Errorf("Pitch = %v, want 1.0", c.Pitch)
	}
	if c.Volume != 1.0 {
		t.Errorf("Volume = %v, want 1.0", c.Volume)
	}
	if !c.EnableCache {
		t.Errorf("EnableCache = %v, want true", c.EnableCache)
	}
	if c.CacheExpiry != 24*time.Hour {
		t.Errorf("CacheExpiry = %v, want 24h", c.CacheExpiry)
	}
	if c.OutputDir != "/tmp" {
		t.Errorf("OutputDir = %q, want %q", c.OutputDir, "/tmp")
	}
}

func TestNewLocalGoSpeechConfigGetProvider(t *testing.T) {
	c := NewLocalGoSpeechConfig(LocalGoSpeechProviderEspeak, "")
	if got := c.GetProvider(); got != base.ProviderLocalGoSpeech {
		t.Errorf("GetProvider() = %q, want %q", got, base.ProviderLocalGoSpeech)
	}
}

func TestLocalGoSpeechServiceProvider(t *testing.T) {
	// Construct the service directly to avoid the command-availability check
	// performed by NewLocalGoSpeechService (which would fail in environments
	// without espeak/say/festival installed).
	svc := &LocalGoSpeechService{
		config: NewLocalGoSpeechConfig(LocalGoSpeechProviderEspeak, ""),
	}
	got := svc.Provider()
	// Provider() returns a dynamic "local-gospeech-<provider>" identifier.
	wantPrefix := "local-gospeech-" + string(LocalGoSpeechProviderEspeak)
	if string(got) != wantPrefix {
		t.Errorf("Provider() = %q, want %q", got, wantPrefix)
	}
}

func TestLocalGoSpeechServiceFormat(t *testing.T) {
	svc := &LocalGoSpeechService{
		config: NewLocalGoSpeechConfig(LocalGoSpeechProviderEspeak, ""),
	}
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

func TestLocalGoSpeechServiceCacheKey(t *testing.T) {
	svc := &LocalGoSpeechService{
		config: NewLocalGoSpeechConfig(LocalGoSpeechProviderEspeak, ""),
	}
	k1 := svc.CacheKey("hello")
	k2 := svc.CacheKey("hello")
	k3 := svc.CacheKey("world")

	if k1 == "" {
		t.Fatal("CacheKey should not be empty when cache enabled")
	}
	if k1 != k2 {
		t.Errorf("CacheKey not deterministic: %q != %q", k1, k2)
	}
	if k1 == k3 {
		t.Errorf("CacheKey should differ for different text: %q == %q", k1, k3)
	}
	if !strings.HasPrefix(k1, "local-gospeech-") {
		t.Errorf("CacheKey should have prefix %q, got %q", "local-gospeech-", k1)
	}
}

func TestLocalGoSpeechServiceCacheKeyDisabled(t *testing.T) {
	cfg := NewLocalGoSpeechConfig(LocalGoSpeechProviderEspeak, "")
	cfg.EnableCache = false
	svc := &LocalGoSpeechService{config: cfg}
	if got := svc.CacheKey("hello"); got != "" {
		t.Errorf("CacheKey when cache disabled = %q, want empty", got)
	}
}
