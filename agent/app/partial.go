package app

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/LingByte/ling-base/agent/agent"
	"github.com/google/uuid"
)

// partialEmitter writes raw model stream chunks as JS-compatible stream_event
// lines: {type:"stream_event", event:<raw chunk>, session_id,
// parent_tool_use_id, uuid}. It is wired to agent.Options.PartialMessages when
// --include-partial-messages is set. Writes are mutex-guarded so partials
// interleave safely with the envelope recorder writing to the same stream.
type partialEmitter struct {
	w         io.Writer
	sessionID string
	mu        *sync.Mutex
}

func newPartialEmitter(w io.Writer, sessionID string, mu *sync.Mutex) *partialEmitter {
	return &partialEmitter{w: w, sessionID: sessionID, mu: mu}
}

// emit serializes one raw stream chunk. The chunk is forwarded as a
// stream_event line, structurally similar to the JS reference's partial output.
func (e *partialEmitter) emit(ev agent.StreamChunk) {
	line, err := json.Marshal(map[string]any{
		"type":               "stream_event",
		"event":              ev,
		"session_id":         e.sessionID,
		"parent_tool_use_id": nil,
		"uuid":               uuid.NewString(),
	})
	if err != nil {
		return
	}
	if e.mu != nil {
		e.mu.Lock()
		defer e.mu.Unlock()
	}
	_, _ = e.w.Write(append(line, '\n'))
}
