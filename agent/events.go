// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

// Event is the interface for all events emitted by the agent loop.
// Consumers (print mode, json mode, future TUI) receive these via
// the sink callback passed to Agent.Prompt.
type Event interface {
	eventMarker()
}

// EvUserMessage is emitted when a user message is added to the transcript.
type EvUserMessage struct {
	Text string
}

func (EvUserMessage) eventMarker() {}

// EvTurnStart is emitted at the beginning of each agent loop step.
type EvTurnStart struct {
	Step int
}

func (EvTurnStart) eventMarker() {}

// EvTurnEnd is emitted at the end of each step, with the stop reason.
type EvTurnEnd struct {
	StopReason string // "stop", "tool_calls", "length", etc.
}

func (EvTurnEnd) eventMarker() {}

// EvAssistantText is emitted when the assistant produces text.
// In non-streaming mode, the full text is delivered in one event.
type EvAssistantText struct {
	Text string
}

func (EvAssistantText) eventMarker() {}

// EvToolCallStart is emitted before a tool is executed.
type EvToolCallStart struct {
	Name      string
	Arguments string
}

func (EvToolCallStart) eventMarker() {}

// EvToolCallEnd is emitted after a tool finishes.
type EvToolCallEnd struct {
	Name    string
	Result  ToolResult
	Err     error
}

func (EvToolCallEnd) eventMarker() {}

// EvUsage is emitted after each LLM call with token usage info.
type EvUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

func (EvUsage) eventMarker() {}

// EvDone is emitted when the agent loop completes.
type EvDone struct{}

func (EvDone) eventMarker() {}

// EvError is emitted when an error occurs in the loop.
type EvError struct {
	Err error
}

func (EvError) eventMarker() {}
