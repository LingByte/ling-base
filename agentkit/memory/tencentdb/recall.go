//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package tencentdb

import (
	"strings"

	compat "github.com/LingByte/ling-base/relay/compat"
)

func latestUserText(req *compat.Request) string {
	if req == nil {
		return ""
	}
	for i := len(req.Messages) - 1; i >= 0; i-- {
		msg := req.Messages[i]
		if msg.Role != compat.RoleUser {
			continue
		}
		if text := messageText(msg); text != "" {
			return text
		}
	}
	return ""
}

func injectRecallContext(req *compat.Request, rsp *recallResponse) {
	if req == nil || rsp == nil {
		return
	}
	if sys := strings.TrimSpace(firstNonEmpty(
		rsp.AppendSystemContext,
		rsp.Context,
	)); sys != "" {
		prependOrMergeSystem(req, sys)
	}
	if userCtx := strings.TrimSpace(rsp.PrependContext); userCtx != "" {
		insertBeforeLatestUser(req, compat.NewUserMessage(userCtx))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func prependOrMergeSystem(req *compat.Request, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	msg := compat.NewSystemMessage(text)
	if len(req.Messages) == 0 {
		req.Messages = []compat.Message{msg}
		return
	}
	for i := range req.Messages {
		if req.Messages[i].Role != compat.RoleSystem {
			continue
		}
		existing := strings.TrimSpace(req.Messages[i].Content)
		if existing == "" {
			req.Messages[i].Content = text
		} else {
			req.Messages[i].Content = existing + "\n\n" + text
		}
		return
	}
	req.Messages = append([]compat.Message{msg}, req.Messages...)
}

func insertBeforeLatestUser(req *compat.Request, msg compat.Message) {
	if req == nil {
		return
	}
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role != compat.RoleUser {
			continue
		}
		req.Messages = append(req.Messages[:i], append([]compat.Message{msg}, req.Messages[i:]...)...)
		return
	}
	req.Messages = append(req.Messages, msg)
}
