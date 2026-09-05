// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package alerting_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/alerting"
)

func TestEngine_FailedTransition(t *testing.T) {
	var mu sync.Mutex
	var events []alerting.Event
	a := alerting.New(alerting.WithHandlerFunc(func(ctx context.Context, e alerting.Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	}))

	// First failure → should emit "failed".
	ev, emitted := a.Record(context.Background(), "api", false, "timeout")
	if !emitted {
		t.Error("First failure should emit")
	}
	if ev.Transition != alerting.TransitionFailed {
		t.Errorf("Transition = %s, want failed", ev.Transition)
	}
	if ev.ToState != alerting.StateFailed {
		t.Errorf("ToState = %s, want failed", ev.ToState)
	}

	// Second failure → should NOT emit (still failed).
	_, emitted = a.Record(context.Background(), "api", false, "timeout")
	if emitted {
		t.Error("Second failure should not emit")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 1 {
		t.Errorf("Events count = %d, want 1", len(events))
	}
}

func TestEngine_RecoveryTransition(t *testing.T) {
	var events []alerting.Event
	var mu sync.Mutex
	a := alerting.New(alerting.WithHandlerFunc(func(ctx context.Context, e alerting.Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	}))

	a.Record(context.Background(), "db", false, "conn refused")
	a.Record(context.Background(), "db", true, "")

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 {
		t.Fatalf("Events count = %d, want 2", len(events))
	}
	if events[1].Transition != alerting.TransitionRecovered {
		t.Errorf("Second event transition = %s, want recovered", events[1].Transition)
	}
	if events[1].ToState != alerting.StateHealthy {
		t.Errorf("ToState = %s, want healthy", events[1].ToState)
	}
}

func TestEngine_NoEmitOnContinuousFailure(t *testing.T) {
	count := 0
	var mu sync.Mutex
	a := alerting.New(alerting.WithHandlerFunc(func(ctx context.Context, e alerting.Event) {
		mu.Lock()
		defer mu.Unlock()
		count++
	}))

	for i := 0; i < 10; i++ {
		a.Record(context.Background(), "svc", false, "err")
	}

	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Errorf("Handler called %d times, want 1", count)
	}
}

func TestEngine_FailThreshold(t *testing.T) {
	var events []alerting.Event
	var mu sync.Mutex
	a := alerting.New(
		alerting.WithFailThreshold(3),
		alerting.WithHandlerFunc(func(ctx context.Context, e alerting.Event) {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, e)
		}),
	)

	// First 2 failures should not emit.
	a.Record(context.Background(), "svc", false, "err1")
	a.Record(context.Background(), "svc", false, "err2")
	mu.Lock()
	if len(events) != 0 {
		t.Errorf("Events after 2 failures = %d, want 0", len(events))
	}
	mu.Unlock()

	// Third failure should emit.
	a.Record(context.Background(), "svc", false, "err3")
	mu.Lock()
	defer mu.Unlock()
	if len(events) != 1 {
		t.Errorf("Events after 3 failures = %d, want 1", len(events))
	}
	if events[0].Transition != alerting.TransitionFailed {
		t.Errorf("Transition = %s, want failed", events[0].Transition)
	}
	if events[0].FailureCount != 3 {
		t.Errorf("FailureCount = %d, want 3", events[0].FailureCount)
	}
}

func TestEngine_RepeatInterval(t *testing.T) {
	var events []alerting.Event
	var mu sync.Mutex
	a := alerting.New(
		alerting.WithRepeatInterval(50*time.Millisecond),
		alerting.WithHandlerFunc(func(ctx context.Context, e alerting.Event) {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, e)
		}),
	)

	a.Record(context.Background(), "svc", false, "err") // emit failed
	a.Record(context.Background(), "svc", false, "err") // no emit (within interval)

	time.Sleep(60 * time.Millisecond)

	a.Record(context.Background(), "svc", false, "err") // emit still_failed

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 {
		t.Fatalf("Events count = %d, want 2", len(events))
	}
	if events[1].Transition != alerting.TransitionStillFailed {
		t.Errorf("Second transition = %s, want still_failed", events[1].Transition)
	}
}

func TestEngine_EmitOnNoChange(t *testing.T) {
	var count int
	var mu sync.Mutex
	a := alerting.New(
		alerting.WithEmitOnNoChange(true),
		alerting.WithHandlerFunc(func(ctx context.Context, e alerting.Event) {
			mu.Lock()
			defer mu.Unlock()
			count++
		}),
	)

	a.Record(context.Background(), "svc", true, "")
	a.Record(context.Background(), "svc", true, "")
	a.Record(context.Background(), "svc", true, "")

	mu.Lock()
	defer mu.Unlock()
	// First is healthy→healthy (no prior), subsequent are still_healthy.
	// With emitOnNoChange, every record after the first should emit.
	if count < 2 {
		t.Errorf("Handler called %d times, want >= 2", count)
	}
}

