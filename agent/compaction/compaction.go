// Package compaction manages conversation length to stay within the model's
// context window, mirroring the JS two-stage scheme (docs/compaction.md):
//
//   - Microcompact: fast, local, no model call. Elides old tool results when
//     they dominate the context.
//   - Autocompact: model-based summarization when nearing the window limit.
//
// Token counts here are estimates (the JS uses a real tokenizer); they are
// close enough to drive the same thresholds.
package compaction

import "github.com/LingByte/ling-base/relay"

// Microcompact constants (05-app-core.js).
const (
	KeepLastNResults         = 3
	ToolResultTokenThreshold = 40000
	MinTokensToSave          = 20000
	EstimatedTokensPerImage  = 2000
)

// Autocompact threshold offsets (05-app-core.js calculateTokenThresholds).
const (
	MaxReservedTokens   = 20000
	CompactBufferTokens = 13000
	BlockingLimitOffset = 3000
)

// DefaultContextWindow is the assumed window when the model's is unknown.
const DefaultContextWindow = 200000

// elidedPlaceholder replaces an old tool result's content during microcompact.
const elidedPlaceholder = "[Old tool result elided to save context]"

// EstimateTokens approximates the token count of a message list. Text is
// charged at ~4 chars/token; images/documents at a flat per-item estimate.
func EstimateTokens(messages []relay.RichMessage) int {
	total := 0
	for _, m := range messages {
		for _, b := range m.Content {
			total += blockTokens(b)
		}
	}
	return total
}

func blockTokens(b relay.ContentBlock) int {
	switch b.Type {
	case relay.BlockTypeText:
		return charTokens(b.Text)
	case relay.BlockTypeToolUse:
		// Rough: the serialized input.
		return charTokens(b.Name) + 50
	case relay.BlockTypeToolResult:
		return toolResultTokens(b)
	case relay.BlockTypeImage:
		return EstimatedTokensPerImage
	case relay.BlockTypeThinking:
		return charTokens(b.Thinking)
	default:
		return 0
	}
}

func toolResultTokens(tr relay.ContentBlock) int {
	n := 0
	// Content can be a string or an array of blocks.
	text := tr.GetToolResultText()
	n += charTokens(text)
	if text == "" && len(tr.Content) > 0 {
		// Likely image blocks in the content array.
		n += EstimatedTokensPerImage
	}
	return n
}

func charTokens(s string) int { return (len(s) + 3) / 4 }

// Result reports what a microcompact pass did.
type Result struct {
	Compacted   bool
	TokensSaved int
	ElidedCount int
}

// Microcompact elides the content of tool results older than the most recent
// KeepLastNResults, but only when tool-result tokens exceed the threshold and
// the savings clear MinTokensToSave. It never calls the model. The returned
// slice is a new slice; inputs are not mutated.
func Microcompact(messages []relay.RichMessage) ([]relay.RichMessage, Result) {
	// Locate every tool_result block as (msgIdx, blockIdx).
	type loc struct{ m, b int }
	var locs []loc
	totalToolTokens := 0
	for mi, m := range messages {
		for bi, blk := range m.Content {
			if blk.IsToolResult() {
				locs = append(locs, loc{mi, bi})
				totalToolTokens += toolResultTokens(blk)
			}
		}
	}

	if totalToolTokens <= ToolResultTokenThreshold || len(locs) <= KeepLastNResults {
		return messages, Result{}
	}

	// Candidates to elide: all but the last KeepLastNResults.
	elide := locs[:len(locs)-KeepLastNResults]

	// Compute potential savings first (only act if it clears the floor).
	saved := 0
	for _, l := range elide {
		tr := messages[l.m].Content[l.b]
		cur := toolResultTokens(tr)
		saved += cur - charTokens(elidedPlaceholder)
	}
	if saved < MinTokensToSave {
		return messages, Result{}
	}

	// Apply: deep-copy the affected messages and replace elided blocks.
	out := make([]relay.RichMessage, len(messages))
	copy(out, messages)
	touched := map[int]bool{}
	for _, l := range elide {
		if !touched[l.m] {
			nc := make([]relay.ContentBlock, len(messages[l.m].Content))
			copy(nc, messages[l.m].Content)
			out[l.m].Content = nc
			touched[l.m] = true
		}
		orig := out[l.m].Content[l.b]
		out[l.m].Content[l.b] = relay.NewToolResultBlock(orig.ToolUseID, elidedPlaceholder, orig.IsError)
	}

	return out, Result{Compacted: true, TokensSaved: saved, ElidedCount: len(elide)}
}

// Thresholds holds the computed autocompact thresholds for a context window.
type Thresholds struct {
	EffectiveWindow  int
	CompactThreshold int // autocompact triggers above this
	BlockingLimit    int // hard stop
}

// ComputeThresholds mirrors calculateTokenThresholds (05-app-core.js:117395).
func ComputeThresholds(contextWindow int) Thresholds {
	if contextWindow <= 0 {
		contextWindow = DefaultContextWindow
	}
	reserve := MaxReservedTokens
	if contextWindow < reserve {
		reserve = contextWindow / 2
	}
	eff := contextWindow - reserve
	return Thresholds{
		EffectiveWindow:  eff,
		CompactThreshold: eff - CompactBufferTokens,
		BlockingLimit:    eff - BlockingLimitOffset,
	}
}

// ShouldAutocompact reports whether the token count has crossed the compact
// threshold for the given context window.
func ShouldAutocompact(tokenCount, contextWindow int) bool {
	return tokenCount > ComputeThresholds(contextWindow).CompactThreshold
}
