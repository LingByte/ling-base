package synthesizer

import (
	"testing"
	"time"

	base "github.com/LingByte/ling-base/recognizer"
)

func TestNewQcloudASROptionDefaults(t *testing.T) {
	opt := NewQcloudASROption("app-1", "sid-2", "skey-3")
	if opt.AppID != "app-1" {
		t.Errorf("AppID = %q, want app-1", opt.AppID)
	}
	if opt.SecretID != "sid-2" {
		t.Errorf("SecretID = %q, want sid-2", opt.SecretID)
	}
	if opt.SecretKey != "skey-3" {
		t.Errorf("SecretKey = %q, want skey-3", opt.SecretKey)
	}
	if opt.Format != 1 {
		t.Errorf("Format = %d, want 1 (PCM)", opt.Format)
	}
	if opt.ModelType != "16k_zh" {
		t.Errorf("ModelType = %q, want 16k_zh", opt.ModelType)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
	if opt.VadSilenceTime != defaultQCloudVadSilenceMs {
		t.Errorf("VadSilenceTime = %d, want %d", opt.VadSilenceTime, defaultQCloudVadSilenceMs)
	}
}

func TestQcloudASROptionGetVendor(t *testing.T) {
	opt := NewQcloudASROption("a", "b", "c")
	if got := opt.GetVendor(); got != base.VendorQCloud {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorQCloud)
	}
}

func TestQcloudVendor(t *testing.T) {
	asq := NewQcloudASR(NewQcloudASROption("a", "b", "c"))
	if got := asq.Vendor(); got != "qcloud" {
		t.Errorf("Vendor() = %q, want qcloud", got)
	}
}

func TestQcloudInitStoresCallbacks(t *testing.T) {
	asq := NewQcloudASR(NewQcloudASROption("a", "b", "c"))
	var gotText string
	var gotErr error
	tr := func(text string, isLast bool, dur time.Duration, dialogID string) {
		gotText = text
	}
	er := func(err error, isFatal bool) {
		gotErr = err
	}
	asq.Init(tr, er)
	if asq.transcribeResult == nil {
		t.Fatal("transcribeResult callback not stored")
	}
	if asq.processError == nil {
		t.Fatal("processError callback not stored")
	}
	asq.transcribeResult("hello", false, 0, "dlg")
	if gotText != "hello" {
		t.Errorf("transcribeResult yielded %q, want hello", gotText)
	}
	asq.processError(base.ErrClientClosed, true)
	if gotErr != base.ErrClientClosed {
		t.Errorf("processError yielded %v, want ErrClientClosed", gotErr)
	}
}

func TestQcloudEffectiveVadSilenceTimeClamping(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"zero falls back to default", 0, defaultQCloudVadSilenceMs},
		{"negative falls back to default", -10, defaultQCloudVadSilenceMs},
		{"below min clamps to 240", 100, 240},
		{"min boundary 240", 240, 240},
		{"above max clamps to 2000", 5000, 2000},
		{"max boundary 2000", 2000, 2000},
		{"in range unchanged", 800, 800},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opt := QCloudASROption{VadSilenceTime: c.in}
			if got := opt.effectiveVadSilenceTime(); got != c.want {
				t.Errorf("effectiveVadSilenceTime(%d) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}
