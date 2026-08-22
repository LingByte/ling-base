//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package dify

import (
	"context"
	"strings"
	"time"

	"github.com/cloudernative/dify-sdk-go"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	log "github.com/LingByte/ling-base/common/logger"
	compat "github.com/LingByte/ling-base/relay/compat"
)

// DifyEventConverter defines an interface for converting Dify response to Event.
type DifyEventConverter interface {
	// ConvertToEvent converts an A2A protocol type to an Event.
	ConvertToEvent(
		resp *dify.ChatMessageResponse,
		agentName string,
		invocation *agent.Invocation,
	) *event.Event

	// ConvertStreamingToEvent converts a streaming A2A protocol type to an Event.
	ConvertStreamingToEvent(
		resp dify.ChatMessageStreamChannelResponse,
		agentName string,
		invocation *agent.Invocation,
	) *event.Event
}

// DifyRequestConverter defines an interface for converting invocations to Dify request messages.
type DifyRequestConverter interface {
	// ConvertToDifyRequest converts agent invocation to Dify request
	ConvertToDifyRequest(
		ctx context.Context,
		invocation *agent.Invocation,
		isStream bool,
	) (*dify.ChatMessageRequest, error)
}

// DifyWorkflowRequestConverter defines an interface for converting invocations to Dify workflow requests.
type DifyWorkflowRequestConverter interface {
	// ConvertToWorkflowRequest converts agent invocation to Dify workflow request
	ConvertToWorkflowRequest(
		ctx context.Context,
		invocation *agent.Invocation,
	) (dify.WorkflowRequest, error)
}

// defaultDifyEventConverter is the default implementation of DifyEventConverter.
// It converts Dify chatflow/workflow responses to internal event format.
type defaultDifyEventConverter struct {
}

// ConvertToEvent converts a Dify ChatMessageResponse to an internal Event.
// If resp is nil, it returns a default empty assistant message event.
func (d *defaultDifyEventConverter) ConvertToEvent(
	resp *dify.ChatMessageResponse,
	agentName string,
	invocation *agent.Invocation,
) (evt *event.Event) {
	if resp == nil {
		defaultMessage := compat.Message{Role: compat.RoleAssistant, Content: ""}
		evt = event.NewResponseEvent(
			invocation.InvocationID,
			agentName,
			&compat.Response{Choices: []compat.Choice{{Message: defaultMessage, Delta: defaultMessage}}},
		)
		return
	}

	// set Dify response ID, ensure AG-UI translator can correctly identify the message
	responseID := resp.ID
	if responseID == "" {
		responseID = resp.ConversationID
	}

	message := compat.Message{
		Role:    compat.RoleAssistant,
		Content: resp.Answer,
	}

	evt = event.New(
		invocation.InvocationID,
		agentName,
		event.WithResponse(&compat.Response{
			ID:        responseID,
			Object:    compat.ObjectTypeChatCompletion,
			Choices:   []compat.Choice{{Message: message, Delta: message}},
			Timestamp: time.Now(),
			Created:   time.Now().Unix(),
			IsPartial: false,
			Done:      true,
		}),
	)
	return
}

// ConvertStreamingToEvent converts a Dify streaming response to an internal Event.
// Returns nil if the response Answer is empty.
func (d *defaultDifyEventConverter) ConvertStreamingToEvent(
	resp dify.ChatMessageStreamChannelResponse,
	agentName string,
	invocation *agent.Invocation,
) (evt *event.Event) {
	if resp.Answer == "" {
		return
	}

	// set Dify return MessageID as Response.ID, ensure AG-UI translator
	// can correctly trigger TextMessageStartEvent. If MessageID is empty, fall back in order.
	responseID := resp.MessageID
	if responseID == "" {
		responseID = resp.ConversationID
	}
	if responseID == "" {
		responseID = resp.ID
	}

	message := compat.Message{
		Role:    compat.RoleAssistant,
		Content: resp.Answer,
	}

	evt = event.New(
		invocation.InvocationID,
		agentName,
		event.WithResponse(&compat.Response{
			ID:        responseID,
			Object:    compat.ObjectTypeChatCompletionChunk,
			Choices:   []compat.Choice{{Delta: message}},
			Timestamp: time.Now(),
			Created:   time.Now().Unix(),
			IsPartial: true,
			Done:      false,
		}),
		event.WithObject(compat.ObjectTypeChatCompletionChunk),
	)
	return
}

// defaultEventDifyConverter is the default implementation of DifyRequestConverter.
// It converts agent invocations to Dify ChatMessageRequest format.
type defaultEventDifyConverter struct {
}

