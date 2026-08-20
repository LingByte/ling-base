package synthesizer

import (
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/recognizer"
)

func TestNewDeepgramASROptionDefaults(t *testing.T) {
	opt := NewDeepgramASROption("dg-key")
	if opt.APIKey != "dg-key" {
		t.Errorf("APIKey = %q, want dg-key", opt.APIKey)
	}
	if opt.URL != "wss://api.deepgram.com/v1/listen" {
		t.Errorf("URL = %q, want default deepgram URL", opt.URL)
	}
	if opt.Language != "en-US" {
		t.Errorf("Language = %q, want en-US", opt.Language)
	}
	if opt.Model != "nova-2" {
		t.Errorf("Model = %q, want nova-2", opt.Model)
	}
	if opt.Encoding != "linear16" {
		t.Errorf("Encoding = %q, want linear16", opt.Encoding)
	}
	if opt.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", opt.SampleRate)
	}
	if opt.Channels != 1 {
		t.Errorf("Channels = %d, want 1", opt.Channels)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
}

func TestDeepgramASROptionGetVendor(t *testing.T) {
	opt := NewDeepgramASROption("k")
	if got := opt.GetVendor(); got != base.VendorDeepgram {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorDeepgram)
	}
}

func TestDeepgramVendor(t *testing.T) {
	dg := NewDeepgramASR(NewDeepgramASROption("k"))
	if got := dg.Vendor(); got != "deepgram" {
		t.Errorf("Vendor() = %q, want deepgram", got)
	}
}

func TestDeepgramInitStoresCallbacks(t *testing.T) {
	dg := NewDeepgramASR(NewDeepgramASROption("k"))
	var gotText string
	var gotErr error
	tr := func(text string, isLast bool, dur time.Duration, dialogID string) {
		gotText = text
	}
	er := func(err error, isFatal bool) {
		gotErr = err
	}
	dg.Init(tr, er)
	if dg.tr == nil {
		t.Fatal("tr callback not stored")
	}
	if dg.er == nil {
		t.Fatal("er callback not stored")
	}
	dg.tr("hello", true, 0, "dlg")
	if gotText != "hello" {
		t.Errorf("tr callback yielded %q, want hello", gotText)
	}
	dg.er(base.ErrClientClosed, true)
	if gotErr != base.ErrClientClosed {
		t.Errorf("er callback yielded %v, want ErrClientClosed", gotErr)
	}
}
