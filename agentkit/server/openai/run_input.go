//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package openai

import (
	"context"
	"errors"
	"fmt"

	"github.com/LingByte/ling-base/agentkit/agent"
	compat "github.com/LingByte/ling-base/relay/compat"
)

const (
	errToolMessageMissingID = "tool message missing tool call id"
	errToolMessageNotString = "tool message content must be a string"
)

// runInputMessages describes how an OpenAI chat completion request should be
// passed to runner.Run. It mirrors the AG-UI runner's input message splitting:
// prior conversation history, the current-turn input message, and any trailing
// tool results submitted in the same request.
type runInputMessages struct {
	inputMessage compat.Message
	history      []compat.Message
	toolMessages []compat.Message
}

// runInputFromMessages splits converted framework messages into history and the
// current run input. When the request ends with one or more role=tool messages,
// those tool results are treated as the current turn input, matching AG-UI's
// external tool resume semantics. All other roles keep the legacy behavior of
// passing the last message through unchanged.
func runInputFromMessages(messages []compat.Message) (*runInputMessages, error) {
	if len(messages) == 0 {
		return nil, errors.New("messages cannot be empty")
	}
	lastMessage := messages[len(messages)-1]
	if lastMessage.Role == compat.RoleTool {
		return toolResultRunInputFromMessages(messages)
	}
	history := []compat.Message(nil)
	if len(messages) > 1 {
		history = messages[:len(messages)-1]
	}
	return &runInputMessages{
		inputMessage: lastMessage,
		history:      history,
	}, nil
}

func toolResultRunInputFromMessages(messages []compat.Message) (*runInputMessages, error) {
	start := len(messages) - 1
	for start >= 0 && messages[start].Role == compat.RoleTool {
		start--
	}
	toolMessages := make([]compat.Message, 0, len(messages)-start-1)
	for _, msg := range messages[start+1:] {
		if msg.ToolID == "" {
			return nil, errors.New(errToolMessageMissingID)
		}
		if len(msg.ContentParts) > 0 {
			return nil, fmt.Errorf("%s: multimodal tool results are not supported", errToolMessageNotString)
		}
		toolMessages = append(toolMessages, msg)
	}
	history := []compat.Message(nil)
	if start >= 0 {
		history = append(history, messages[:start+1]...)
	}
	return &runInputMessages{
		inputMessage: toolMessages[len(toolMessages)-1],
		history:      history,
		toolMessages: toolMessages,
	}, nil
}

func withToolResultMessageRewriter(toolMessages []compat.Message) agent.RunOption {
	currentTurnMessages := append([]compat.Message(nil), toolMessages...)
	return func(opts *agent.RunOptions) {
		if len(toolMessages) <= 1 {
			return
		}
		userMessageRewriter := opts.UserMessageRewriter
		if userMessageRewriter == nil {
			opts.UserMessageRewriter = func(
				context.Context,
				*agent.UserMessageRewriteArgs,
			) ([]compat.Message, error) {
				return append([]compat.Message(nil), currentTurnMessages...), nil
			}
			return
		}
		opts.UserMessageRewriter = func(
			ctx context.Context,
			args *agent.UserMessageRewriteArgs,
		) ([]compat.Message, error) {
			rewritten, err := userMessageRewriter(ctx, args)
			if err != nil {
				return nil, err
			}
			return mergeToolResultRewriteMessages(rewritten, currentTurnMessages), nil
		}
	}
}

func mergeToolResultRewriteMessages(
	rewritten []compat.Message,
	toolResults []compat.Message,
) []compat.Message {
	toolResultIDs := make(map[string]struct{}, len(toolResults))
	rewrittenToolResults := make(map[string]compat.Message, len(toolResults))
	for _, msg := range toolResults {
		if msg.ToolID != "" {
			toolResultIDs[msg.ToolID] = struct{}{}
		}
	}
	merged := make([]compat.Message, 0, len(rewritten)+len(toolResults))
	for _, msg := range rewritten {
		if msg.Role == compat.RoleTool && msg.ToolID != "" {
			if _, ok := toolResultIDs[msg.ToolID]; ok {
				rewrittenToolResults[msg.ToolID] = msg
				continue
			}
		}
		merged = append(merged, msg)
	}
	for _, msg := range toolResults {
		if rewrittenMsg, ok := rewrittenToolResults[msg.ToolID]; ok {
			rewrittenMsg.Role = compat.RoleTool
			rewrittenMsg.ToolID = msg.ToolID
			if msg.ToolName != "" {
				rewrittenMsg.ToolName = msg.ToolName
			}
			merged = append(merged, rewrittenMsg)
			continue
		}
		merged = append(merged, msg)
	}
	return merged
}

// buildRunOptions assembles the RunOptions shared by streaming and
// non-streaming paths: prior conversation history, optional parallel tool
// result rewriting, and any caller-declared external tools from req.Tools.
func buildRunOptions(
	req *openAIRequest,
	input *runInputMessages,
) ([]agent.RunOption, error) {
	runOpts := []agent.RunOption{}
	if len(input.history) > 0 {
		runOpts = append(runOpts, agent.WithMessages(input.history))
	}
	if len(input.toolMessages) > 1 {
		runOpts = append(runOpts, withToolResultMessageRewriter(input.toolMessages))
	}
	return appendExternalToolRunOption(runOpts, req)
}
