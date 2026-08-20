package synthesizer

import (
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/recognizer"
)

func TestNewWhisperASROptionDefaults(t *testing.T) {
	opt := NewWhisperASROption("wss://example/ws", "key-1")
	if opt.URL != "wss://example/ws" {
		t.Errorf("URL = %q, want wss://example/ws", opt.URL)
	}
	if opt.APIKey != "key-1" {
		t.Errorf("APIKey = %q, want key-1", opt.APIKey)
	}
	if opt.Model != "base" {
		t.Errorf("Model = %q, want base", opt.Model)
	}
	if opt.Language != "en" {
		t.Errorf("Language = %q, want en", opt.Language)
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
	if opt.Format != "pcm" {
		t.Errorf("Format = %q, want pcm", opt.Format)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
}

func TestWhisperASROptionGetVendor(t *testing.T) {
	opt := NewWhisperASROption("wss://x", "k")
	if got := opt.GetVendor(); got != base.VendorWhisper {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorWhisper)
	}
}

func TestWhisperVendor(t *testing.T) {
	w := NewWhisperASR(NewWhisperASROption("wss://x", "k"))
	if got := w.Vendor(); got != "whisper" {
		t.Errorf("Vendor() = %q, want whisper", got)
	}
}

func TestWhisperInitStoresCallbacks(t *testing.T) {
	w := NewWhisperASR(NewWhisperASROption("wss://x", "k"))
	var gotText string
	var gotErr error
	tr := func(text string, isLast bool, dur time.Duration, dialogID string) {
		gotText = text
	}
	er := func(err error, isFatal bool) {
		gotErr = err
	}
	w.Init(tr, er)
	if w.tr == nil {
		t.Fatal("tr callback not stored")
	}
	if w.er == nil {
		t.Fatal("er callback not stored")
	}
	w.tr("hello", true, 0, "dlg")
	if gotText != "hello" {
		t.Errorf("tr callback yielded %q, want hello", gotText)
	}
	w.er(base.ErrClientClosed, true)
	if gotErr != base.ErrClientClosed {
		t.Errorf("er callback yielded %v, want ErrClientClosed", gotErr)
	}
}
