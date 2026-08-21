package compaction

import "github.com/LingByte/ling-base/relay"

// SummaryInstruction is appended to the conversation to elicit a structured
// summary, condensed from the JS compaction prompt (compactConversation).
const SummaryInstruction = `Your context window is nearly full. Produce a detailed summary of the conversation so far that captures everything needed to continue the work seamlessly. Include:
1. The user's overall goal and any explicit requirements or constraints.
2. Key files, functions, and decisions made, with paths.
3. What has been done so far and the current state.
4. The next steps that remain.
Write the summary as plain prose. Do not ask questions or take any further action.`

// BuildSummaryRequest returns a rich chat request that asks the model to
// summarize the given conversation. The summary instruction is appended as a
// final user turn. model is the model id string (e.g. "claude-sonnet-4-5-20250929").
func BuildSummaryRequest(messages []relay.RichMessage, model string, maxTokens int64) *relay.RichChatRequest {
	msgs := append([]relay.RichMessage{}, messages...)
	msgs = append(msgs, relay.NewUserMessage(SummaryInstruction))
	return &relay.RichChatRequest{
		Model:     model,
		MaxTokens: int(maxTokens),
		Messages:  msgs,
	}
}

// ReplaceWithSummary returns the post-compaction conversation: a single user
// message carrying the summary, mirroring how the JS replaces history with a
// compact-boundary + summary (here condensed to one carried-forward message).
func ReplaceWithSummary(summary string) []relay.RichMessage {
	const preamble = "[Conversation compacted to save context. Summary of the prior conversation follows.]\n\n"
	return []relay.RichMessage{
		relay.NewUserMessage(preamble + summary),
	}
}
