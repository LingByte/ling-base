package synthesizer

import (
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/recognizer"
)

func TestNewGladiaASROptionDefaults(t *testing.T) {
	opt := NewGladiaASROption("gladia-key")
	if opt.APIKey != "gladia-key" {
		t.Errorf("APIKey = %q, want gladia-key", opt.APIKey)
	}
	if opt.URL != "wss://api.gladia.io/audio/text/streaming" {
		t.Errorf("URL = %q, want default gladia URL", opt.URL)
	}
	if opt.Language != "english" {
		t.Errorf("Language = %q, want english", opt.Language)
	}
	if opt.Model != "fast-conversational" {
		t.Errorf("Model = %q, want fast-conversational", opt.Model)
	}
	if opt.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", opt.SampleRate)
	}
	if opt.Encoding != "wav/pcm" {
		t.Errorf("Encoding = %q, want wav/pcm", opt.Encoding)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
}

func TestGladiaASROptionGetVendor(t *testing.T) {
	opt := NewGladiaASROption("k")
	if got := opt.GetVendor(); got != base.VendorGladia {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorGladia)
	}
}

func TestGladiaVendor(t *testing.T) {
	g := NewGladiaASR(NewGladiaASROption("k"))
	if got := g.Vendor(); got != "gladia" {
		t.Errorf("Vendor() = %q, want gladia", got)
	}
}

func TestGladiaInitStoresCallbacks(t *testing.T) {
	g := NewGladiaASR(NewGladiaASROption("k"))
	var gotText string
	var gotErr error
	tr := func(text string, isLast bool, dur time.Duration, dialogID string) {
		gotText = text
	}
	er := func(err error, isFatal bool) {
		gotErr = err
	}
	g.Init(tr, er)
	if g.tr == nil {
		t.Fatal("tr callback not stored")
	}
	if g.er == nil {
		t.Fatal("er callback not stored")
	}
	g.tr("hello", true, 0, "dlg")
	if gotText != "hello" {
		t.Errorf("tr callback yielded %q, want hello", gotText)
	}
	g.er(base.ErrClientClosed, true)
	if gotErr != base.ErrClientClosed {
		t.Errorf("er callback yielded %v, want ErrClientClosed", gotErr)
	}
}
