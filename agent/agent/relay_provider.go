// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/LingByte/ling-base/relay"
)

// RelayProvider implements agent.Provider by calling relay.Client.RichChatStream.
// It is the bridge between the agent loop (which speaks ContentBlock/RichMessage)
// and the relay layer (which speaks to Anthropic/OpenAI/Gemini/etc.).
type RelayProvider struct {
	client *relay.Client
	// idleTimeout aborts a stream if no chunk arrives within this duration.
	// 0 = no timeout (use ctx deadline only).
	idleTimeout time.Duration
}

// NewRelayProvider wraps a relay.Client as an agent.Provider.
func NewRelayProvider(client *relay.Client) *RelayProvider {
	return &RelayProvider{client: client, idleTimeout: 5 * time.Minute}
}

// NewRelayProviderWithIdleTimeout sets a custom idle timeout for stream chunks.
func NewRelayProviderWithIdleTimeout(client *relay.Client, idle time.Duration) *RelayProvider {
	return &RelayProvider{client: client, idleTimeout: idle}
}

// StreamTurn implements agent.Provider.
func (p *RelayProvider) StreamTurn(ctx context.Context, req *relay.RichChatRequest, sink StreamSink) (*relay.RichResponse, error) {
	req.Stream = true

	result, err := p.client.RichChatStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("relay stream: %w", err)
	}

	var final *relay.RichResponse
	// idleWatch fires if no chunk arrives within idleTimeout.
	var idleTimer *time.Timer
	var idleReset chan struct{}

	if p.idleTimeout > 0 {
		idleReset = make(chan struct{}, 1)
		idleTimer = time.NewTimer(p.idleTimeout)
		defer idleTimer.Stop()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for chunk := range result.Ch {
			if chunk.Err != nil {
				// Store error in a fake response for the caller.
				final = &relay.RichResponse{
					Content:    []relay.ContentBlock{relay.NewTextBlock(fmt.Sprintf("[Stream error: %v]", chunk.Err))},
					StopReason: "error",
				}
				return
			}
			if chunk.Done {
				continue
			}
			// Reset idle timer on every chunk.
			if idleReset != nil {
				select {
				case idleReset <- struct{}{}:
				default:
				}
			}
			// Forward text deltas.
			if chunk.Type == relay.ChunkTypeTextDelta && chunk.Text != "" {
				sink.Text(chunk.Text)
			}
			// Forward all chunks to the raw sink.
			sink.Chunk(chunk)
		}
		final = &result.Final
	}()

	if idleTimer != nil {
		select {
		case <-done:
		case <-idleTimer.C:
			return nil, fmt.Errorf("relay stream: idle timeout (%v) exceeded", p.idleTimeout)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else {
		select {
		case <-done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if final == nil {
		return nil, fmt.Errorf("relay stream: no response received")
	}
	return final, nil
}
