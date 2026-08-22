//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package inprocess

import (
	"errors"
	"strings"

	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
)

type replyAccumulator struct {
	text     string
	builder  strings.Builder
	seenFull bool
	err      error
}

func (a *replyAccumulator) consume(evt *event.Event) {
	if evt == nil || evt.Response == nil {
		return
	}
	if evt.Response.Error != nil {
		a.err = errors.New(evt.Response.Error.Message)
		return
	}
	switch evt.Response.Object {
	case compat.ObjectTypeChatCompletion:
		a.consumeFull(evt.Response)
	case compat.ObjectTypeChatCompletionChunk:
		a.consumeDelta(evt.Response)
	}
}

func (a *replyAccumulator) consumeFull(rsp *compat.Response) {
	if rsp == nil || len(rsp.Choices) == 0 {
		return
	}
	content := rsp.Choices[0].Message.Content
	if content == "" {
		return
	}
	a.text = content
	a.seenFull = true
}

func (a *replyAccumulator) consumeDelta(rsp *compat.Response) {
	if rsp == nil || a.seenFull {
		return
	}
	for _, choice := range rsp.Choices {
		if choice.Delta.Content == "" {
			continue
		}
		a.builder.WriteString(choice.Delta.Content)
	}
	a.text = a.builder.String()
}

func trimResult(text string) string {
	return summarizeText(text, defaultStoredResultRunes)
}

func summarizeText(text string, limit int) string {
	trimmed := strings.TrimSpace(text)
	if limit <= 0 {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}
	return string(runes[:limit])
}
