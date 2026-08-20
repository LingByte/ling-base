package synthesizer

import (
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/recognizer"
)

func TestNewFunASROptionDefaults(t *testing.T) {
	opt := NewFunASROption("wss://funasr/ws")
	if opt.URL != "wss://funasr/ws" {
		t.Errorf("URL = %q, want wss://funasr/ws", opt.URL)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
	if opt.Model != "paraformer-realtime-v2" {
		t.Errorf("Model = %q, want paraformer-realtime-v2", opt.Model)
	}
	if opt.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", opt.SampleRate)
	}
	if opt.Format != "pcm" {
		t.Errorf("Format = %q, want pcm", opt.Format)
	}
	if opt.Mode != "2pass" {
		t.Errorf("Mode = %q, want 2pass", opt.Mode)
	}
	if len(opt.ChunkSize) != 3 || opt.ChunkSize[0] != 5 || opt.ChunkSize[1] != 10 || opt.ChunkSize[2] != 5 {
		t.Errorf("ChunkSize = %v, want [5 10 5]", opt.ChunkSize)
	}
	if opt.ChunkInterval != 10 {
		t.Errorf("ChunkInterval = %d, want 10", opt.ChunkInterval)
	}
	if opt.EncoderChunkLookBack != 4 {
		t.Errorf("EncoderChunkLookBack = %d, want 4", opt.EncoderChunkLookBack)
	}
	if opt.DecoderChunkLookBack != 0 {
		t.Errorf("DecoderChunkLookBack = %d, want 0", opt.DecoderChunkLookBack)
	}
	if opt.AudioFs != 16000 {
		t.Errorf("AudioFs = %d, want 16000", opt.AudioFs)
	}
	if opt.WavName != "demo" {
		t.Errorf("WavName = %q, want demo", opt.WavName)
	}
	if opt.WavFormat != "pcm" {
		t.Errorf("WavFormat = %q, want pcm", opt.WavFormat)
	}
	if !opt.IsSpeaking {
		t.Errorf("IsSpeaking = %v, want true", opt.IsSpeaking)
	}
}

func TestFunASROptionGetVendor(t *testing.T) {
	opt := NewFunASROption("wss://x")
	if got := opt.GetVendor(); got != base.VendorFunASR {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorFunASR)
	}
}

func TestFunASRVendor(t *testing.T) {
	fun := NewFunASR(NewFunASROption("wss://x"))
	if got := fun.Vendor(); got != "funasr" {
		t.Errorf("Vendor() = %q, want funasr", got)
	}
}

func TestFunASRInitStoresCallbacks(t *testing.T) {
	fun := NewFunASR(NewFunASROption("wss://x"))
	var gotText string
	var gotErr error
	tr := func(text string, isLast bool, dur time.Duration, dialogID string) {
		gotText = text
	}
	er := func(err error, isFatal bool) {
		gotErr = err
	}
	fun.Init(tr, er)
	if fun.tr == nil {
		t.Fatal("tr callback not stored")
	}
	if fun.er == nil {
		t.Fatal("er callback not stored")
	}
	fun.tr("hello", true, 0, "dlg")
	if gotText != "hello" {
		t.Errorf("tr callback yielded %q, want hello", gotText)
	}
	fun.er(base.ErrClientClosed, true)
	if gotErr != base.ErrClientClosed {
		t.Errorf("er callback yielded %v, want ErrClientClosed", gotErr)
	}
}
