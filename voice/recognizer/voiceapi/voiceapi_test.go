package synthesizer

import (
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/recognizer"
)

func TestNewVoiceAPIASROptionDefaults(t *testing.T) {
	opt := NewVoiceAPIASROption("wss://voiceapi/ws", "vapi-key")
	if opt.URL != "wss://voiceapi/ws" {
		t.Errorf("URL = %q, want wss://voiceapi/ws", opt.URL)
	}
	if opt.APIKey != "vapi-key" {
		t.Errorf("APIKey = %q, want vapi-key", opt.APIKey)
	}
	if opt.Model != "default" {
		t.Errorf("Model = %q, want default", opt.Model)
	}
	if opt.Language != "zh-CN" {
		t.Errorf("Language = %q, want zh-CN", opt.Language)
	}
	if opt.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", opt.SampleRate)
	}
	if opt.Encoding != "pcm" {
		t.Errorf("Encoding = %q, want pcm", opt.Encoding)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
}

func TestVoiceAPIASROptionGetVendor(t *testing.T) {
	opt := NewVoiceAPIASROption("wss://x", "k")
	if got := opt.GetVendor(); got != base.VendorVoiceAPI {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorVoiceAPI)
	}
}

func TestVoiceAPIVendor(t *testing.T) {
	vapi := NewVoiceAPIASR(NewVoiceAPIASROption("wss://x", "k"))
	if got := vapi.Vendor(); got != "voiceapi" {
		t.Errorf("Vendor() = %q, want voiceapi", got)
	}
}

func TestVoiceAPIInitStoresCallbacks(t *testing.T) {
	vapi := NewVoiceAPIASR(NewVoiceAPIASROption("wss://x", "k"))
	var gotText string
	var gotErr error
	tr := func(text string, isLast bool, dur time.Duration, dialogID string) {
		gotText = text
	}
	er := func(err error, isFatal bool) {
		gotErr = err
	}
	vapi.Init(tr, er)
	if vapi.tr == nil {
		t.Fatal("tr callback not stored")
	}
	if vapi.er == nil {
		t.Fatal("er callback not stored")
	}
	vapi.tr("hello", true, 0, "dlg")
	if gotText != "hello" {
		t.Errorf("tr callback yielded %q, want hello", gotText)
	}
	vapi.er(base.ErrClientClosed, true)
	if gotErr != base.ErrClientClosed {
		t.Errorf("er callback yielded %v, want ErrClientClosed", gotErr)
	}
}
