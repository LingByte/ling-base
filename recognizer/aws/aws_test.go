package synthesizer

import (
	"testing"
	"time"

	base "github.com/LingByte/ling-base/recognizer"
)

func TestNewAWSASROptionDefaults(t *testing.T) {
	opt := NewAWSASROption("ak", "sk", "")
	if opt.AccessKey != "ak" {
		t.Errorf("AccessKey = %q, want ak", opt.AccessKey)
	}
	if opt.SecretKey != "sk" {
		t.Errorf("SecretKey = %q, want sk", opt.SecretKey)
	}
	if opt.Region != "us-east-1" {
		t.Errorf("Region = %q, want us-east-1", opt.Region)
	}
	if opt.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", opt.SampleRate)
	}
	if opt.AudioEncoding != "pcm" {
		t.Errorf("AudioEncoding = %q, want pcm", opt.AudioEncoding)
	}
	if opt.LanguageCode != "en-US" {
		t.Errorf("LanguageCode = %q, want en-US", opt.LanguageCode)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
}

func TestNewAWSASROptionExplicitRegion(t *testing.T) {
	opt := NewAWSASROption("ak", "sk", "eu-west-1")
	if opt.Region != "eu-west-1" {
		t.Errorf("Region = %q, want eu-west-1", opt.Region)
	}
}

func TestAWSASROptionGetVendor(t *testing.T) {
	opt := NewAWSASROption("ak", "sk", "us-east-1")
	if got := opt.GetVendor(); got != base.VendorAWS {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorAWS)
	}
}

func TestAWSVendor(t *testing.T) {
	a := NewAWSASR(NewAWSASROption("ak", "sk", "us-east-1"))
	if got := a.Vendor(); got != "aws" {
		t.Errorf("Vendor() = %q, want aws", got)
	}
}

func TestAWSInitStoresCallbacks(t *testing.T) {
	a := NewAWSASR(NewAWSASROption("ak", "sk", "us-east-1"))
	var gotText string
	var gotErr error
	tr := func(text string, isLast bool, dur time.Duration, dialogID string) {
		gotText = text
	}
	er := func(err error, isFatal bool) {
		gotErr = err
	}
	a.Init(tr, er)
	if a.tr == nil {
		t.Fatal("tr callback not stored")
	}
	if a.er == nil {
		t.Fatal("er callback not stored")
	}
	a.tr("hello", true, 0, "dlg")
	if gotText != "hello" {
		t.Errorf("tr callback yielded %q, want hello", gotText)
	}
	a.er(base.ErrClientClosed, true)
	if gotErr != base.ErrClientClosed {
		t.Errorf("er callback yielded %v, want ErrClientClosed", gotErr)
	}
}
