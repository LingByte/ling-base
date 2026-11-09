package recognizer

import (
	"errors"
	"testing"
	"time"
)

func TestComputeSampleByteCount(t *testing.T) {
	tests := []struct {
		name       string
		sampleRate int
		bitDepth   int
		channels   int
		want       int
	}{
		{"16k/16bit/mono", 16000, 16, 1, 32000},
		{"24k/16bit/mono", 24000, 16, 1, 48000},
		{"16k/8bit/mono", 16000, 8, 1, 16000},
		{"16k/16bit/stereo", 16000, 16, 2, 64000},
		{"44.1k/16bit/stereo", 44100, 16, 2, 176400},
		{"zero", 0, 16, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeSampleByteCount(tt.sampleRate, tt.bitDepth, tt.channels)
			if got != tt.want {
				t.Errorf("ComputeSampleByteCount(%d, %d, %d) = %d, want %d",
					tt.sampleRate, tt.bitDepth, tt.channels, got, tt.want)
			}
		})
	}
}

func TestDefaultTimeoutConfig(t *testing.T) {
	cfg := DefaultTimeoutConfig()
	if cfg.Send != 10*time.Second {
		t.Errorf("Send = %v, want 10s", cfg.Send)
	}
	if cfg.Read != 30*time.Second {
		t.Errorf("Read = %v, want 30s", cfg.Read)
	}
}

func TestErrClientClosed(t *testing.T) {
	err := ErrClientClosed
	if err.Error() != "asr client closed" {
		t.Errorf("Error() = %q, want %q", err.Error(), "asr client closed")
	}

	if !errors.Is(err, ErrClientClosed) {
		t.Error("errors.Is(ErrClientClosed, ErrClientClosed) should be true")
	}

	if errors.Is(err, errors.New("other")) {
		t.Error("errors.Is should return false for unrelated error")
	}
}

func TestResultStruct(t *testing.T) {
	now := time.Now()
	r := Result{
		Text:      "hello",
		IsFinal:   true,
		Timestamp: now,
	}
	if r.Text != "hello" {
		t.Errorf("Text = %q, want %q", r.Text, "hello")
	}
	if !r.IsFinal {
		t.Error("IsFinal should be true")
	}
	if !r.Timestamp.Equal(now) {
		t.Error("Timestamp mismatch")
	}
}

func TestHotWordStruct(t *testing.T) {
	hw := HotWord{Word: "test", Weight: 10}
	if hw.Word != "test" {
		t.Errorf("Word = %q, want %q", hw.Word, "test")
	}
	if hw.Weight != 10 {
		t.Errorf("Weight = %d, want 10", hw.Weight)
	}
}