func TestEngine_State(t *testing.T) {
	a := alerting.New()

	if a.State("unknown-check") != alerting.StateUnknown {
		t.Error("Unknown check should return StateUnknown")
	}

	a.Record(context.Background(), "check", false, "err")
	if a.State("check") != alerting.StateFailed {
		t.Errorf("State = %s, want failed", a.State("check"))
	}

	a.Record(context.Background(), "check", true, "")
	if a.State("check") != alerting.StateHealthy {
		t.Errorf("State = %s, want healthy", a.State("check"))
	}
}

func TestEngine_FailureCount(t *testing.T) {
	a := alerting.New()

	a.Record(context.Background(), "check", false, "e1")
	a.Record(context.Background(), "check", false, "e2")
	a.Record(context.Background(), "check", false, "e3")

	if a.FailureCount("check") != 3 {
		t.Errorf("FailureCount = %d, want 3", a.FailureCount("check"))
	}

	a.Record(context.Background(), "check", true, "")
	if a.FailureCount("check") != 0 {
		t.Errorf("FailureCount after recovery = %d, want 0", a.FailureCount("check"))
	}
}

func TestEngine_Checks(t *testing.T) {
	a := alerting.New()
	a.Record(context.Background(), "a", true, "")
	a.Record(context.Background(), "b", false, "")
	a.Record(context.Background(), "c", true, "")

	checks := a.Checks()
	if len(checks) != 3 {
		t.Errorf("Checks count = %d, want 3", len(checks))
	}
}

func TestEngine_Remove(t *testing.T) {
	a := alerting.New()
	a.Record(context.Background(), "check", false, "err")
	a.Remove("check")

	if a.State("check") != alerting.StateUnknown {
		t.Errorf("State after remove = %s, want unknown", a.State("check"))
	}
}

func TestEngine_Reset(t *testing.T) {
	a := alerting.New()
	a.Record(context.Background(), "a", false, "")
	a.Record(context.Background(), "b", false, "")
	a.Reset()

	if len(a.Checks()) != 0 {
		t.Errorf("Checks after reset = %d, want 0", len(a.Checks()))
	}
}

func TestEngine_Snapshot(t *testing.T) {
	a := alerting.New()
	a.Record(context.Background(), "a", true, "")
	a.Record(context.Background(), "b", false, "")

	snap := a.Snapshot()
	if snap["a"] != alerting.StateHealthy {
		t.Errorf("Snapshot[a] = %s, want healthy", snap["a"])
	}
	if snap["b"] != alerting.StateFailed {
		t.Errorf("Snapshot[b] = %s, want failed", snap["b"])
	}
}

func TestEngine_Metadata(t *testing.T) {
	var events []alerting.Event
	var mu sync.Mutex
	a := alerting.New(alerting.WithHandlerFunc(func(ctx context.Context, e alerting.Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	}))

	a.RecordWithMetadata(context.Background(), "check", false, "err", map[string]string{
		"host": "node-1",
		"env":  "prod",
	})

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 1 {
		t.Fatalf("Events = %d, want 1", len(events))
	}
	if events[0].Metadata["host"] != "node-1" {
		t.Errorf("Metadata[host] = %q", events[0].Metadata["host"])
	}
}

// ──────────────────────────────────────────────
// Notifier integration
// ──────────────────────────────────────────────

type mockNotifier struct {
	mu      sync.Mutex
	calls   int
	lastErr error
}

func (m *mockNotifier) Notify(ctx context.Context, e alerting.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	return m.lastErr
}

func TestEngine_WithNotifier(t *testing.T) {
	n := &mockNotifier{}
	a := alerting.New(alerting.WithNotifier(n))

	a.Record(context.Background(), "check", false, "err")
	a.Record(context.Background(), "check", false, "err") // no emit
	a.Record(context.Background(), "check", true, "")     // emit recovered

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.calls != 2 {
		t.Errorf("Notifier calls = %d, want 2", n.calls)
	}
}

// ──────────────────────────────────────────────
// Formatting
// ──────────────────────────────────────────────

func TestFormatEvent(t *testing.T) {
	e := alerting.Event{
		CheckName:    "api",
		Transition:   alerting.TransitionFailed,
		FromState:    alerting.StateHealthy,
		ToState:      alerting.StateFailed,
		FailureCount: 3,
		Message:      "timeout",
	}
	s := alerting.FormatEvent(e)
	if s == "" {
		t.Error("FormatEvent returned empty string")
	}
}
