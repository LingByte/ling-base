//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package trpcagent

import (
	astructure "github.com/LingByte/ling-base/agentkit/agent/structure"
	atrace "github.com/LingByte/ling-base/agentkit/agent/trace"
	"github.com/LingByte/ling-base/agentkit/event"
	"github.com/LingByte/ling-base/agentkit/internal/profilecompiler"
	"github.com/LingByte/ling-base/agentkit/model"
)

type session struct {
	UserID    string `json:"userId"`
	SessionID string `json:"sessionId"`
}

type runOptions struct {
	RequestID             string         `json:"requestID,omitempty"`
	ExecutionTraceEnabled bool           `json:"executionTraceEnabled,omitempty"`
	RuntimeState          map[string]any `json:"runtimeState,omitempty"`
}

type runRequest struct {
	Session session       `json:"session"`
	Input   model.Message `json:"input"`
	// Profile must be runtime-normalized and include nodeID and type.
	Profile    *profilecompiler.Profile `json:"profile,omitempty"`
	RunOptions runOptions               `json:"runOptions,omitempty"`
}

type runResponse struct {
	Status         atrace.TraceStatus `json:"status"`
	Events         []event.Event      `json:"events,omitempty"`
	ExecutionTrace *atrace.Trace      `json:"executionTrace,omitempty"`
	ErrorMessage   string             `json:"errorMessage,omitempty"`
}

type structureResponse struct {
	Structure *astructure.Snapshot `json:"structure"`
}
