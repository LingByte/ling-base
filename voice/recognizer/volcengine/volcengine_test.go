package synthesizer

import (
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/recognizer"
)

func TestNewVolcengineOptionDefaults(t *testing.T) {
	opt := NewVolcengineOption("app-123", "tok-456", "", "")
	if opt.Url != "wss://openspeech.bytedance.com/api/v2/asr" {
		t.Errorf("Url = %q, want default wss URL", opt.Url)
	}
	if opt.AppID != "app-123" {
		t.Errorf("AppID = %q, want app-123", opt.AppID)
	}
	if opt.Token != "tok-456" {
		t.Errorf("Token = %q, want tok-456", opt.Token)
	}
	if opt.Cluster != "volcengine_input_common" {
		t.Errorf("Cluster = %q, want volcengine_input_common", opt.Cluster)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
	if opt.WorkFlow != "audio_in,resample,partition,vad,fe,decode" {
		t.Errorf("WorkFlow = %q, want default workflow", opt.WorkFlow)
	}
	if opt.Codec != "raw" {
		t.Errorf("Codec = %q, want raw", opt.Codec)
	}
	if opt.Format != "raw" {
		t.Errorf("Format = %q, want raw", opt.Format)
	}
	if opt.EndWindowSize != base.DefaultVolcEndWindowMs() {
		t.Errorf("EndWindowSize = %d, want %d", opt.EndWindowSize, base.DefaultVolcEndWindowMs())
	}
}

func TestVolcengineOptionGetVendor(t *testing.T) {
	opt := NewVolcengineOption("a", "b", "c", "d")
	if got := opt.GetVendor(); got != base.VendorVolcengine {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorVolcengine)
	}
}

func TestVolcengineVendor(t *testing.T) {
	volc := NewVolcengineASR(NewVolcengineOption("a", "b", "c", "d"))
	if got := volc.Vendor(); got != "volcengine" {
		t.Errorf("Vendor() = %q, want volcengine", got)
	}
}

func TestVolcengineInitStoresCallbacks(t *testing.T) {
	volc := NewVolcengineASR(NewVolcengineOption("a", "b", "c", "d"))
	var gotText string
	var gotErr error
	tr := func(text string, isLast bool, dur time.Duration, dialogID string) {
		gotText = text
	}
	er := func(err error, isFatal bool) {
		gotErr = err
	}
	volc.Init(tr, er)
	if volc.tr == nil {
		t.Fatal("tr callback not stored")
	}
	if volc.er == nil {
		t.Fatal("er callback not stored")
	}
	// Invoke callbacks to confirm they are the ones we passed.
	volc.tr("hello", true, 0, "dlg-1")
	if gotText != "hello" {
		t.Errorf("tr callback yielded %q, want hello", gotText)
	}
	volc.er(base.ErrClientClosed, true)
	if gotErr != base.ErrClientClosed {
		t.Errorf("er callback yielded %v, want ErrClientClosed", gotErr)
	}
}

func TestVolcengineEffectiveEndWindowSizeClamping(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"zero falls back to default", 0, base.DefaultVolcEndWindowMs()},
		{"negative falls back to default", -5, base.DefaultVolcEndWindowMs()},
		{"below min clamps to 200", 100, 200},
		{"min boundary 200", 200, 200},
		{"above max clamps to 3000", 5000, 3000},
		{"max boundary 3000", 3000, 3000},
		{"in range unchanged", 500, 500},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opt := VolcengineOption{EndWindowSize: c.in}
			if got := opt.effectiveEndWindowSize(); got != c.want {
				t.Errorf("effectiveEndWindowSize(%d) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestGzipCompressDecompressRoundTrip(t *testing.T) {
	inputs := [][]byte{
		[]byte(""),
		[]byte("hello world"),
		[]byte("火山引擎语音识别"),
		[]byte("the quick brown fox jumps over the lazy dog 1234567890"),
	}
	for _, in := range inputs {
		compressed, err := gzipCompress(in)
		if err != nil {
			t.Fatalf("gzipCompress(%q) error: %v", in, err)
		}
		out, err := gzipDecompress(compressed)
		if err != nil {
			t.Fatalf("gzipDecompress error: %v", err)
		}
		if string(out) != string(in) {
			t.Errorf("round-trip mismatch: got %q, want %q", out, in)
		}
	}
}
