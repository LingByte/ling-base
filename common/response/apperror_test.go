// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package response

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErr(t *testing.T) {
	ae := Err(CodeNotFound)
	assert.Equal(t, CodeNotFound, ae.Code)
	assert.Equal(t, KeyNotFound, ae.MsgKey)
	assert.Empty(t, ae.Message)
	assert.Nil(t, ae.Cause)
	assert.Empty(t, ae.MsgArgs)
	assert.Equal(t, 0, ae.HTTPStatus)
}

func TestNew(t *testing.T) {
	ae := New(CodeBadRequest, "bad input")
	assert.Equal(t, CodeBadRequest, ae.Code)
	assert.Equal(t, "bad input", ae.Message)
	assert.Equal(t, KeyInvalidParams, ae.MsgKey)
}

func TestNewf(t *testing.T) {
	ae := Newf(CodeConflict, "resource %d conflict for %s", 42, "user")
	assert.Equal(t, CodeConflict, ae.Code)
	assert.Equal(t, "resource 42 conflict for user", ae.Message)
	assert.Equal(t, KeyConflict, ae.MsgKey)
}

func TestNewI18n(t *testing.T) {
	ae := NewI18n(CodeNotFound, "common.not_found", "user", 7)
	assert.Equal(t, CodeNotFound, ae.Code)
	assert.Equal(t, "common.not_found", ae.MsgKey)
	assert.Equal(t, []any{"user", 7}, ae.MsgArgs)
	assert.Empty(t, ae.Message)
}

func TestWrap(t *testing.T) {
	cause := errors.New("db down")
	ae := Wrap(CodeInternal, "internal failure", cause)
	assert.Equal(t, CodeInternal, ae.Code)
	assert.Equal(t, "internal failure", ae.Message)
	assert.Equal(t, KeyInternalError, ae.MsgKey)
	assert.Same(t, cause, ae.Cause)
}

func TestWrapErr(t *testing.T) {
	cause := errors.New("io error")
	ae := WrapErr(CodeServiceUnavail, cause)
	assert.Equal(t, CodeServiceUnavail, ae.Code)
	assert.Equal(t, KeyServiceUnavailable, ae.MsgKey)
	assert.Same(t, cause, ae.Cause)
	assert.Empty(t, ae.Message)
}

func TestWrapI18n(t *testing.T) {
	cause := errors.New("upstream")
	ae := WrapI18n(CodeUpstreamTimeout, "common.upstream_timeout", cause, "svc", 3)
	assert.Equal(t, CodeUpstreamTimeout, ae.Code)
	assert.Equal(t, "common.upstream_timeout", ae.MsgKey)
	assert.Same(t, cause, ae.Cause)
	assert.Equal(t, []any{"svc", 3}, ae.MsgArgs)
}

func TestWithStatus(t *testing.T) {
	ae := Err(CodeNotFound).WithStatus(http.StatusGone)
	assert.Equal(t, http.StatusGone, ae.HTTPStatus)
	assert.Equal(t, http.StatusGone, HTTPStatusOf(ae))
}

func TestWithDetails(t *testing.T) {
	d := map[string]any{"field": "email"}
	ae := Err(CodeValidation).WithDetails(d)
	assert.Equal(t, d, ae.Details)
}

func TestWithCause(t *testing.T) {
	cause := errors.New("root")
	ae := Err(CodeInternal).WithCause(cause)
	assert.Same(t, cause, ae.Cause)
}

func TestWithArg(t *testing.T) {
	ae := NewI18n(CodeNotFound, "common.not_found").WithArg("user")
	assert.Equal(t, []any{"user"}, ae.MsgArgs)
	ae.WithArg(7)
	assert.Equal(t, []any{"user", 7}, ae.MsgArgs)
}

func TestWithArgs(t *testing.T) {
	ae := NewI18n(CodeNotFound, "common.not_found").WithArgs("a", 1, true)
	assert.Equal(t, []any{"a", 1, true}, ae.MsgArgs)
	// WithArgs replaces existing args
	ae.WithArgs("b")
	assert.Equal(t, []any{"b"}, ae.MsgArgs)
}

