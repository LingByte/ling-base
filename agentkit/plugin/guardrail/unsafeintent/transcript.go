//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package unsafeintent

import (
	"context"

	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/plugin/guardrail/internal/currentinput"
	guardtranscript "github.com/LingByte/ling-base/agentkit/plugin/guardrail/internal/transcript"
	unsafereview "github.com/LingByte/ling-base/agentkit/plugin/guardrail/unsafeintent/review"
)

func (p *Plugin) buildReviewRequest(ctx context.Context, messages []compat.Message) *unsafereview.Request {
	req := currentinput.Build(ctx, messages, p.tokenCounter, func(entry guardtranscript.Entry) unsafereview.TranscriptEntry {
		return unsafereview.TranscriptEntry{
			Role:    entry.Role,
			Content: entry.Content,
		}
	})
	if req == nil {
		return nil
	}
	return &unsafereview.Request{
		LastUserInput: req.LastUserInput,
		Transcript:    req.Transcript,
	}
}
