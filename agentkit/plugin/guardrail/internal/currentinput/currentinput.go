//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

// Package currentinput provides shared helpers for extracting the latest user input and transcript.
package currentinput

import (
	"context"
	"strings"

	compat "github.com/LingByte/ling-base/relay/compat"
	guardtranscript "github.com/LingByte/ling-base/agentkit/plugin/guardrail/internal/transcript"
)

// Request contains the latest user input plus supporting transcript evidence.
type Request[T any] struct {
	LastUserInput string
	Transcript    []T
}

// Build extracts the latest user input and supporting transcript for content reviewers.
func Build[T any](
	ctx context.Context,
	messages []compat.Message,
	tokenCounter compat.TokenCounter,
	mapEntry func(guardtranscript.Entry) T,
) *Request[T] {
	lastUserInput, lastUserIndex := extractLastUserInput(messages)
	if lastUserInput == "" {
		return nil
	}
	transcript := buildTranscript(ctx, collectTranscriptEntries(messages, lastUserIndex), tokenCounter, mapEntry)
	return &Request[T]{
		LastUserInput: lastUserInput,
		Transcript:    transcript,
	}
}

func buildTranscript[T any](
	ctx context.Context,
	rawEntries []guardtranscript.Record,
	tokenCounter compat.TokenCounter,
	mapEntry func(guardtranscript.Entry) T,
) []T {
	if len(rawEntries) == 0 {
		return nil
	}
	entries := guardtranscript.Build(
		ctx,
		rawEntries,
		func(ctx context.Context, entry guardtranscript.Entry) int {
			return countTranscriptTokens(ctx, tokenCounter, entry)
		},
		guardtranscript.DefaultOptions(),
	)
	if len(entries) == 0 {
		return nil
	}
	transcript := make([]T, 0, len(entries))
	for _, entry := range entries {
		transcript = append(transcript, mapEntry(entry))
	}
	return transcript
}

func countTranscriptTokens(
	ctx context.Context,
	tokenCounter compat.TokenCounter,
	entry guardtranscript.Entry,
) int {
	if tokenCounter == nil {
		return guardtranscript.DefaultMessageTranscriptBudget + 1
	}
	tokens, err := tokenCounter.CountTokens(ctx, compat.Message{
		Role:    entry.Role,
		Content: entry.Content,
	})
	if err != nil || tokens < 0 {
		return guardtranscript.DefaultMessageTranscriptBudget + 1
	}
	return tokens
}

func collectTranscriptEntries(messages []compat.Message, excludedUserIndex int) []guardtranscript.Record {
	entries := make([]guardtranscript.Record, 0, len(messages))
	nextIndex := 0
	for i := range messages {
		message := messages[i]
		if message.Role == compat.RoleSystem || i == excludedUserIndex {
			continue
		}
		text := extractMessageText(message)
		if text == "" {
			continue
		}
		category := guardtranscript.CategoryMessage
		if message.Role == compat.RoleTool {
			category = guardtranscript.CategoryTool
		}
		entries = append(entries, guardtranscript.Record{
			Index: nextIndex,
			Entry: guardtranscript.Entry{
				Role:    message.Role,
				Content: text,
			},
			Category: category,
		})
		nextIndex++
	}
	return entries
}

func extractLastUserInput(messages []compat.Message) (string, int) {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != compat.RoleUser {
			continue
		}
		return extractMessageText(messages[i]), i
	}
	return "", -1
}

func extractMessageText(message compat.Message) string {
	parts := make([]string, 0, 1+len(message.ContentParts))
	if message.Content != "" {
		parts = append(parts, message.Content)
	}
	for i := range message.ContentParts {
		if message.ContentParts[i].Type != compat.ContentTypeText || message.ContentParts[i].Text == nil {
			continue
		}
		if *message.ContentParts[i].Text != "" {
			parts = append(parts, *message.ContentParts[i].Text)
		}
	}
	return strings.Join(parts, "\n")
}