// ConvertToDifyRequest converts an agent invocation to a Dify ChatMessageRequest.
// It handles text, image, and file content parts from the invocation message.
func (d *defaultEventDifyConverter) ConvertToDifyRequest(
	ctx context.Context,
	invocation *agent.Invocation,
	isStream bool,
) (*dify.ChatMessageRequest, error) {
	req := &dify.ChatMessageRequest{
		Query:  invocation.Message.Content,
		Inputs: make(map[string]any),
	}
	if invocation.Session != nil {
		req.User = invocation.Session.UserID
	}
	if req.User == "" {
		req.User = "anonymous"
	}

	// Enable streaming response mode
	if isStream {
		req.ResponseMode = "streaming"
	}

	// Handle content parts if available
	for _, contentPart := range invocation.Message.ContentParts {
		switch contentPart.Type {
		case compat.ContentTypeText:
			if contentPart.Text != nil {
				// Append text content to query
				if req.Query != "" {
					req.Query += "\n" + *contentPart.Text
				} else {
					req.Query = *contentPart.Text
				}
			}
		case compat.ContentTypeImage:
			// For now, we can't directly handle images in Dify requests
			// This would need to be handled based on specific Dify capabilities
			if contentPart.Image != nil && contentPart.Image.URL != "" {
				req.Inputs["image_url"] = contentPart.Image.URL
			}
		case compat.ContentTypeFile:
			// Similar to images, file handling depends on Dify capabilities
			if contentPart.File != nil && contentPart.File.Name != "" {
				req.Inputs["file_name"] = contentPart.File.Name
			}
		default:
			// Handle other content types as needed
			req.Inputs["other_content_type"] = contentPart.Type
		}
	}

	return req, nil
}

// defaultWorkflowRequestConverter is the default implementation of DifyWorkflowRequestConverter.
// It converts agent invocations to Dify WorkflowRequest format.
type defaultWorkflowRequestConverter struct{}

// ConvertToWorkflowRequest converts an agent invocation to a Dify WorkflowRequest.
// It extracts query, image, and file inputs from the invocation message content parts.
func (d *defaultWorkflowRequestConverter) ConvertToWorkflowRequest(
	ctx context.Context,
	invocation *agent.Invocation,
) (dify.WorkflowRequest, error) {
	inputs := make(map[string]any)
	inputs["query"] = invocation.Message.Content

	// Handle content parts if available
	for _, contentPart := range invocation.Message.ContentParts {
		switch contentPart.Type {
		case compat.ContentTypeText:
			if contentPart.Text != nil {
				inputs["query"] = *contentPart.Text
			}
		case compat.ContentTypeImage:
			if contentPart.Image != nil && contentPart.Image.URL != "" {
				inputs["image_url"] = contentPart.Image.URL
			}
		case compat.ContentTypeFile:
			if contentPart.File != nil && contentPart.File.Name != "" {
				inputs["file_name"] = contentPart.File.Name
			}
		default:
			inputs["other_content_type"] = contentPart.Type
		}
	}

	user := "anonymous"
	if invocation.Session != nil && invocation.Session.UserID != "" {
		user = invocation.Session.UserID
	}

	return dify.WorkflowRequest{
		Inputs:       inputs,
		User:         user,
		ResponseMode: "blocking", // workflow default mode
	}, nil
}

// extractTextFromParts extracts text content from protocol message parts
func extractTextFromParts(parts []protocol.Part) string {
	var content strings.Builder
	for _, part := range parts {
		if part.GetKind() == protocol.KindText {
			p, ok := part.(*protocol.TextPart)
			if !ok {
				log.Warnf("unexpected part type: %T", part)
				continue
			}
			content.WriteString(p.Text)
		}
	}
	return content.String()
}

// buildResponseForEvent creates a Response object based on streaming mode
func buildResponseForEvent(isStreaming bool, content string) *compat.Response {
	message := compat.Message{
		Role:    compat.RoleAssistant,
		Content: content,
	}

	if isStreaming {
		return &compat.Response{
			Choices:   []compat.Choice{{Delta: message}},
			Timestamp: time.Now(),
			Created:   time.Now().Unix(),
			IsPartial: true,
			Done:      false,
		}
	}

	return &compat.Response{
		Choices:   []compat.Choice{{Message: message}},
		Timestamp: time.Now(),
		Created:   time.Now().Unix(),
		IsPartial: false,
		Done:      true,
	}
}

// buildRespEvent converts A2A response to tRPC event
func (d *defaultDifyEventConverter) buildRespEvent(
	isStreaming bool,
	msg *protocol.Message,
	agentName string,
	invocation *agent.Invocation) *event.Event {

	// Extract text content from parts
	content := extractTextFromParts(msg.Parts)

	// Create event with appropriate response
	evt := event.New(invocation.InvocationID, agentName)
	evt.Response = buildResponseForEvent(isStreaming, content)

	return evt
}

// convertTaskToMessage converts a Task to a Message
func convertTaskToMessage(task *protocol.Task) *protocol.Message {
	var parts []protocol.Part

	// Add artifacts if any
	for _, artifact := range task.Artifacts {
		parts = append(parts, artifact.Parts...)
	}

	return &protocol.Message{
		Role:  protocol.MessageRoleAgent,
		Parts: parts,
	}
}

// convertTaskStatusToMessage converts a TaskStatusUpdateEvent to a Message
func convertTaskStatusToMessage(event *protocol.TaskStatusUpdateEvent) *protocol.Message {
	msg := &protocol.Message{
		Role: protocol.MessageRoleAgent,
	}
	if event.Status.Message != nil {
		msg.Parts = event.Status.Message.Parts
	}
	return msg
}

// convertTaskArtifactToMessage converts a TaskArtifactUpdateEvent to a Message
func convertTaskArtifactToMessage(event *protocol.TaskArtifactUpdateEvent) *protocol.Message {
	return &protocol.Message{
		Role:  protocol.MessageRoleAgent,
		Parts: event.Artifact.Parts,
	}
}
