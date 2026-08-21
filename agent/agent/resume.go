package agent

import (
	"encoding/json"
	"fmt"

	"github.com/LingByte/ling-base/agent/session"
)

// MessagesFromEntries reconstructs the conversation as message params from
// transcript entries, for resuming a session. Each entry's stored message JSON
// is parsed into a Message (extra envelope fields like id/usage on assistant
// messages are ignored).
func MessagesFromEntries(entries []session.Entry) ([]Message, error) {
	msgs := make([]Message, 0, len(entries))
	for i, e := range entries {
		var m Message
		if err := json.Unmarshal(e.Message, &m); err != nil {
			return nil, fmt.Errorf("entry %d (%s %s): %w", i, e.Type, e.UUID, err)
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}
