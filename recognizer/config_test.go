package recognizer

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	if cfg.Auth.ResourceId != "volc.bigasr.sauc.duration" {
		t.Errorf("Auth.ResourceId = %q, want %q", cfg.Auth.ResourceId, "volc.bigasr.sauc.duration")
	}
	if cfg.Audio.Rate != 16000 {
		t.Errorf("Audio.Rate = %d, want 16000", cfg.Audio.Rate)
	}
	if cfg.Audio.Bits != 16 {
		t.Errorf("Audio.Bits = %d, want 16", cfg.Audio.Bits)
	}
	if cfg.Audio.Channel != 1 {
		t.Errorf("Audio.Channel = %d, want 1", cfg.Audio.Channel)
	}
	if cfg.Request.ModelName != "bigmodel" {
		t.Errorf("Request.ModelName = %q, want %q", cfg.Request.ModelName, "bigmodel")
	}
	if cfg.Request.EndWindowSize != defaultVolcEndWindowMs {
		t.Errorf("Request.EndWindowSize = %d, want %d", cfg.Request.EndWindowSize, defaultVolcEndWindowMs)
	}
	if cfg.Buffer.SegmentDurationMs != 200 {
		t.Errorf("Buffer.SegmentDurationMs = %d, want 200", cfg.Buffer.SegmentDurationMs)
	}
}

func TestDefaultVolcEndWindowMs(t *testing.T) {
	if DefaultVolcEndWindowMs() != 300 {
		t.Errorf("DefaultVolcEndWindowMs() = %d, want 300", DefaultVolcEndWindowMs())
	}
}

func TestConfigBuilder(t *testing.T) {
	cfg := DefaultConfig().
		WithURL("wss://example.com").
		WithAuth(AuthConfig{ResourceId: "test", AccessKey: "key", AppKey: "app"}).
		WithUser(UserConfig{UID: "user123"}).
		WithAudio(AudioConfig{Format: "wav", Rate: 8000, Bits: 16, Channel: 1}).
		WithRequest(RequestConfig{ModelName: "test-model"}).
		WithBuffer(BufferConfig{SegmentDurationMs: 100})

	if cfg.URL != "wss://example.com" {
		t.Errorf("URL = %q, want %q", cfg.URL, "wss://example.com")
	}
	if cfg.Auth.ResourceId != "test" {
		t.Errorf("Auth.ResourceId = %q, want %q", cfg.Auth.ResourceId, "test")
	}
	if cfg.User.UID != "user123" {
		t.Errorf("User.UID = %q, want %q", cfg.User.UID, "user123")
	}
	if cfg.Audio.Format != "wav" {
		t.Errorf("Audio.Format = %q, want %q", cfg.Audio.Format, "wav")
	}
	if cfg.Audio.Rate != 8000 {
		t.Errorf("Audio.Rate = %d, want 8000", cfg.Audio.Rate)
	}
	if cfg.Request.ModelName != "test-model" {
		t.Errorf("Request.ModelName = %q, want %q", cfg.Request.ModelName, "test-model")
	}
	if cfg.Buffer.SegmentDurationMs != 100 {
		t.Errorf("Buffer.SegmentDurationMs = %d, want 100", cfg.Buffer.SegmentDurationMs)
	}
}

func TestCalculateBufferSize(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want int
	}{
		{
			name: "default 16k/16bit/mono 200ms",
			cfg:  DefaultConfig(),
			want: 2 * 1 * 16000 / 1000 * 200, // (16/8)*1*16000/1000 * 200 = 6400
		},
		{
			name: "with max buffer size override",
			cfg: &Config{
				Audio:  AudioConfig{Bits: 16, Channel: 1, Rate: 16000},
				Buffer: BufferConfig{SegmentDurationMs: 200, MaxBufferSize: 9999},
			},
			want: 9999,
		},
		{
			name: "8k/16bit/mono 100ms",
			cfg: &Config{
				Audio:  AudioConfig{Bits: 16, Channel: 1, Rate: 8000},
				Buffer: BufferConfig{SegmentDurationMs: 100},
			},
			want: 2 * 1 * 8000 / 1000 * 100, // 1600
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.CalculateBufferSize()
			if got != tt.want {
				t.Errorf("CalculateBufferSize() = %d, want %d", got, tt.want)
			}
		})
	}
}
