// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package gin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/ling-base/common/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain sets gin to test mode for all tests in this package.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// resetResolver ensures the package-level Resolver is cleared after each
// test so tests do not leak state into each other.
func resetResolver(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { Resolver = nil })
}

// newRouter returns a fresh gin engine in test mode.
func newRouter() *gin.Engine {
	return gin.New()
}

// successEnvelope is the shape of a success response body.
type successEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"msg"`
	Data    json.RawMessage `json:"data"`
}

// errorEnvelope mirrors buildEnvelope's output shape.
type errorEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"msg"`
	Error   string          `json:"error"`
	Data    json.RawMessage `json:"data"`
	Details json.RawMessage `json:"details"`
}

// runHandler invokes a single handler on a fresh router and returns the
// recorded response.
func runHandler(handler gin.HandlerFunc) *httptest.ResponseRecorder {
	r := newRouter()
	r.GET("/", handler)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	return w
}

// runHandlerWithMiddleware invokes a handler with the given middleware
// chain on a fresh router.
func runHandlerWithMiddleware(middleware gin.HandlerFunc, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	r := newRouter()
	r.Use(middleware)
	r.GET("/", handler)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	return w
}

// decodeSuccess parses a success envelope from the recorder body.
func decodeSuccess(t *testing.T, w *httptest.ResponseRecorder) successEnvelope {
	t.Helper()
	var env successEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	return env
}

// decodeError parses an error envelope from the recorder body.
func decodeError(t *testing.T, w *httptest.ResponseRecorder) errorEnvelope {
	t.Helper()
	var env errorEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	return env
}

// ──────────────────────────────────────────────
// Success helpers
// ──────────────────────────────────────────────

func TestSuccess(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { Success(c, "hello") })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeSuccess(t, w)
	assert.Equal(t, response.CodeSuccess, env.Code)
	assert.Equal(t, response.KeySuccess, env.Message)
	assert.Equal(t, `"hello"`, string(env.Data))
}

func TestSuccess_NilData(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { Success(c, nil) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeSuccess(t, w)
	assert.Equal(t, response.CodeSuccess, env.Code)
	assert.Equal(t, response.KeySuccess, env.Message)
	assert.Equal(t, "null", string(env.Data))
}

func TestSuccessMsg(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { SuccessMsg(c, "custom message", 42) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeSuccess(t, w)
	assert.Equal(t, response.CodeSuccess, env.Code)
	assert.Equal(t, "custom message", env.Message)
	assert.Equal(t, "42", string(env.Data))
}

func TestSuccessI18n_WithResolver(t *testing.T) {
	resetResolver(t)
	Resolver = response.ResolverFunc(func(key string, args ...any) string {
		if key == response.KeySuccess {
			return "成功"
		}
		return key
	})

	w := runHandler(func(c *gin.Context) { SuccessI18n(c, response.KeySuccess, "data") })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeSuccess(t, w)
	assert.Equal(t, response.CodeSuccess, env.Code)
	assert.Equal(t, "成功", env.Message)
	assert.Equal(t, `"data"`, string(env.Data))
}

func TestSuccessI18n_WithResolverAndArgs(t *testing.T) {
	resetResolver(t)
	Resolver = response.ResolverFunc(func(key string, args ...any) string {
		if key == "common.welcome" && len(args) == 1 {
			return "Welcome " + args[0].(string)
		}
		return key
	})

	w := runHandler(func(c *gin.Context) { SuccessI18n(c, "common.welcome", nil, "Alice") })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeSuccess(t, w)
	assert.Equal(t, response.CodeSuccess, env.Code)
	assert.Equal(t, "Welcome Alice", env.Message)
}

func TestSuccessI18n_WithoutResolver(t *testing.T) {
	resetResolver(t)
	// Resolver is nil, so the key should be returned as-is.
	w := runHandler(func(c *gin.Context) { SuccessI18n(c, response.KeySuccess, "data") })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeSuccess(t, w)
	assert.Equal(t, response.CodeSuccess, env.Code)
	assert.Equal(t, response.KeySuccess, env.Message)
}

func TestSuccessI18n_ResolverReturnsEmpty(t *testing.T) {
	resetResolver(t)
	// Resolver returns empty string -> falls back to key.
	Resolver = response.ResolverFunc(func(key string, args ...any) string { return "" })

	w := runHandler(func(c *gin.Context) { SuccessI18n(c, response.KeySuccess, "data") })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeSuccess(t, w)
	assert.Equal(t, response.KeySuccess, env.Message)
}

func TestCreated(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { Created(c, map[string]int{"id": 1}) })

	assert.Equal(t, http.StatusCreated, w.Code)
	env := decodeSuccess(t, w)
	assert.Equal(t, response.CodeSuccess, env.Code)
	assert.Equal(t, response.KeyCreated, env.Message)
	assert.Contains(t, string(env.Data), `"id":1`)
}

func TestNoContent(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { NoContent(c) })

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.Bytes())
}

