package middleware

import (
	"context"
	"fmt"

	"github.com/LingByte/ling-base/agentkit/flowcraft/core/message"
	"github.com/LingByte/ling-base/agentkit/flowcraft/core/telemetry"
	"github.com/LingByte/ling-base/agentkit/flowcraft/core/tool"

	otellog "go.opentelemetry.io/otel/log"
)

// Recover converts a panicking tool (or inner middleware) into an
// IsError result instead of crashing the caller's goroutine. Put it
// first in the chain: without it a panic inside ExecuteAll escapes
// the fan-out goroutine and takes the process down.
func Recover() tool.Middleware {
	return func(next tool.Dispatch) tool.Dispatch {
		return func(ctx context.Context, call message.ToolCall) (res message.ToolResult) {
			defer func() {
				if rv := recover(); rv != nil {
					telemetry.Warn(ctx, "tool panicked",
						otellog.String(telemetry.AttrToolName, call.Name),
						otellog.String(telemetry.AttrToolCallID, call.ID),
						otellog.String(telemetry.AttrErrorMessage, fmt.Sprint(rv)))
					res = message.ToolResult{
						CallID:  call.ID,
						Content: fmt.Sprintf("tool %q panicked: %v", call.Name, rv),
						IsError: true,
					}
				}
			}()
			return next(ctx, call)
		}
	}
}
