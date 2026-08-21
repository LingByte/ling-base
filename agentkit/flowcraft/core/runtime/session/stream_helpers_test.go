package session

import (
	"github.com/LingByte/ling-base/agentkit/flowcraft/core/agent"
	"github.com/LingByte/ling-base/agentkit/flowcraft/core/message"
)

// tokenText extracts the text of a Part-model stream delta for test
// assertions.
func tokenText(delta agent.StreamDeltaPayload) string {
	if text, ok := delta.Part.(message.TextPart); ok {
		return text.Text
	}
	return ""
}
