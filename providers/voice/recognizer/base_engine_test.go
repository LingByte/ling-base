package recognizer

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBaseEngineInit(t *testing.T) {
	b := NewBaseEngine("test")
	var trCalled bool
	var erCalled bool
	b.Init(
		func(_ string, _ bool, _ time.Duration, _ string) { trCalled = true },
		func(_ error, _ bool) { erCalled = true },
	)

	b.EmitPartial("hello")
	if !trCalled {
		t.Error("result callback was not called by EmitPartial")
	}

	b.EmitError(errors.New("test error"), false)
	if !erCalled {
		t.Error("error callback was not called by EmitError")
	}
}

func TestBaseEngineVendor(t *testing.T) {
	b := NewBaseEngine("myvendor")
	if b.Vendor() != "myvendor" {
		t.Errorf("Vendor() = %q, want %q", b.Vendor(), "myvendor")
	}
}

func TestBaseEngineDialogID(t *testing.T) {
	b := NewBaseEngine("test")
	b.SetDialogID("dialog-123")
	if b.DialogID() != "dialog-123" {
		t.Errorf("DialogID() = %q, want %q", b.DialogID(), "dialog-123")
	}
}

func TestBaseEngineResetState(t *testing.T) {
	b := NewBaseEngine("test")
	b.SetDialogID("d1")

	// Emit partial to set sentence
	b.EmitPartial("hello")

	if !b.HasSentence() {
		t.Error("HasSentence should be true after EmitPartial")
	}

	b.ResetState()

	if b.HasSentence() {
		t.Error("HasSentence should be false after ResetState")
	}
}

func TestBaseEngineSinceSend(t *testing.T) {
	b := NewBaseEngine("test")

	// Before ResetState, SinceSend should return 0
	if d := b.SinceSend(); d != 0 {
		t.Errorf("SinceSend() before ResetState = %v, want 0", d)
	}

	b.ResetState()

	// After ResetState, SinceSend should be >= 0 (could be 0 if called within same nanosecond)
	d := b.SinceSend()
	if d < 0 {
		t.Errorf("SinceSend() after ResetState = %v, want >= 0", d)
	}

	time.Sleep(10 * time.Millisecond)
	d2 := b.SinceSend()
	if d2 <= d {
		t.Errorf("SinceSend() should increase over time: %v -> %v", d, d2)
	}
}

func TestBaseEngineEmitPartial(t *testing.T) {
	b := NewBaseEngine("test")
	b.SetDialogID("dialog-1")

	var results []struct {
		text     string
		isLast   bool
		dialogID string
	}
	var mu sync.Mutex

	b.Init(func(text string, isLast bool, _ time.Duration, dialogID string) {
		mu.Lock()
		defer mu.Unlock()
		results = append(results, struct {
			text     string
			isLast   bool
			dialogID string
		}{text, isLast, dialogID})
	}, nil)

	b.EmitPartial("hello")
	b.EmitPartial("hello world")

	mu.Lock()
	defer mu.Unlock()
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].text != "hello" {
		t.Errorf("result[0].text = %q, want %q", results[0].text, "hello")
	}
	if results[0].isLast {
		t.Error("result[0].isLast should be false for partial")
	}
	if results[1].text != "hello world" {
		t.Errorf("result[1].text = %q, want %q", results[1].text, "hello world")
	}
	if results[0].dialogID != "dialog-1" {
		t.Errorf("result[0].dialogID = %q, want %q", results[0].dialogID, "dialog-1")
	}
}

func TestBaseEngineEmitFinal(t *testing.T) {
	b := NewBaseEngine("test")
	b.SetDialogID("dialog-final")

	var lastText string
	var lastIsFinal bool
	var lastDialogID string

	b.Init(func(text string, isLast bool, _ time.Duration, dialogID string) {
		lastText = text
		lastIsFinal = isLast
		lastDialogID = dialogID
	}, nil)

	b.EmitFinal("final text")

	if lastText != "final text" {
		t.Errorf("text = %q, want %q", lastText, "final text")
	}
	if !lastIsFinal {
		t.Error("isLast should be true for EmitFinal")
	}
	if lastDialogID != "dialog-final" {
		t.Errorf("dialogID = %q, want %q", lastDialogID, "dialog-final")
	}

	// After EmitFinal, sentence should be cleared
	if b.HasSentence() {
		t.Error("HasSentence should be false after EmitFinal")
	}
}

