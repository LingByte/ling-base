package synthesizer

import (
	"context"
	"testing"
	"time"
)

func TestProviderToString(t *testing.T) {
	p := ProviderOpenAI
	if p.ToString() != "openai" {
		t.Errorf("ToString() = %q, want %q", p.ToString(), "openai")
	}
}

func TestDefaultFormat(t *testing.T) {
	f := DefaultFormat()
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

func TestComputeSampleByteCount(t *testing.T) {
	tests := []struct {
		sampleRate int
		bitDepth   int
		channels   int
		want       int
	}{
		{16000, 16, 1, 32000},
		{24000, 16, 1, 48000},
		{16000, 8, 1, 16000},
		{16000, 16, 2, 64000},
		{0, 16, 1, 0},
	}
	for _, tt := range tests {
		got := ComputeSampleByteCount(tt.sampleRate, tt.bitDepth, tt.channels)
		if got != tt.want {
			t.Errorf("ComputeSampleByteCount(%d, %d, %d) = %d, want %d",
				tt.sampleRate, tt.bitDepth, tt.channels, got, tt.want)
		}
	}
}

func TestNormalizeFramePeriod(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"20ms", 20 * time.Millisecond},
		{"10ms", 10 * time.Millisecond},
		{"50ms", 50 * time.Millisecond},
		{"100ms", 100 * time.Millisecond},
		{"300ms", 300 * time.Millisecond},
		{"5ms", 20 * time.Millisecond},     // too small → default
		{"350ms", 20 * time.Millisecond},   // too large → default
		{"0ms", 20 * time.Millisecond},     // zero → default
		{"invalid", 20 * time.Millisecond}, // parse error → default
		{"", 20 * time.Millisecond},        // empty → default
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeFramePeriod(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeFramePeriod(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestHashText(t *testing.T) {
	hash1 := HashText("hello world")
	hash2 := HashText("hello world")
	hash3 := HashText("hello world!")

	if hash1 != hash2 {
		t.Error("HashText should be deterministic for same input")
	}
	if hash1 == hash3 {
		t.Error("HashText should produce different hashes for different input")
	}
	if len(hash1) != 16 {
		t.Errorf("HashText length = %d, want 16", len(hash1))
	}
}

func TestHashTextEmpty(t *testing.T) {
	hash := HashText("")
	if len(hash) != 16 {
		t.Errorf("HashText(empty) length = %d, want 16", len(hash))
	}
}

func TestStripEmoji(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", "hello world"},
		{"hello 😀 world", "hello  world"},
		{"no emoji here", "no emoji here"},
		{"🎉party🎉", "party"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := StripEmoji(tt.input)
			if got != tt.want {
				t.Errorf("StripEmoji(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHandlerFunc(t *testing.T) {
	var messageCalled bool
	var timestampCalled bool
	var receivedData []byte

	h := HandlerFunc{
		OnMessageFn: func(data []byte) {
			messageCalled = true
			receivedData = data
		},
		OnTimestampFn: func(ts SentenceTimestamp) {
			timestampCalled = true
		},
	}

	h.OnMessage([]byte("test"))
	if !messageCalled {
		t.Error("OnMessage was not called")
	}
	if string(receivedData) != "test" {
		t.Errorf("receivedData = %q, want %q", string(receivedData), "test")
	}

	h.OnTimestamp(SentenceTimestamp{})
	if !timestampCalled {
		t.Error("OnTimestamp was not called")
	}
}

func TestHandlerFuncNilCallbacks(t *testing.T) {
	h := HandlerFunc{}
	// Should not panic with nil callbacks
	h.OnMessage([]byte("test"))
	h.OnTimestamp(SentenceTimestamp{})
}

func TestSynthesisBuffer(t *testing.T) {
	buf := &SynthesisBuffer{}
	buf.OnMessage([]byte("chunk1"))
	buf.OnMessage([]byte("chunk2"))
	buf.OnMessage([]byte("chunk3"))

	if string(buf.Data) != "chunk1chunk2chunk3" {
		t.Errorf("Data = %q, want %q", string(buf.Data), "chunk1chunk2chunk3")
	}

	ts := SentenceTimestamp{Words: []Word{{Word: "hello", StartTime: 0, EndTime: 100}}}
	buf.OnTimestamp(ts)
	if len(buf.Timestamp.Words) != 1 {
		t.Errorf("Timestamp.Words length = %d, want 1", len(buf.Timestamp.Words))
	}
	if buf.Timestamp.Words[0].Word != "hello" {
		t.Errorf("Timestamp.Words[0].Word = %q, want %q", buf.Timestamp.Words[0].Word, "hello")
	}
}

func TestWordStruct(t *testing.T) {
	w := Word{
		Confidence: 0.95,
		StartTime:  100,
		EndTime:    200,
		Word:       "test",
	}
	if w.Confidence != 0.95 {
		t.Errorf("Confidence = %f, want 0.95", w.Confidence)
	}
	if w.Word != "test" {
		t.Errorf("Word = %q, want %q", w.Word, "test")
	}
}

func TestSentenceTimestampStruct(t *testing.T) {
	ts := SentenceTimestamp{
		Words: []Word{
			{Word: "hello", StartTime: 0, EndTime: 100},
			{Word: "world", StartTime: 100, EndTime: 200},
		},
	}
	if len(ts.Words) != 2 {
		t.Errorf("Words length = %d, want 2", len(ts.Words))
	}
}

// mockEngine implements Engine for testing.
type mockEngine struct {
	provider Provider
	format   StreamFormat
}

func (m *mockEngine) Provider() Provider          { return m.provider }
func (m *mockEngine) Format() StreamFormat        { return m.format }
func (m *mockEngine) CacheKey(text string) string { return "mock-" + HashText(text) }
func (m *mockEngine) Synthesize(_ context.Context, h Handler, _ string) error {
	h.OnMessage([]byte("mock audio"))
	return nil
}
func (m *mockEngine) Close() error { return nil }

func TestEngineInterface(t *testing.T) {
	var e Engine = &mockEngine{
		provider: ProviderOpenAI,
		format:   DefaultFormat(),
	}
	if e.Provider() != ProviderOpenAI {
		t.Errorf("Provider() = %q, want %q", e.Provider(), ProviderOpenAI)
	}
	if e.Format().SampleRate != 16000 {
		t.Errorf("Format().SampleRate = %d, want 16000", e.Format().SampleRate)
	}
	if e.CacheKey("test") == "" {
		t.Error("CacheKey() should not be empty")
	}

	buf := &SynthesisBuffer{}
	if err := e.Synthesize(context.Background(), buf, "test"); err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}
	if string(buf.Data) != "mock audio" {
		t.Errorf("Data = %q, want %q", string(buf.Data), "mock audio")
	}
	if err := e.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
