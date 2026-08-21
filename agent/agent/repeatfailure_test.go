package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/LingByte/ling-base/agent/tools"
	"github.com/LingByte/ling-base/relay"
)

func TestRepeatedFailureMsg(t *testing.T) {
	edit := repeatedFailureMsg("Edit", 3)
	if !strings.Contains(edit, "already tried this exact Edit call 3 times") {
		t.Errorf("Edit message missing the count: %q", edit)
	}
	if !strings.Contains(edit, "Use Read") {
		t.Errorf("Edit message should steer to Read: %q", edit)
	}
	// A tool without specific guidance still gets the generic stop-repeating push.
	if got := repeatedFailureMsg("Bash", 2); !strings.Contains(got, "Stop repeating it") {
		t.Errorf("generic message = %q", got)
	}
}

// TestDispatchBreaksRetryLoop drives dispatch with the same failing call
// repeatedly: once the per-Run failure count reaches the limit, dispatch must
// stop and return the directive steering instead of the original error.
func TestDispatchBreaksRetryLoop(t *testing.T) {
	read, _ := tools.NewRead()
	l := New(nil, tools.NewRegistry(read))

	// An unknown tool always fails (errResult), which increments the counter.
	inputJSON, _ := json.Marshal(map[string]any{"a": 1})
	tu := ContentBlock{Type: BlockToolUse, ID: "t1", Name: "Frobnicate", Input: relay.ContentBlock{}.Input}
	tu.Input = inputJSON
	failures := map[string]int{}
	streaks := map[string]errStreak{}
	reveal := func(...string) {}

	textOf := func(b ContentBlock) string {
		return b.GetToolResultText()
	}

	// First repeatFailureLimit attempts return the normal error and bump the count.
	for i := 0; i < repeatFailureLimit; i++ {
		got := textOf(l.dispatch(context.Background(), tu, Options{}, nil, reveal, failures, streaks))
		if !strings.Contains(got, "No such tool available") {
			t.Fatalf("attempt %d: expected the normal error, got %q", i+1, got)
		}
	}

	// The next identical attempt is short-circuited with the steering message.
	got := textOf(l.dispatch(context.Background(), tu, Options{}, nil, reveal, failures, streaks))
	if !strings.Contains(got, "already tried this exact") {
		t.Fatalf("expected loop-breaker steering, got %q", got)
	}
}

// TestDispatchBreaksSameShapeErrorLoop covers loop-breaker B: same tool,
// DIFFERENT args each call, but the same error shape every time.
func TestDispatchBreaksSameShapeErrorLoopVariedInputsGetsEnvMessage(t *testing.T) {
	read, _ := tools.NewRead()
	l := New(nil, tools.NewRegistry(read))

	failures := map[string]int{}
	streaks := map[string]errStreak{}
	textOf := func(b ContentBlock) string {
		return b.GetToolResultText()
	}

	// Three calls to "Find" with DIFFERENT args each time.
	for i, args := range []map[string]any{{"q": "alpha"}, {"q": "beta"}, {"q": "gamma"}} {
		inputJSON, _ := json.Marshal(args)
		tu := ContentBlock{Type: BlockToolUse, ID: "t", Name: "Find", Input: inputJSON}
		got := textOf(l.dispatch(context.Background(), tu, Options{}, nil, func(...string) {}, failures, streaks))
		switch i {
		case 0, 1:
			if !strings.Contains(got, "No such tool available: Find") {
				t.Fatalf("call %d: expected the standard error, got %q", i+1, got)
			}
		case 2:
			if !strings.Contains(got, "across DIFFERENT inputs") {
				t.Fatalf("call %d: expected env-flavored message, got %q", i+1, got)
			}
			if !strings.Contains(got, "environment issue") || !strings.Contains(got, "recurring error") {
				t.Fatalf("call %d: expected env-flavored message, got %q", i+1, got)
			}
		}
	}
}
