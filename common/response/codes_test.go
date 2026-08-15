// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package response

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPStatusFor(t *testing.T) {
	tests := []struct {
		name     string
		code     Code
		expected int
	}{
		{"bad request", CodeBadRequest, http.StatusBadRequest},
		{"validation", CodeValidation, http.StatusBadRequest},
		{"unauthorized", CodeUnauthorized, http.StatusUnauthorized},
		{"auth failed", CodeAuthFailed, http.StatusUnauthorized},
		{"credential invalid", CodeCredentialInvalid, http.StatusUnauthorized},
		{"forbidden", CodeForbidden, http.StatusForbidden},
		{"tenant mismatch", CodeTenantMismatch, http.StatusForbidden},
		{"not found", CodeNotFound, http.StatusNotFound},
		{"conflict", CodeConflict, http.StatusConflict},
		{"duplicate", CodeDuplicate, http.StatusConflict},
		{"rate limited", CodeRateLimited, http.StatusTooManyRequests},
		{"quota exceeded", CodeQuotaExceeded, http.StatusPaymentRequired},
		{"upstream timeout", CodeUpstreamTimeout, http.StatusGatewayTimeout},
		{"service unavailable", CodeServiceUnavail, http.StatusServiceUnavailable},
		{"provider error", CodeProviderError, http.StatusServiceUnavailable},
		{"internal", CodeInternal, http.StatusInternalServerError},
		{"unknown code", Code("UNKNOWN_CODE"), http.StatusInternalServerError},
		{"empty code", Code(""), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, HTTPStatusFor(tt.code))
		})
	}
}

func TestErrCodeFor(t *testing.T) {
	tests := []struct {
		name     string
		code     Code
		expected int
	}{
		{"bad request", CodeBadRequest, CodeNumInvalidParams},
		{"validation", CodeValidation, CodeNumValidationFailed},
		{"unauthorized", CodeUnauthorized, CodeNumUnauthorized},
		{"auth failed", CodeAuthFailed, CodeNumUnauthorized},
		{"credential invalid", CodeCredentialInvalid, CodeNumUnauthorized},
		{"forbidden", CodeForbidden, CodeNumForbidden},
		{"tenant mismatch", CodeTenantMismatch, CodeNumTenantMismatch},
		{"not found", CodeNotFound, CodeNumNotFound},
		{"conflict", CodeConflict, CodeNumConflict},
		{"duplicate", CodeDuplicate, CodeNumConflict},
		{"rate limited", CodeRateLimited, CodeNumRateLimited},
		{"quota exceeded", CodeQuotaExceeded, CodeNumQuotaExceeded},
		{"upstream timeout", CodeUpstreamTimeout, CodeNumUpstreamTimeout},
		{"service unavailable", CodeServiceUnavail, CodeNumServiceUnavail},
		{"provider error", CodeProviderError, CodeNumServiceUnavail},
		{"internal", CodeInternal, CodeNumInternal},
		{"unknown code", Code("UNKNOWN_CODE"), CodeNumInternal},
		{"empty code", Code(""), CodeNumInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ErrCodeFor(tt.code))
		})
	}
}

func TestCodeForHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected Code
	}{
		{"bad request", http.StatusBadRequest, CodeBadRequest},
		{"unauthorized", http.StatusUnauthorized, CodeUnauthorized},
		{"forbidden", http.StatusForbidden, CodeForbidden},
		{"not found", http.StatusNotFound, CodeNotFound},
		{"conflict", http.StatusConflict, CodeConflict},
		{"too many requests", http.StatusTooManyRequests, CodeRateLimited},
		{"payment required", http.StatusPaymentRequired, CodeQuotaExceeded},
		{"gateway timeout", http.StatusGatewayTimeout, CodeUpstreamTimeout},
		{"service unavailable", http.StatusServiceUnavailable, CodeServiceUnavail},
		{"internal default", http.StatusInternalServerError, CodeInternal},
		{"unknown status", http.StatusTeapot, CodeInternal},
		{"zero status", 0, CodeInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, CodeForHTTPStatus(tt.status))
		})
	}
}
