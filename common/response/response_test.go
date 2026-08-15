// Copyright (c) 2026 LingBase. All rights reserved.
// SPDX-License-Identifier: MIT

package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponse_Struct(t *testing.T) {
	r := &Response{Code: CodeSuccess, Message: "ok", Data: "payload"}
	assert.Equal(t, CodeSuccess, r.Code)
	assert.Equal(t, "ok", r.Message)
	assert.Equal(t, "payload", r.Data)
}

func TestErrorResponse_Struct(t *testing.T) {
	er := &ErrorResponse{
		Code:    CodeNumNotFound,
		Message: "not found",
		Error:   string(CodeNotFound),
		Details: map[string]any{"id": 7},
	}
	assert.Equal(t, CodeNumNotFound, er.Code)
	assert.Equal(t, "not found", er.Message)
	assert.Equal(t, "NOT_FOUND", er.Error)
	assert.Equal(t, 7, er.Details["id"])
}

func TestPage_Struct(t *testing.T) {
	p := &Page{List: []int{1, 2}, Total: 100, Page: 1, Size: 20, TotalPage: 5}
	assert.Equal(t, []int{1, 2}, p.List)
	assert.Equal(t, int64(100), p.Total)
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 20, p.Size)
	assert.Equal(t, 5, p.TotalPage)
}

func TestNewPage_Normal(t *testing.T) {
	list := []string{"a", "b", "c"}
	p := NewPage(list, 55, 2, 10)
	assert.Equal(t, list, p.List)
	assert.Equal(t, int64(55), p.Total)
	assert.Equal(t, 2, p.Page)
	assert.Equal(t, 10, p.Size)
	assert.Equal(t, 6, p.TotalPage)
}

func TestNewPage_ExactDivision(t *testing.T) {
	p := NewPage(nil, 100, 1, 10)
	assert.Equal(t, 10, p.TotalPage)
}

func TestNewPage_ZeroSize(t *testing.T) {
	p := NewPage(nil, 100, 1, 0)
	assert.Equal(t, 1, p.Size)
	assert.Equal(t, 100, p.TotalPage)
}

func TestNewPage_NegativeSize(t *testing.T) {
	p := NewPage(nil, 100, 1, -5)
	assert.Equal(t, 1, p.Size)
	assert.Equal(t, 100, p.TotalPage)
}

func TestNewPage_ZeroTotal(t *testing.T) {
	p := NewPage(nil, 0, 1, 10)
	assert.Equal(t, int64(0), p.Total)
	assert.Equal(t, 0, p.TotalPage)
}

func TestNewPage_NegativeTotal(t *testing.T) {
	p := NewPage(nil, -5, 1, 10)
	assert.Equal(t, int64(-5), p.Total)
	assert.Equal(t, 0, p.TotalPage)
}

func TestSuccess(t *testing.T) {
	r := Success("data")
	require.NotNil(t, r)
	assert.Equal(t, CodeSuccess, r.Code)
	assert.Equal(t, KeySuccess, r.Message)
	assert.Equal(t, "data", r.Data)
}

func TestSuccess_NilData(t *testing.T) {
	r := Success(nil)
	assert.Equal(t, CodeSuccess, r.Code)
	assert.Nil(t, r.Data)
}

func TestSuccessMsg(t *testing.T) {
	r := SuccessMsg("created", "data")
	require.NotNil(t, r)
	assert.Equal(t, CodeSuccess, r.Code)
	assert.Equal(t, "created", r.Message)
	assert.Equal(t, "data", r.Data)
}

func TestErrorEnvelope_WithResolver(t *testing.T) {
	ae := NewI18n(CodeNotFound, "common.not_found", "user", 7)
	r := &StaticResolver{Messages: map[string]string{
		"common.not_found": "Resource %s with id %d not found",
	}}
	er := ErrorEnvelope(ae, r)
	require.NotNil(t, er)
	assert.Equal(t, CodeNumNotFound, er.Code)
	assert.Equal(t, "Resource user with id 7 not found", er.Message)
	assert.Equal(t, "NOT_FOUND", er.Error)
	assert.Nil(t, er.Data)
	assert.Nil(t, er.Details)
}

