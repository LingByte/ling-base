// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package validate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldError_ErrorWithMessage(t *testing.T) {
	fe := FieldError{
		Field:   "User.Email",
		Rule:    "email",
		Value:   "not-an-email",
		Message: "must be a valid email address",
	}
	msg := fe.Error()
	assert.Contains(t, msg, `field "User.Email"`)
	assert.Contains(t, msg, "must be a valid email address")
}

func TestFieldError_ErrorWithoutMessage(t *testing.T) {
	fe := FieldError{
		Field: "User.Name",
		Rule:  "required",
		Value: "",
	}
	msg := fe.Error()
	assert.Contains(t, msg, `field "User.Name"`)
	assert.Contains(t, msg, `rule "required"`)
	assert.NotContains(t, msg, "is required")
}

func TestErrors_ErrorEmpty(t *testing.T) {
	var errs Errors
	assert.Equal(t, "validate: no errors", errs.Error())
}

func TestErrors_ErrorSingle(t *testing.T) {
	errs := Errors{
		{Field: "Name", Rule: "required", Value: "", Message: "is required"},
	}
	msg := errs.Error()
	assert.Contains(t, msg, `field "Name"`)
	assert.NotContains(t, msg, "1 errors:")
}

func TestErrors_ErrorMultiple(t *testing.T) {
	errs := Errors{
		{Field: "Name", Rule: "required", Value: "", Message: "is required"},
		{Field: "Email", Rule: "email", Value: "x", Message: "must be a valid email address"},
	}
	msg := errs.Error()
	assert.Contains(t, msg, "2 errors:")
	assert.Contains(t, msg, `field "Name"`)
	assert.Contains(t, msg, `field "Email"`)
	assert.True(t, strings.HasSuffix(msg, "\n"))
}

func TestErrors_Has(t *testing.T) {
	errs := Errors{
		{Field: "Name", Rule: "required"},
		{Field: "Email", Rule: "email"},
	}
	assert.True(t, errs.Has("Name"))
	assert.True(t, errs.Has("Email"))
	assert.False(t, errs.Has("Age"))
}

func TestErrors_ForField(t *testing.T) {
	errs := Errors{
		{Field: "Name", Rule: "required"},
		{Field: "Name", Rule: "min"},
		{Field: "Email", Rule: "email"},
	}
	nameErrs := errs.ForField("Name")
	require.Len(t, nameErrs, 2)
	assert.Equal(t, "required", nameErrs[0].Rule)
	assert.Equal(t, "min", nameErrs[1].Rule)

	emailErrs := errs.ForField("Email")
	require.Len(t, emailErrs, 1)

	none := errs.ForField("Age")
	assert.Len(t, none, 0)
}

func TestErrors_Fields(t *testing.T) {
	errs := Errors{
		{Field: "Name", Rule: "required"},
		{Field: "Name", Rule: "min"},
		{Field: "Email", Rule: "email"},
		{Field: "Age", Rule: "min"},
	}
	fields := errs.Fields()
	assert.ElementsMatch(t, []string{"Name", "Email", "Age"}, fields)
}

func TestErrors_FieldsEmpty(t *testing.T) {
	var errs Errors
	fields := errs.Fields()
	assert.Len(t, fields, 0)
}
