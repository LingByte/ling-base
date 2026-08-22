//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package currentinput

import (
	"context"
	"errors"
	"testing"

	compat "github.com/LingByte/ling-base/relay/compat"
	guardtranscript "github.com/LingByte/ling-base/agentkit/plugin/guardrail/internal/transcript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedTokenCounter struct {
	count int
	err   error
}

func (c fixedTokenCounter) CountTokens(ctx context.Context, message compat.Message) (int, error) {
	return c.count, c.err
}

func (c fixedTokenCounter) CountTokensRange(
	ctx context.Context,
	messages []compat.Message,
	start, end int,
) (int, error) {
	return c.count, c.err
}

func TestBuild_KeepsLatestUserInputOutsideTranscript(t *testing.T) {
	req := Build(context.Background(), []compat.Message{
		{Role: compat.RoleUser, Content: "Earlier user context."},
		{Role: compat.RoleAssistant, Content: "Assistant context."},
		{Role: compat.RoleUser, Content: "Latest user input."},
	}, fixedTokenCounter{count: 1}, func(entry guardtranscript.Entry) guardtranscript.Entry {
		return entry
	})
	require.NotNil(t, req)
	require.Len(t, req.Transcript, 2)
	assert.Equal(t, compat.RoleUser, req.Transcript[0].Role)
	assert.Equal(t, "Earlier user context.", req.Transcript[0].Content)
	assert.Equal(t, compat.RoleAssistant, req.Transcript[1].Role)
	assert.Equal(t, "Assistant context.", req.Transcript[1].Content)
	assert.Equal(t, "Latest user input.", req.LastUserInput)
}

func TestBuild_KeepsFullLatestUserInput(t *testing.T) {
	longInput := repeat("user ", guardtranscript.DefaultMessageEntryCap+10)
	req := Build(context.Background(), []compat.Message{
		{Role: compat.RoleUser, Content: longInput},
	}, fixedTokenCounter{count: 1}, func(entry guardtranscript.Entry) guardtranscript.Entry {
		return entry
	})
	require.NotNil(t, req)
	require.Empty(t, req.Transcript)
	assert.Equal(t, longInput, req.LastUserInput)
}

func TestBuild_CountTokenFailureFallsBackToOmission(t *testing.T) {
	req := Build(context.Background(), []compat.Message{
		{Role: compat.RoleUser, Content: "Latest user input."},
		{Role: compat.RoleAssistant, Content: "Assistant context."},
	}, fixedTokenCounter{err: errors.New("count tokens failed")}, func(entry guardtranscript.Entry) guardtranscript.Entry {
		return entry
	})
	require.NotNil(t, req)
	require.Len(t, req.Transcript, 1)
	assert.Equal(t, compat.RoleAssistant, req.Transcript[0].Role)
	assert.Equal(t, guardtranscript.DefaultOmissionNote, req.Transcript[0].Content)
	assert.Equal(t, "Latest user input.", req.LastUserInput)
}

func TestBuild_WithoutLatestUserInputReturnsNil(t *testing.T) {
	req := Build(context.Background(), []compat.Message{
		{Role: compat.RoleAssistant, Content: "Assistant context."},
		{Role: compat.RoleTool, Content: "Tool context."},
	}, fixedTokenCounter{count: 1}, func(entry guardtranscript.Entry) guardtranscript.Entry {
		return entry
	})
	require.Nil(t, req)
}

func TestBuild_WithoutTextInLatestUserMessageReturnsNil(t *testing.T) {
	req := Build(context.Background(), []compat.Message{
		{Role: compat.RoleUser, Content: "Earlier user context."},
		{
			Role: compat.RoleUser,
			ContentParts: []compat.ContentPart{{
				Type:  compat.ContentTypeImage,
				Image: &compat.Image{URL: "https://example.com/image.png"},
			}},
		},
	}, fixedTokenCounter{count: 1}, func(entry guardtranscript.Entry) guardtranscript.Entry {
		return entry
	})
	require.Nil(t, req)
}

func repeat(value string, n int) string {
	if n <= 0 {
		return ""
	}
	result := make([]byte, 0, len(value)*n)
	for i := 0; i < n; i++ {
		result = append(result, value...)
	}
	return string(result)
}