func TestBaseEngineEmitFinalUsesPartialIfEmpty(t *testing.T) {
	b := NewBaseEngine("test")

	var lastText string
	b.Init(func(text string, _ bool, _ time.Duration, _ string) {
		lastText = text
	}, nil)

	b.EmitPartial("partial text")
	b.EmitFinal("") // empty text should use last partial

	if lastText != "partial text" {
		t.Errorf("text = %q, want %q (should use last partial)", lastText, "partial text")
	}
}

func TestBaseEngineEmitFinalEmptySkips(t *testing.T) {
	b := NewBaseEngine("test")

	var called bool
	b.Init(func(_ string, _ bool, _ time.Duration, _ string) {
		called = true
	}, nil)

	b.EmitFinal("")
	b.EmitFinal("   ")

	if called {
		t.Error("callback should not be called when both text and sentence are empty")
	}
}

func TestBaseEngineEmitPartialEmptySkips(t *testing.T) {
	b := NewBaseEngine("test")

	var called bool
	b.Init(func(_ string, _ bool, _ time.Duration, _ string) {
		called = true
	}, nil)

	b.EmitPartial("")
	b.EmitPartial("   ")

	if called {
		t.Error("callback should not be called for empty partial")
	}
}

func TestBaseEngineEmitError(t *testing.T) {
	b := NewBaseEngine("test")

	var lastErr error
	var lastIsFatal bool
	b.Init(nil, func(err error, isFatal bool) {
		lastErr = err
		lastIsFatal = isFatal
	})

	testErr := errors.New("test error")
	b.EmitError(testErr, true)

	if lastErr != testErr {
		t.Error("error mismatch")
	}
	if !lastIsFatal {
		t.Error("isFatal should be true")
	}
}

func TestBaseEngineEmitErrorNilCallback(t *testing.T) {
	b := NewBaseEngine("test")
	// Should not panic with nil callback
	b.EmitError(errors.New("test"), false)
}

func TestBaseEngineSentence(t *testing.T) {
	b := NewBaseEngine("test")

	if b.Sentence() != "" {
		t.Errorf("Sentence() = %q, want empty", b.Sentence())
	}

	b.EmitPartial("test sentence")

	if b.Sentence() != "test sentence" {
		t.Errorf("Sentence() = %q, want %q", b.Sentence(), "test sentence")
	}
}

func TestBaseEngineCallbacks(t *testing.T) {
	b := NewBaseEngine("test")
	var tr ResultFunc = func(_ string, _ bool, _ time.Duration, _ string) {}
	var er ErrorFunc = func(_ error, _ bool) {}

	b.Init(tr, er)

	gotTr, gotEr := b.Callbacks()
	if gotTr == nil {
		t.Error("Callbacks() tr should not be nil")
	}
	if gotEr == nil {
		t.Error("Callbacks() er should not be nil")
	}
}

func TestBaseEngineMarkEnd(t *testing.T) {
	b := NewBaseEngine("test")
	b.ResetState()
	b.MarkEnd()

	// Just verify it doesn't panic
	// The end time is stored internally
}

func TestBaseEngineConcurrentAccess(t *testing.T) {
	b := NewBaseEngine("test")
	b.Init(func(_ string, _ bool, _ time.Duration, _ string) {}, func(_ error, _ bool) {})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			b.SetDialogID("dialog")
			b.EmitPartial("partial")
			_ = b.DialogID()
			_ = b.Sentence()
			_ = b.HasSentence()
		}(i)
	}
	wg.Wait()
}

func TestBaseEngineEmitPartialTrimSpace(t *testing.T) {
	b := NewBaseEngine("test")

	var lastText string
	b.Init(func(text string, _ bool, _ time.Duration, _ string) {
		lastText = text
	}, nil)

	b.EmitPartial("  hello  ")

	if lastText != "hello" {
		t.Errorf("text = %q, want %q (should be trimmed)", lastText, "hello")
	}
	if !strings.Contains(b.Sentence(), "hello") {
		t.Errorf("Sentence should contain trimmed text, got %q", b.Sentence())
	}
}
