package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/LingByte/ling-base/agent/permission"
	"github.com/LingByte/ling-base/agent/tools"
	"github.com/LingByte/ling-base/relay"
)

// captureRecorder collects every Record call in order — to assert that the
// transcript only sees the assistant turn AFTER its tool_result is also ready
// (so a process-death window during dispatch can't leave an orphan).
type captureRecorder struct {
	mu   sync.Mutex
	rows []string
}

func (r *captureRecorder) Record(role string, msg json.RawMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, role)
	return nil
}

func (r *captureRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.rows...)
}

// scriptedProvider returns each preset message in turn, then synthesises an
// empty end_turn so the loop exits cleanly. Implements Provider.
type scriptedProvider struct {
	turns []Response
	n     int
}

func (p *scriptedProvider) StreamTurn(_ context.Context, _ *relay.RichChatRequest, _ StreamSink) (*Response, error) {
	if p.n >= len(p.turns) {
		return &Response{StopReason: "end_turn"}, nil
	}
	m := p.turns[p.n]
	p.n++
	return &m, nil
}

// markerTool stamps the captureRecorder during Execute so the test can prove
// where dispatch lands relative to the assistant/user records.
type markerTool struct{ rec *captureRecorder }

func (markerTool) Name() string                                { return "Marker" }
func (markerTool) Description(context.Context) (string, error) { return "", nil }
func (markerTool) InputSchema() json.RawMessage                { return json.RawMessage(`{"type":"object"}`) }
func (markerTool) ValidateInput(json.RawMessage) error         { return nil }
func (markerTool) PermissionRequest(json.RawMessage) permission.PermissionRequest {
	return permission.PermissionRequest{}
}
func (markerTool) CheckPermissions(_ permission.Context, _ permission.PermissionRequest) permission.Decision {
	return permission.Decision{Behavior: permission.Allow}
}
func (m markerTool) Execute(context.Context, tools.Context, json.RawMessage) ([]tools.Result, error) {
	_ = m.rec.Record("DISPATCH", nil)
	return []tools.Result{{Content: "ok"}}, nil
}

// TestAssistantToolUseRecordedAfterDispatch verifies the recording order fix.
func TestAssistantToolUseRecordedAfterDispatch(t *testing.T) {
	asstJSON := `{
		"role": "assistant",
		"stop_reason": "tool_use",
		"content": [{"type": "tool_use", "id": "tu_1", "name": "Marker", "input": {}}]
	}`
	var asst Response
	if err := json.Unmarshal([]byte(asstJSON), &asst); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	rec := &captureRecorder{}
	provider := &scriptedProvider{turns: []Response{asst}}
	loop := New(provider, tools.NewRegistry(markerTool{rec: rec}))

	if _, err := loop.Run(context.Background(), Options{Recorder: rec, MaxTurns: 1}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	rows := rec.snapshot()
	if len(rows) < 3 {
		t.Fatalf("want at least DISPATCH, assistant, user; got %v", rows)
	}
	var dispatchIdx, assistantIdx int = -1, -1
	for i, r := range rows {
		if r == "DISPATCH" && dispatchIdx < 0 {
			dispatchIdx = i
		}
		if r == "assistant" && assistantIdx < 0 {
			assistantIdx = i
		}
	}
	if dispatchIdx < 0 || assistantIdx < 0 || dispatchIdx >= assistantIdx {
		t.Errorf("dispatch must record before assistant; order = %v (dispatch=%d, assistant=%d)",
			rows, dispatchIdx, assistantIdx)
	}
	if assistantIdx+1 >= len(rows) || rows[assistantIdx+1] != "user" {
		t.Errorf("assistant must be paired with user back-to-back; got rows=%v", rows)
	}
	if !strings.Contains(strings.Join(rows, ","), "DISPATCH,assistant,user") {
		t.Errorf("expected DISPATCH,assistant,user contiguous; got %v", rows)
	}
}
