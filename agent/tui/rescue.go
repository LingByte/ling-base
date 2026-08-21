package tui

import (
	"errors"
	"strings"
)

// isRecoverableProviderError reports whether an error from the agent
// loop looks like a recoverable provider error — one where switching
// models or retrying after a brief wait might succeed. This drives the
// rescue hint shown after a failed turn.
//
// Matches on error message text because provider errors arrive as
// wrapped HTTP/API errors whose concrete types vary across providers.
func isRecoverableProviderError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// Auth errors.
	if strings.Contains(msg, "401") || strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "invalid api key") || strings.Contains(msg, "authentication") {
		return true
	}
	// Rate limits.
	if strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "too many requests") || strings.Contains(msg, "quota") {
		return true
	}
	// Temporary / server errors.
	if strings.Contains(msg, "500") || strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") || strings.Contains(msg, "504") ||
		strings.Contains(msg, "internal server error") ||
		strings.Contains(msg, "bad gateway") ||
		strings.Contains(msg, "service unavailable") ||
		strings.Contains(msg, "gateway timeout") ||
		strings.Contains(msg, "timeout") || strings.Contains(msg, "temporary") {
		return true
	}
	// Network errors.
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "context deadline exceeded") {
		return true
	}
	// Wrapped errors.
	return isRecoverableProviderError(errors.Unwrap(err))
}
