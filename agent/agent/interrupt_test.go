package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LingByte/ling-base/agent/tools"
	"github.com/LingByte/ling-base/relay"
)

// blockingProvider blocks in StreamTurn until the context is cancelled, then
// returns the context error — modeling an in-flight model call the user
// interrupts with Esc.
type blockingProvider struct{}

func (blockingProvider) StreamTurn(ctx context.Context, _ *relay.RichChatRequest, _ StreamSink) (*Response, error) {
	<-ctx.Done()
	return &Response{}, ctx.Err()
}

func TestRunReturnsOnContextCancel(t *testing.T) {
	read, _ := tools.NewRead()
	loop := New(blockingProvider{}, tools.NewRegistry(read))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := loop.Run(ctx, Options{Prompt: "hello", Model: "test"}, nil)
		done <- err
	}()

	// Simulate the Esc interrupt shortly after the turn starts.
	time.AfterFunc(20*time.Millisecond, cancel)

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancel (interrupt is broken)")
	}
}