func TestAppError_Error(t *testing.T) {
	t.Run("message without cause", func(t *testing.T) {
		ae := New(CodeNotFound, "not found")
		assert.Equal(t, "not found", ae.Error())
	})
	t.Run("message with cause", func(t *testing.T) {
		ae := Wrap(CodeInternal, "internal", errors.New("db down"))
		assert.Equal(t, "internal: db down", ae.Error())
	})
	t.Run("msgkey without cause", func(t *testing.T) {
		ae := Err(CodeNotFound)
		assert.Equal(t, KeyNotFound, ae.Error())
	})
	t.Run("msgkey with cause", func(t *testing.T) {
		ae := WrapErr(CodeInternal, errors.New("boom"))
		assert.Equal(t, KeyInternalError+": boom", ae.Error())
	})
	t.Run("code fallback", func(t *testing.T) {
		ae := &AppError{Code: CodeNotFound}
		assert.Equal(t, "NOT_FOUND", ae.Error())
	})
	t.Run("code fallback with cause", func(t *testing.T) {
		ae := &AppError{Code: CodeNotFound, Cause: errors.New("x")}
		// Message empty, MsgKey empty -> returns code string (cause ignored)
		assert.Equal(t, "NOT_FOUND", ae.Error())
	})
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("root")
	ae := Wrap(CodeInternal, "fail", cause)
	assert.Same(t, cause, ae.Unwrap())

	ae2 := Err(CodeNotFound)
	assert.Nil(t, ae2.Unwrap())
}

func TestAppError_Is(t *testing.T) {
	t.Run("same code", func(t *testing.T) {
		ae := New(CodeNotFound, "a")
		target := Err(CodeNotFound)
		assert.True(t, errors.Is(ae, target))
	})
	t.Run("different code", func(t *testing.T) {
		ae := New(CodeNotFound, "a")
		target := Err(CodeBadRequest)
		assert.False(t, errors.Is(ae, target))
	})
	t.Run("non-AppError target", func(t *testing.T) {
		ae := New(CodeNotFound, "a")
		assert.False(t, errors.Is(ae, errors.New("plain")))
	})
	t.Run("wrapped chain", func(t *testing.T) {
		inner := Err(CodeConflict)
		outer := Wrap(CodeInternal, "outer", inner)
		assert.True(t, errors.Is(outer, Err(CodeConflict)))
	})
}

func TestHTTPStatusOf(t *testing.T) {
	t.Run("override", func(t *testing.T) {
		ae := Err(CodeNotFound).WithStatus(http.StatusTeapot)
		assert.Equal(t, http.StatusTeapot, HTTPStatusOf(ae))
	})
	t.Run("default", func(t *testing.T) {
		ae := Err(CodeNotFound)
		assert.Equal(t, http.StatusNotFound, HTTPStatusOf(ae))
	})
}

func TestAppError_NumCode(t *testing.T) {
	ae := Err(CodeNotFound)
	assert.Equal(t, CodeNumNotFound, ae.NumCode())

	ae2 := Err(CodeInternal)
	assert.Equal(t, CodeNumInternal, ae2.NumCode())
}

func TestIsAppError(t *testing.T) {
	t.Run("true for AppError", func(t *testing.T) {
		ae := Err(CodeNotFound)
		assert.True(t, IsAppError(ae))
	})
	t.Run("true for wrapped AppError", func(t *testing.T) {
		ae := Err(CodeNotFound)
		wrapped := fmt.Errorf("wrap: %w", ae)
		assert.True(t, IsAppError(wrapped))
	})
	t.Run("false for plain error", func(t *testing.T) {
		assert.False(t, IsAppError(errors.New("plain")))
	})
	t.Run("false for nil", func(t *testing.T) {
		assert.False(t, IsAppError(nil))
	})
}

func TestFrom(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, From(nil))
	})
	t.Run("already AppError returned as-is", func(t *testing.T) {
		ae := New(CodeNotFound, "nf")
		got := From(ae)
		require.NotNil(t, got)
		assert.Same(t, ae, got)
	})
	t.Run("wrapped AppError returned as-is", func(t *testing.T) {
		ae := New(CodeNotFound, "nf")
		wrapped := fmt.Errorf("ctx: %w", ae)
		got := From(wrapped)
		require.NotNil(t, got)
		assert.Same(t, ae, got)
	})
	t.Run("plain error becomes internal", func(t *testing.T) {
		plain := errors.New("boom")
		got := From(plain)
		require.NotNil(t, got)
		assert.Equal(t, CodeInternal, got.Code)
		assert.Equal(t, KeyInternalError, got.MsgKey)
		assert.Equal(t, "boom", got.Message)
		assert.Same(t, plain, got.Cause)
	})
}

func TestAsAppError(t *testing.T) {
	t.Run("nil input returns internal", func(t *testing.T) {
		got := AsAppError(nil)
		require.NotNil(t, got)
		assert.Equal(t, CodeInternal, got.Code)
		assert.Equal(t, KeyInternalError, got.MsgKey)
	})
	t.Run("plain error converted", func(t *testing.T) {
		plain := errors.New("boom")
		got := AsAppError(plain)
		require.NotNil(t, got)
		assert.Equal(t, CodeInternal, got.Code)
		assert.Equal(t, "boom", got.Message)
	})
	t.Run("AppError passthrough", func(t *testing.T) {
		ae := New(CodeNotFound, "nf")
		got := AsAppError(ae)
		assert.Same(t, ae, got)
	})
}
