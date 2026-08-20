package synthesizer

import (
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/recognizer"
)

func TestNewGoogleASROptionDefaults(t *testing.T) {
	opt := NewGoogleASROption("{\"type\":\"service_account\"}")
	if opt.CredentialsJSON != "{\"type\":\"service_account\"}" {
		t.Errorf("CredentialsJSON = %q, want the provided credentials", opt.CredentialsJSON)
	}
	if opt.LanguageCode != "en-US" {
		t.Errorf("LanguageCode = %q, want en-US", opt.LanguageCode)
	}
	if opt.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", opt.SampleRate)
	}
	if opt.Encoding != "LINEAR16" {
		t.Errorf("Encoding = %q, want LINEAR16", opt.Encoding)
	}
	if opt.Model != "latest_long" {
		t.Errorf("Model = %q, want latest_long", opt.Model)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
}

func TestGoogleASROptionGetVendor(t *testing.T) {
	opt := NewGoogleASROption("{}")
	if got := opt.GetVendor(); got != base.VendorGoogle {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorGoogle)
	}
}

func TestGoogleVendor(t *testing.T) {
	g := NewGoogleASR(NewGoogleASROption("{}"))
	if got := g.Vendor(); got != "google" {
		t.Errorf("Vendor() = %q, want google", got)
	}
}

func TestGoogleInitStoresCallbacks(t *testing.T) {
	g := NewGoogleASR(NewGoogleASROption("{}"))
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
