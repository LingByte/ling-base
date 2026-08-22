//
// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT
//

package relaymodel

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	compat "github.com/LingByte/ling-base/relay/compat"
)

// TestRelayModelLive tests the relay-backed model against a live endpoint.
// Set RIGHTAPI_API_KEY to run this test.
func TestRelayModelLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test in short mode")
	}
	apiKey := os.Getenv("RIGHTAPI_API_KEY")
	if apiKey == "" {
		t.Skip("set RIGHTAPI_API_KEY to run this live test")
	}

	m := New("gpt-5.4",
		WithAPIKey(apiKey),
		WithBaseURL("https://rightapi.ai/codex"),
		WithChannel(ChannelOpenAI),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("You are a helpful assistant. Answer concisely."),
			compat.NewUserMessage("What is 2+2? Answer in one word."),
		},
		GenerationConfig: compat.GenerationConfig{
			Stream:    true,
			MaxTokens: ptrInt(100),
		},
	}

	respCh, err := m.GenerateContent(ctx, req)
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}

	var textBuf strings.Builder
	var gotFinal bool
	for resp := range respCh {
		if resp.Error != nil {
			t.Fatalf("response error: %v", resp.Error)
		}
		if resp.Done {
			gotFinal = true
			if resp.Usage == nil {
				t.Error("final response has no usage")
			} else {
				t.Logf("usage: prompt=%d completion=%d",
					resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
			}
			if len(resp.Choices) > 0 {
				finishReason := ""
				if resp.Choices[0].FinishReason != nil {
					finishReason = *resp.Choices[0].FinishReason
				}
				t.Logf("finish_reason: %s", finishReason)
				if finishReason != "stop" {
					t.Errorf("finish_reason = %q, want stop", finishReason)
				}
			}
		}
		for _, ch := range resp.Choices {
			if ch.Delta.Content != "" {
				textBuf.WriteString(ch.Delta.Content)
			}
		}
	}

	if !gotFinal {
		t.Error("did not receive final response")
	}
	text := strings.TrimSpace(textBuf.String())
	t.Logf("response text: %q", text)
	if text == "" {
		t.Error("response text is empty")
	}
}

// TestRelayModelProviderLive tests using the provider package to construct
// a relay-backed compat.
func TestRelayModelProviderLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test in short mode")
	}
	apiKey := os.Getenv("RIGHTAPI_API_KEY")
	if apiKey == "" {
		t.Skip("set RIGHTAPI_API_KEY to run this live test")
	}

	m := New("gpt-5.4",
		WithAPIKey(apiKey),
		WithBaseURL("https://rightapi.ai/codex"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("Say hello in one word."),
		},
		GenerationConfig: compat.GenerationConfig{
			Stream:    true,
			MaxTokens: ptrInt(50),
		},
	}

	respCh, err := m.GenerateContent(ctx, req)
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}

	var textBuf strings.Builder
	for resp := range respCh {
		if resp.Error != nil {
			t.Fatalf("response error: %v", resp.Error)
		}
		for _, ch := range resp.Choices {
			if ch.Delta.Content != "" {
				textBuf.WriteString(ch.Delta.Content)
			}
		}
	}
	text := strings.TrimSpace(textBuf.String())
	t.Logf("response: %q", text)
	if text == "" {
		t.Error("response is empty")
	}
}
