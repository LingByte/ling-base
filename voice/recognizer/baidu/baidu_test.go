package synthesizer

import (
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/recognizer"
)

func TestNewBaiduASROptionDefaults(t *testing.T) {
	opt := NewBaiduASROption("ak", "sk")
	if opt.APIKey != "ak" {
		t.Errorf("APIKey = %q, want ak", opt.APIKey)
	}
	if opt.SecretKey != "sk" {
		t.Errorf("SecretKey = %q, want sk", opt.SecretKey)
	}
	if opt.Format != baiduDefaultFormat {
		t.Errorf("Format = %q, want %q", opt.Format, baiduDefaultFormat)
	}
	if opt.Rate != baiduDefaultRate {
		t.Errorf("Rate = %d, want %d", opt.Rate, baiduDefaultRate)
	}
	if opt.Channel != baiduDefaultChannel {
		t.Errorf("Channel = %d, want %d", opt.Channel, baiduDefaultChannel)
	}
	if opt.Cuid != "ling-base-cuid" {
		t.Errorf("Cuid = %q, want ling-base-cuid", opt.Cuid)
	}
	if opt.DevPid != baiduDefaultDevPid {
		t.Errorf("DevPid = %d, want %d", opt.DevPid, baiduDefaultDevPid)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
}

func TestBaiduASROptionGetVendor(t *testing.T) {
	opt := NewBaiduASROption("ak", "sk")
	if got := opt.GetVendor(); got != base.VendorBaidu {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorBaidu)
	}
}

func TestBaiduVendor(t *testing.T) {
	b := NewBaiduASR(NewBaiduASROption("ak", "sk"))
	if got := b.Vendor(); got != "baidu" {
		t.Errorf("Vendor() = %q, want baidu", got)
	}
}

func TestBaiduInitStoresCallbacks(t *testing.T) {
	b := NewBaiduASR(NewBaiduASROption("ak", "sk"))
	var gotText string
	var gotErr error
	tr := func(text string, isLast bool, dur time.Duration, dialogID string) {
		gotText = text
	}
	er := func(err error, isFatal bool) {
		gotErr = err
	}
	b.Init(tr, er)
	if b.tr == nil {
		t.Fatal("tr callback not stored")
	}
	if b.er == nil {
		t.Fatal("er callback not stored")
	}
	b.tr("hello", true, 0, "dlg")
	if gotText != "hello" {
		t.Errorf("tr callback yielded %q, want hello", gotText)
	}
	b.er(base.ErrClientClosed, true)
	if gotErr != base.ErrClientClosed {
		t.Errorf("er callback yielded %v, want ErrClientClosed", gotErr)
	}
}
