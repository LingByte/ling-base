package synthesizer

import (
	"testing"
	"time"

	base "github.com/LingByte/ling-base/voice/recognizer"
)

func TestNewLocalASROptionDefaults(t *testing.T) {
	opt := NewLocalASROption("whisper-cpp")
	if opt.Command != "whisper-cpp" {
		t.Errorf("Command = %q, want whisper-cpp", opt.Command)
	}
	if opt.Language != "en" {
		t.Errorf("Language = %q, want en", opt.Language)
	}
	if opt.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", opt.SampleRate)
	}
	if opt.Format != "wav" {
		t.Errorf("Format = %q, want wav", opt.Format)
	}
	if opt.ReqChanSize != 128 {
		t.Errorf("ReqChanSize = %d, want 128", opt.ReqChanSize)
	}
}

func TestLocalASROptionGetVendor(t *testing.T) {
	opt := NewLocalASROption("whisper-cpp")
	if got := opt.GetVendor(); got != base.VendorLocal {
		t.Errorf("GetVendor() = %q, want %q", got, base.VendorLocal)
	}
}

func TestLocalVendor(t *testing.T) {
	l := NewLocalASR(NewLocalASROption("whisper-cpp"))
	if got := l.Vendor(); got != "local" {
		t.Errorf("Vendor() = %q, want local", got)
	}
}

func TestLocalInitStoresCallbacks(t *testing.T) {
	l := NewLocalASR(NewLocalASROption("whisper-cpp"))
	var gotText string
	var gotErr error
	tr := func(text string, isLast bool, dur time.Duration, dialogID string) {
		gotText = text
	}
	er := func(err error, isFatal bool) {
		gotErr = err
	}
	l.Init(tr, er)
	if l.tr == nil {
		t.Fatal("tr callback not stored")
	}
	if l.er == nil {
		t.Fatal("er callback not stored")
	}
	l.tr("hello", true, 0, "dlg")
	if gotText != "hello" {
		t.Errorf("tr callback yielded %q, want hello", gotText)
	}
	l.er(base.ErrClientClosed, true)
	if gotErr != base.ErrClientClosed {
		t.Errorf("er callback yielded %v, want ErrClientClosed", gotErr)
	}
}

func TestDetectLocalASRCommandReturnsString(t *testing.T) {
	cmd := DetectLocalASRCommand()
	// Should return a string (possibly empty if nothing is installed); we only
	// assert the type/behavior contract here.
	_ = cmd // cmd is already a string; just ensure no panic.
	// If a known tool is on PATH it should be returned; otherwise empty is fine.
	known := []string{"whisper", "whisper-cpp", "vosk-transcriber", "deepspeech"}
	if cmd == "" {
		return
	}
	for _, k := range known {
		if cmd == k {
			return
		}
	}
	t.Errorf("DetectLocalASRCommand() = %q, not one of known tools %v", cmd, known)
}

func TestCheckLocalASRAvailableNonExistent(t *testing.T) {
	if CheckLocalASRAvailable("definitely-not-a-real-cmd-12345") {
		t.Error("CheckLocalASRAvailable returned true for non-existent command")
	}
	if CheckLocalASRAvailable("") {
		t.Error("CheckLocalASRAvailable returned true for empty command")
	}
}

func TestGetLocalASRInfoReturnsMap(t *testing.T) {
	info := GetLocalASRInfo()
	if info == nil {
		t.Fatal("GetLocalASRInfo returned nil map")
	}
	expected := []string{"whisper", "whisper-cpp", "vosk-transcriber", "deepspeech"}
	for _, tool := range expected {
		if _, ok := info[tool]; !ok {
			t.Errorf("GetLocalASRInfo missing entry for %q", tool)
		}
	}
}
