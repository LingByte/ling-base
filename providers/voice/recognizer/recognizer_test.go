package recognizer

import (
	"testing"
)

func TestNewRecognizer(t *testing.T) {
	cfg := DefaultConfig()
	r := NewRecognizer(cfg)
	if r == nil {
		t.Fatal("NewRecognizer returned nil")
	}
	if r.client == nil {
		t.Error("client should not be nil")
	}
	if r.targetBufferSize <= 0 {
		t.Error("targetBufferSize should be positive")
	}
}

func TestNewRecognizerCustomBufferDuration(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Buffer.SegmentDurationMs = 100
	cfg.Audio.Rate = 8000
	cfg.Audio.Bits = 16
	cfg.Audio.Channel = 1

	r := NewRecognizer(cfg)
	if r == nil {
		t.Fatal("NewRecognizer returned nil")
	}

	// Buffer size = (16/8) * 1 * 8000 / 1000 * 100 = 1600
	expected := 2 * 1 * 8000 / 1000 * 100
	if r.targetBufferSize != expected {
		t.Errorf("targetBufferSize = %d, want %d", r.targetBufferSize, expected)
	}
}

func TestNewRecognizerDefaultBufferDuration(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Buffer.SegmentDurationMs = 0 // should default to 100ms

	r := NewRecognizer(cfg)
	if r == nil {
		t.Fatal("NewRecognizer returned nil")
	}

	// With 100ms default: (16/8) * 1 * 16000 / 1000 * 100 = 3200
	expected := 2 * 1 * 16000 / 1000 * 100
	if r.targetBufferSize != expected {
		t.Errorf("targetBufferSize = %d, want %d", r.targetBufferSize, expected)
	}
}

func TestRecognizerOnResult(t *testing.T) {
	cfg := DefaultConfig()
	r := NewRecognizer(cfg)

	var called bool
	r.OnResult(func(_ *Result) {
		called = true
	})

	if r.onResult == nil {
		t.Error("onResult callback was not set")
	}

	// Call the callback directly
	r.onResult(&Result{Text: "test"})
	if !called {
		t.Error("onResult callback was not called")
	}
}

func TestRecognizerOnError(t *testing.T) {
	cfg := DefaultConfig()
	r := NewRecognizer(cfg)

	var called bool
	r.OnError(func(_ error) {
		called = true
	})

	if r.onError == nil {
		t.Error("onError callback was not set")
	}

	// Call the callback directly
	r.onError(nil)
	if !called {
		t.Error("onError callback was not called")
	}
}

func TestConvertResponseToResult(t *testing.T) {
	cfg := DefaultConfig()
	r := NewRecognizer(cfg)

	// Test successful response
	resp := &Response{
		Code:          0,
		IsLastPackage: false,
		PayloadMsg: &ResponsePayload{
			Result: struct {
				Text       string `json:"text"`
				Utterances []struct {
					Definite  bool   `json:"definite"`
					EndTime   int    `json:"end_time"`
					StartTime int    `json:"start_time"`
					Text      string `json:"text"`
					Words     []struct {
						EndTime   int    `json:"end_time"`
						StartTime int    `json:"start_time"`
						Text      string `json:"text"`
					} `json:"words"`
				} `json:"utterances,omitempty"`
			}{
				Text: "hello world",
			},
		},
	}

	result := r.convertResponseToResult(resp)
	if result.Text != "hello world" {
		t.Errorf("Text = %q, want %q", result.Text, "hello world")
	}
	if result.IsFinal {
		t.Error("IsFinal should be false")
	}
	if result.Error != nil {
		t.Errorf("Error should be nil, got %v", result.Error)
	}
}

func TestConvertResponseToResultError(t *testing.T) {
	cfg := DefaultConfig()
	r := NewRecognizer(cfg)

	resp := &Response{
		Code: 1001,
	}

	result := r.convertResponseToResult(resp)
	if result.Error == nil {
		t.Error("Error should not be nil for non-zero code")
	}
}

func TestConvertResponseToResultFinal(t *testing.T) {
	cfg := DefaultConfig()
	r := NewRecognizer(cfg)

	resp := &Response{
		Code:          0,
		IsLastPackage: true,
	}

	result := r.convertResponseToResult(resp)
	if !result.IsFinal {
		t.Error("IsFinal should be true for IsLastPackage")
	}
}

func TestGetTraceIDEmpty(t *testing.T) {
	cfg := DefaultConfig()
	r := NewRecognizer(cfg)
	if r.GetTraceID() != "" {
		t.Errorf("GetTraceID() = %q, want empty", r.GetTraceID())
	}
}
