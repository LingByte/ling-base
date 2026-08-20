package synthesizer

import (
	"strings"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/recognizer"
)

func TestNewVolcengineLLMOptionDefaults(t *testing.T) {
	opt := NewVolcengineLLMOption("tok-abc", "app-xyz")
	if opt.Url != "wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async" {
		t.Errorf("Url = %q, want default bigmodel URL", opt.Url)
	}
	if opt.ResourceId != "volc.bigasr.sauc.duration" {
		t.Errorf("ResourceId = %q, want volc.bigasr.sauc.duration", opt.ResourceId)
	}
	if opt.AccessToken != "tok-abc" {
		t.Errorf("AccessToken = %q, want tok-abc", opt.AccessToken)
	}
	if opt.AppID != "app-xyz" {
		t.Errorf("AppID = %q, want app-xyz", opt.AppID)
	}
	if opt.Format != "pcm" {
		t.Errorf("Format = %q, want pcm", opt.Format)
	}
	if opt.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", opt.SampleRate)
	}
	if opt.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", opt.BitDepth)
	}
	if opt.Channel != 1 {
		t.Errorf("Channel = %d, want 1", opt.Channel)
	}
	if opt.Codec != "raw" {
		t.Errorf("Codec = %q, want raw", opt.Codec)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
	if opt.EndWindowSize != base.DefaultVolcEndWindowMs() {
		t.Errorf("EndWindowSize = %d, want %d", opt.EndWindowSize, base.DefaultVolcEndWindowMs())
	}
}

func TestVolcengineLLMOptionGetVendor(t *testing.T) {
	opt := NewVolcengineLLMOption("tok", "app")
	if got := opt.GetVendor(); got != base.VendorVolcengineLLM {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorVolcengineLLM)
	}
}

func TestVolcengineLLMVendor(t *testing.T) {
	v := NewVolcengineLLM(NewVolcengineLLMOption("tok", "app"))
	if got := v.Vendor(); got != "volcllmasr" {
		t.Errorf("Vendor() = %q, want volcllmasr", got)
	}
}

func TestVolcengineLLMInitStoresCallbacks(t *testing.T) {
	v := NewVolcengineLLM(NewVolcengineLLMOption("tok", "app"))
	var gotText string
	var gotErr error
	tr := func(text string, isLast bool, dur time.Duration, dialogID string) {
		gotText = text
	}
	er := func(err error, isFatal bool) {
		gotErr = err
	}
	v.Init(tr, er)
	if v.tr == nil {
		t.Fatal("tr callback not stored")
	}
	if v.er == nil {
		t.Fatal("er callback not stored")
	}
	v.tr("hi", true, 0, "dlg")
	if gotText != "hi" {
		t.Errorf("tr callback yielded %q, want hi", gotText)
	}
	v.er(base.ErrClientClosed, true)
	if gotErr != base.ErrClientClosed {
		t.Errorf("er callback yielded %v, want ErrClientClosed", gotErr)
	}
}

func TestGenerateCorpusContextWithHotwords(t *testing.T) {
	hotwords := []base.HotWord{
		{Word: "火山引擎", Weight: 5},
		{Word: "语音识别", Weight: 3},
	}
	ctx := GenerateCorpusContext(hotwords)
	if ctx == "" {
		t.Fatal("GenerateCorpusContext returned empty string")
	}
	if !strings.Contains(ctx, "火山引擎") {
		t.Errorf("context %q missing hotword 火山引擎", ctx)
	}
	if !strings.Contains(ctx, "语音识别") {
		t.Errorf("context %q missing hotword 语音识别", ctx)
	}
	if !strings.Contains(ctx, "hotwords") {
		t.Errorf("context %q missing hotwords key", ctx)
	}
}

func TestGenerateCorpusContextEmpty(t *testing.T) {
	ctx := GenerateCorpusContext(nil)
	if ctx == "" {
		t.Fatal("expected non-empty JSON for nil hotwords")
	}
	if !strings.Contains(ctx, "hotwords") {
		t.Errorf("context %q missing hotwords key", ctx)
	}
}