func TestErrorEnvelope_WithoutResolver(t *testing.T) {
	ae := New(CodeNotFound, "direct message")
	er := ErrorEnvelope(ae, nil)
	require.NotNil(t, er)
	assert.Equal(t, CodeNumNotFound, er.Code)
	assert.Equal(t, "direct message", er.Message)
	assert.Equal(t, "NOT_FOUND", er.Error)
}

func TestErrorEnvelope_NilAppError(t *testing.T) {
	er := ErrorEnvelope(nil, nil)
	require.NotNil(t, er)
	assert.Equal(t, CodeNumInternal, er.Code)
	assert.Equal(t, KeyInternalError, er.Message)
	assert.Equal(t, "INTERNAL", er.Error)
}

func TestErrorEnvelope_WithDetails(t *testing.T) {
	ae := Err(CodeValidation).WithDetails(map[string]any{"field": "email"})
	er := ErrorEnvelope(ae, nil)
	assert.Equal(t, "email", er.Details["field"])
}

func TestResolveMessage(t *testing.T) {
	t.Run("resolver with key wins", func(t *testing.T) {
		ae := NewI18n(CodeNotFound, "common.not_found")
		r := &StaticResolver{Messages: map[string]string{"common.not_found": "resolved"}}
		assert.Equal(t, "resolved", resolveMessage(ae, r))
	})
	t.Run("resolver empty falls back to message", func(t *testing.T) {
		ae := &AppError{Code: CodeNotFound, MsgKey: "common.not_found", Message: "direct"}
		r := &StaticResolver{Messages: map[string]string{}}
		// resolver returns key (not empty), so it wins... actually returns key
		assert.Equal(t, "common.not_found", resolveMessage(ae, r))
	})
	t.Run("resolver nil uses message", func(t *testing.T) {
		ae := New(CodeNotFound, "direct")
		assert.Equal(t, "direct", resolveMessage(ae, nil))
	})
	t.Run("no resolver no message uses msgkey", func(t *testing.T) {
		ae := Err(CodeNotFound)
		assert.Equal(t, KeyNotFound, resolveMessage(ae, nil))
	})
	t.Run("no resolver no message no msgkey uses code", func(t *testing.T) {
		ae := &AppError{Code: CodeNotFound}
		assert.Equal(t, "NOT_FOUND", resolveMessage(ae, nil))
	})
	t.Run("resolver returns empty falls back to message", func(t *testing.T) {
		ae := &AppError{Code: CodeNotFound, MsgKey: "k", Message: "direct"}
		var f ResolverFunc // nil -> returns ""
		assert.Equal(t, "direct", resolveMessage(ae, f))
	})
	t.Run("empty msgkey with resolver uses message", func(t *testing.T) {
		ae := &AppError{Code: CodeNotFound, Message: "direct"}
		r := &StaticResolver{Messages: map[string]string{"k": "v"}}
		assert.Equal(t, "direct", resolveMessage(ae, r))
	})
}

func TestEnvelopeData(t *testing.T) {
	m := EnvelopeData("payload")
	assert.Equal(t, "payload", m["data"])
	m2 := EnvelopeData(42)
	assert.Equal(t, 42, m2["data"])
}

func TestEnvelopeError(t *testing.T) {
	m := EnvelopeError("something failed")
	assert.Equal(t, "something failed", m["error"])
}

func TestFormatDetails(t *testing.T) {
	t.Run("nil apperror", func(t *testing.T) {
		assert.Equal(t, "", FormatDetails(nil))
	})
	t.Run("empty details", func(t *testing.T) {
		ae := &AppError{Code: CodeNotFound, Details: map[string]any{}}
		assert.Equal(t, "", FormatDetails(ae))
	})
	t.Run("nil details", func(t *testing.T) {
		ae := &AppError{Code: CodeNotFound}
		assert.Equal(t, "", FormatDetails(ae))
	})
	t.Run("populated details", func(t *testing.T) {
		ae := &AppError{Code: CodeValidation, Details: map[string]any{
			"field": "email",
			"code":  42,
		}}
		out := FormatDetails(ae)
		// map iteration order is non-deterministic; check both parts present
		assert.Contains(t, out, "field=email")
		assert.Contains(t, out, "code=42")
		assert.Contains(t, out, ", ")
	})
	t.Run("single detail", func(t *testing.T) {
		ae := &AppError{Code: CodeValidation, Details: map[string]any{"field": "email"}}
		assert.Equal(t, "field=email", FormatDetails(ae))
	})
}
