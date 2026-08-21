package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/ling-base/agent/agent"
)

// timelineEntry is one recorded event in the current session.
type timelineEntry struct {
	Time     time.Time
	Type     string
	ToolName string
	Summary  string
	IsError  bool
}

// timelineRecorder collects events for the /timeline command.
type timelineRecorder struct {
	entries []timelineEntry
}

func newTimelineRecorder() *timelineRecorder {
	return &timelineRecorder{}
}

func (t *timelineRecorder) record(ev agent.Event) {
	entry := timelineEntry{
		Time: time.Now(),
		Type: ev.Type,
	}
	switch ev.Type {
	case "assistant":
		entry.Summary = truncate(strings.TrimSpace(ev.Text), 80)
	case "tool_use":
		entry.ToolName = ev.ToolName
		entry.Summary = fmt.Sprintf("%v", ev.Input)
		entry.Summary = truncate(entry.Summary, 80)
	case "tool_result":
		entry.ToolName = ev.ToolName
		entry.Summary = truncate(strings.TrimSpace(ev.Content), 80)
		entry.IsError = ev.IsError
	case "compaction":
		entry.Summary = ev.Content
	case "usage":
		entry.Summary = fmt.Sprintf("in=%d out=%d turns=%d", ev.InputDelta, ev.OutputDelta, ev.TurnDelta)
	}
	t.entries = append(t.entries, entry)
}

// renderTimeline produces the text shown by /timeline.
func (t *timelineRecorder) render() string {
	if len(t.entries) == 0 {
		return "No events recorded yet."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Timeline (%d events):\n", len(t.entries)))
	for i, e := range t.entries {
		marker := "  "
		if e.IsError {
			marker = "✗ "
		}
		label := e.Type
		if e.ToolName != "" {
			label = e.ToolName
		}
		line := fmt.Sprintf("  %s[%s] %s — %s", marker, e.Time.Format("15:04:05"), label, e.Summary)
		b.WriteString(strings.TrimRight(line, " —") + "\n")
		_ = i
	}
	return strings.TrimSpace(b.String())
}
