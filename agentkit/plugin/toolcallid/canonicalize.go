//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package toolcallid

import (
	"fmt"

	"github.com/LingByte/ling-base/agentkit/agent"
	compat "github.com/LingByte/ling-base/relay/compat"
)

const canonicalToolCallIDPrefix = "trpc-agent-go-toolcall"

func canonicalizeResponse(inv *agent.Invocation, rsp *compat.Response) (*compat.Response, error) {
	if rsp == nil || rsp.IsPartial {
		return nil, nil
	}
	hasToolCalls := false
	for _, choice := range rsp.Choices {
		if len(choice.Message.ToolCalls) > 0 || len(choice.Delta.ToolCalls) > 0 {
			hasToolCalls = true
			break
		}
	}
	if !hasToolCalls {
		return nil, nil
	}
	invocationID := ""
	if inv != nil {
		invocationID = inv.InvocationID
	}
	cloned := rsp.Clone()
	for i, choice := range rsp.Choices {
		clonedChoice := cloned.Choices[i]
		canonicalizedMessageToolCalls, err := canonicalizeToolCalls(
			choice.Message.ToolCalls,
			invocationID,
			rsp.ID,
			choice.Index,
		)
		if err != nil {
			return nil, err
		}
		if canonicalizedMessageToolCalls != nil {
			clonedMessage := clonedChoice.Message
			clonedMessage.ToolCalls = canonicalizedMessageToolCalls
			clonedChoice.Message = clonedMessage
		}
		canonicalizedDeltaToolCalls, err := canonicalizeToolCalls(
			choice.Delta.ToolCalls,
			invocationID,
			rsp.ID,
			choice.Index,
		)
		if err != nil {
			return nil, err
		}
		if canonicalizedDeltaToolCalls != nil {
			clonedDelta := clonedChoice.Delta
			clonedDelta.ToolCalls = canonicalizedDeltaToolCalls
			clonedChoice.Delta = clonedDelta
		}
		cloned.Choices[i] = clonedChoice
	}
	return cloned, nil
}

func canonicalizeToolCalls(
	toolCalls []compat.ToolCall,
	invocationID string,
	responseID string,
	choiceIndex int,
) ([]compat.ToolCall, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}
	canonicalized := append([]compat.ToolCall(nil), toolCalls...)
	for slotIndex := range canonicalized {
		rawToolCallID := toolCalls[slotIndex].ID
		canonicalized[slotIndex].ID = canonicalToolCallID(
			invocationID,
			responseID,
			rawToolCallID,
			choiceIndex,
			slotIndex,
		)
	}
	return canonicalized, nil
}

func canonicalToolCallID(
	invocationID string,
	responseID string,
	rawToolCallID string,
	choiceIndex int,
	slotIndex int,
) string {
	return fmt.Sprintf(
		"%s:%s:%s:%s:c%d:t%d",
		canonicalToolCallIDPrefix,
		invocationID,
		responseID,
		rawToolCallID,
		choiceIndex,
		slotIndex,
	)
}