// ──────────────────────────────────────────────
// Error helpers
// ──────────────────────────────────────────────

func TestWriteError_WithAppError(t *testing.T) {
	resetResolver(t)
	ae := response.Err(response.CodeNotFound).WithDetails(map[string]any{"field": "id"})
	w := runHandler(func(c *gin.Context) { WriteError(c, ae) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumNotFound, env.Code)
	assert.Equal(t, string(response.CodeNotFound), env.Error)
	assert.Equal(t, response.KeyNotFound, env.Message) // no resolver -> key returned
	assert.Contains(t, string(env.Details), `"field":"id"`)
}

func TestWriteError_WithAppError_OverrideStatus(t *testing.T) {
	resetResolver(t)
	ae := response.Err(response.CodeNotFound).WithStatus(http.StatusTeapot)
	w := runHandler(func(c *gin.Context) { WriteError(c, ae) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, string(response.CodeNotFound), env.Error)
}

func TestWriteError_WithPlainError(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { WriteError(c, errors.New("boom")) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumInternal, env.Code)
	assert.Equal(t, string(response.CodeInternal), env.Error)
	assert.Equal(t, "boom", env.Message) // plain error message used
}

func TestWriteError_WithNilError(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { WriteError(c, nil) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumInternal, env.Code)
	assert.Equal(t, string(response.CodeInternal), env.Error)
	// AsAppError(nil) -> Err(CodeInternal) which has MsgKey=KeyInternalError
	assert.Equal(t, response.KeyInternalError, env.Message)
}

func TestWriteError_WithResolver(t *testing.T) {
	resetResolver(t)
	Resolver = &response.StaticResolver{Messages: map[string]string{
		response.KeyNotFound: "资源未找到",
	}}
	ae := response.Err(response.CodeNotFound)
	w := runHandler(func(c *gin.Context) { WriteError(c, ae) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "资源未找到", env.Message)
}

func TestHandleError_Alias(t *testing.T) {
	resetResolver(t)
	ae := response.Err(response.CodeBadRequest)
	w := runHandler(func(c *gin.Context) { HandleError(c, ae) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumInvalidParams, env.Code)
	assert.Equal(t, string(response.CodeBadRequest), env.Error)
}

func TestFail(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { Fail(c, "something broke", nil) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumInternal, env.Code)
	assert.Equal(t, string(response.CodeInternal), env.Error)
	assert.Equal(t, "something broke", env.Message)
}

func TestFail_WithData(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { Fail(c, "err", map[string]string{"hint": "x"}) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Contains(t, string(env.Data), `"hint":"x"`)
}

func TestFailWithCode_NotFound(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { FailWithCode(c, response.CodeNotFound, "missing", nil) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumNotFound, env.Code)
	assert.Equal(t, string(response.CodeNotFound), env.Error)
	assert.Equal(t, "missing", env.Message)
}

func TestFailWithCode_BadRequest(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { FailWithCode(c, response.CodeBadRequest, "bad input", nil) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumInvalidParams, env.Code)
	assert.Equal(t, string(response.CodeBadRequest), env.Error)
	assert.Equal(t, "bad input", env.Message)
}

func TestFailWithCode_EmptyMsg(t *testing.T) {
	resetResolver(t)
	// Empty msg -> uses default MsgKey from Err(code).
	w := runHandler(func(c *gin.Context) { FailWithCode(c, response.CodeNotFound, "", nil) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumNotFound, env.Code)
	// With no resolver and empty Message, resolveMessage returns MsgKey.
	assert.Equal(t, response.KeyNotFound, env.Message)
}

func TestFailWithCode_EmptyMsg_WithResolver(t *testing.T) {
	resetResolver(t)
	Resolver = &response.StaticResolver{Messages: map[string]string{
		response.KeyNotFound: "Not Found (localized)",
	}}
	w := runHandler(func(c *gin.Context) { FailWithCode(c, response.CodeNotFound, "", nil) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "Not Found (localized)", env.Message)
}

func TestFailI18n_KeyNotFound(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { FailI18n(c, response.KeyNotFound, nil) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumNotFound, env.Code)
	assert.Equal(t, string(response.CodeNotFound), env.Error)
	// No resolver -> key returned as message.
	assert.Equal(t, response.KeyNotFound, env.Message)
}

func TestFailI18n_WithResolver(t *testing.T) {
	resetResolver(t)
	Resolver = &response.StaticResolver{Messages: map[string]string{
		response.KeyNotFound: "未找到",
	}}
	w := runHandler(func(c *gin.Context) { FailI18n(c, response.KeyNotFound, nil) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "未找到", env.Message)
}

func TestFailI18n_WithArgs(t *testing.T) {
	resetResolver(t)
	Resolver = &response.StaticResolver{Messages: map[string]string{
		"common.not_found": "Item %s not found",
	}}
	w := runHandler(func(c *gin.Context) { FailI18n(c, response.KeyNotFound, nil, "widget") })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, "Item widget not found", env.Message)
}

func TestFailAppError_Valid(t *testing.T) {
	resetResolver(t)
	ae := response.New(response.CodeConflict, "dup")
	w := runHandler(func(c *gin.Context) { FailAppError(c, ae) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumConflict, env.Code)
	assert.Equal(t, string(response.CodeConflict), env.Error)
	// New() sets both Message and MsgKey; resolver nil -> Message used.
	assert.Equal(t, "dup", env.Message)
}

func TestFailAppError_Nil(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) { FailAppError(c, nil) })

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumInternal, env.Code)
	assert.Equal(t, string(response.CodeInternal), env.Error)
}

func TestAbortWithStatusJSON_PlainError(t *testing.T) {
	resetResolver(t)
	w := runHandler(func(c *gin.Context) {
		AbortWithStatusJSON(c, http.StatusNotFound, errors.New("nope"))
	})

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, string(response.CodeNotFound), env.Error)
	assert.Equal(t, "nope", env.Message)
}

func TestAbortWithStatusJSON_AppError(t *testing.T) {
	resetResolver(t)
	ae := response.Err(response.CodeBadRequest)
	w := runHandler(func(c *gin.Context) {
		AbortWithStatusJSON(c, http.StatusInternalServerError, ae)
	})

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	// AppError code is BAD_REQUEST but httpStatus is 500; since ae.Code != CodeInternal,
	// the code is NOT overridden. So error stays BAD_REQUEST.
	assert.Equal(t, string(response.CodeBadRequest), env.Error)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAbortWithStatusJSON_AppErrorInternal_OverridesCode(t *testing.T) {
	resetResolver(t)
	// Plain error -> AsAppError wraps as CodeInternal. httpStatus=404 != 500,
	// so ae.Code gets overridden to CodeForHTTPStatus(404) = CodeNotFound.
	w := runHandler(func(c *gin.Context) {
		AbortWithStatusJSON(c, http.StatusNotFound, errors.New("missing"))
	})

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, string(response.CodeNotFound), env.Error)
}

func TestAbortWithStatusJSON_IsAborted(t *testing.T) {
	resetResolver(t)
	aborted := false
	r := newRouter()
	r.GET("/", func(c *gin.Context) {
		AbortWithStatusJSON(c, http.StatusForbidden, errors.New("denied"))
		aborted = c.IsAborted()
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	assert.True(t, aborted)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ──────────────────────────────────────────────
// Recovery middleware
// ──────────────────────────────────────────────

func TestRecovery_Panic(t *testing.T) {
	resetResolver(t)
	w := runHandlerWithMiddleware(Recovery(), func(c *gin.Context) {
		panic("boom!")
	})

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Equal(t, response.CodeNumInternal, env.Code)
	assert.Equal(t, string(response.CodeInternal), env.Error)
	assert.Contains(t, env.Message, "panic: boom!")
}

func TestRecovery_PanicWithError(t *testing.T) {
	resetResolver(t)
	w := runHandlerWithMiddleware(Recovery(), func(c *gin.Context) {
		panic(errors.New("custom error"))
	})

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeError(t, w)
	assert.Contains(t, env.Message, "panic: custom error")
}

func TestRecovery_NoPanic(t *testing.T) {
	resetResolver(t)
	w := runHandlerWithMiddleware(Recovery(), func(c *gin.Context) {
		Success(c, "ok")
	})

	assert.Equal(t, http.StatusOK, w.Code)
	env := decodeSuccess(t, w)
	assert.Equal(t, response.CodeSuccess, env.Code)
	assert.Equal(t, `"ok"`, string(env.Data))
}

// ──────────────────────────────────────────────
// codeForI18nKey mappings
// ──────────────────────────────────────────────

func TestCodeForI18nKey(t *testing.T) {
	cases := []struct {
		key  string
		want response.Code
	}{
		{response.KeyInvalidParams, response.CodeBadRequest},
		{response.KeyInvalidBody, response.CodeBadRequest},
		{response.KeyUnauthorized, response.CodeUnauthorized},
		{response.KeyAuthInvalidCredentials, response.CodeUnauthorized},
		{response.KeyAuthMissingToken, response.CodeUnauthorized},
		{response.KeyAuthInvalidToken, response.CodeUnauthorized},
		{response.KeyForbidden, response.CodeForbidden},
		{response.KeyPermInsufficient, response.CodeForbidden},
		{response.KeyNotFound, response.CodeNotFound},
		{response.KeyTenantNotFound, response.CodeNotFound},
		{response.KeyAuthEmailNotRegistered, response.CodeNotFound},
		{response.KeyConflict, response.CodeConflict},
		{response.KeyDuplicate, response.CodeConflict},
		{response.KeyTenantEmailExists, response.CodeConflict},
		{response.KeyRateLimited, response.CodeRateLimited},
		{response.KeyQuotaExceeded, response.CodeQuotaExceeded},
		{response.KeyUpstreamTimeout, response.CodeUpstreamTimeout},
		{response.KeyServiceUnavailable, response.CodeServiceUnavail},
		{response.KeyTenantMismatch, response.CodeTenantMismatch},
		{response.KeyTenantRegisterDisabled, response.CodeForbidden},
		{response.KeyTenantSuspended, response.CodeForbidden},
		{response.KeyTenantUserUnavailable, response.CodeForbidden},
		{"unknown.key", response.CodeInternal},
		{"", response.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			got := codeForI18nKey(tc.key)
			assert.Equal(t, tc.want, got, "key=%q", tc.key)
		})
	}
}

// ──────────────────────────────────────────────
// resolveMessage edge cases (via exported helpers)
// ──────────────────────────────────────────────

func TestResolveMessage_AppErrorWithMessageAndResolver(t *testing.T) {
	resetResolver(t)
	// When both Message and MsgKey are set and a resolver is configured,
	// the resolver result takes priority.
	Resolver = &response.StaticResolver{Messages: map[string]string{
		response.KeyNotFound: "localized",
	}}
	ae := response.New(response.CodeNotFound, "direct message")
	w := runHandler(func(c *gin.Context) { FailAppError(c, ae) })

	env := decodeError(t, w)
	assert.Equal(t, "localized", env.Message)
}

func TestResolveMessage_AppErrorWithMessageNoResolver(t *testing.T) {
	resetResolver(t)
	ae := response.New(response.CodeNotFound, "direct message")
	w := runHandler(func(c *gin.Context) { FailAppError(c, ae) })

	env := decodeError(t, w)
	assert.Equal(t, "direct message", env.Message)
}

func TestResolveMessage_ResolverReturnsEmptyFallsBackToMessage(t *testing.T) {
	resetResolver(t)
	Resolver = response.ResolverFunc(func(key string, args ...any) string { return "" })
	ae := response.New(response.CodeNotFound, "direct message")
	w := runHandler(func(c *gin.Context) { FailAppError(c, ae) })

	env := decodeError(t, w)
	assert.Equal(t, "direct message", env.Message)
}

func TestResolveMessage_NoMessageNoResolverReturnsKey(t *testing.T) {
	resetResolver(t)
	ae := response.Err(response.CodeNotFound) // only MsgKey set
	w := runHandler(func(c *gin.Context) { FailAppError(c, ae) })

	env := decodeError(t, w)
	assert.Equal(t, response.KeyNotFound, env.Message)
}

func TestResolveMessage_NoMessageNoResolverNoKeyReturnsCode(t *testing.T) {
	resetResolver(t)
	// Construct an AppError with no Message and no MsgKey.
	ae := &response.AppError{Code: response.CodeNotFound}
	w := runHandler(func(c *gin.Context) { FailAppError(c, ae) })

	env := decodeError(t, w)
	assert.Equal(t, string(response.CodeNotFound), env.Message)
}
