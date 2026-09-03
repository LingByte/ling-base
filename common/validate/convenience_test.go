// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package validate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// ValidateSlice
// ──────────────────────────────────────────────

func TestValidateSlice_MinLength(t *testing.T) {
	errs := ValidateSlice([]string{"a", "ab", "abc"}, "min=2")
	require.NotNil(t, errs)
	assert.Contains(t, errs, 0)
	assert.NotContains(t, errs, 1)
	assert.NotContains(t, errs, 2)
	assert.Contains(t, errs[0].Error(), "min")
}

func TestValidateSlice_IntGtAllValid(t *testing.T) {
	errs := ValidateSlice([]int{1, 2, 3}, "gt=0")
	assert.Nil(t, errs)
}

func TestValidateSlice_RequiredString(t *testing.T) {
	errs := ValidateSlice([]string{"x"}, "required")
	assert.Nil(t, errs)
}

func TestValidateSlice_NilSlice(t *testing.T) {
	errs := ValidateSlice(nil, "min=2")
	assert.Nil(t, errs)
}

func TestValidateSlice_NonSlice(t *testing.T) {
	errs := ValidateSlice(42, "min=2")
	assert.Nil(t, errs)
}

func TestValidateSlice_EmptySlice(t *testing.T) {
	errs := ValidateSlice([]string{}, "min=2")
	assert.Nil(t, errs)
}

func TestValidateSlice_UnknownRule(t *testing.T) {
	errs := ValidateSlice([]string{"a", "b"}, "unknownrule")
	require.NotNil(t, errs)
	assert.Contains(t, errs, 0)
	assert.Contains(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "unknown rule")
	assert.Contains(t, errs[1].Error(), "unknown rule")
}

func TestValidateSlice_PointerToSlice(t *testing.T) {
	s := []string{"a", "abcd"}
	errs := ValidateSlice(&s, "min=3")
	require.NotNil(t, errs)
	assert.Contains(t, errs, 0)
	assert.NotContains(t, errs, 1)
}

func TestValidateSlice_NilPointerToSlice(t *testing.T) {
	var s *[]string
	errs := ValidateSlice(s, "min=2")
	assert.Nil(t, errs)
}

// ──────────────────────────────────────────────
// ValidateMap
// ──────────────────────────────────────────────

func TestValidateMap_IntGt(t *testing.T) {
	errs := ValidateMap(map[string]int{"a": 1, "b": 5}, "gt=3")
	require.NotNil(t, errs)
	assert.Contains(t, errs, "a")
	assert.NotContains(t, errs, "b")
	assert.Contains(t, errs["a"].Error(), "gt")
}

func TestValidateMap_StringMinAllValid(t *testing.T) {
	errs := ValidateMap(map[string]string{"x": "hello"}, "min=3")
	assert.Nil(t, errs)
}

func TestValidateMap_Nil(t *testing.T) {
	errs := ValidateMap(nil, "gt=3")
	assert.Nil(t, errs)
}

func TestValidateMap_NonMap(t *testing.T) {
	errs := ValidateMap(42, "gt=3")
	assert.Nil(t, errs)
}

func TestValidateMap_EmptyMap(t *testing.T) {
	errs := ValidateMap(map[string]int{}, "gt=3")
	assert.Nil(t, errs)
}

func TestValidateMap_PointerToMap(t *testing.T) {
	m := map[string]int{"a": 1, "b": 5}
	errs := ValidateMap(&m, "gt=3")
	require.NotNil(t, errs)
	assert.Contains(t, errs, "a")
	assert.NotContains(t, errs, "b")
}

func TestValidateMap_NilPointerToMap(t *testing.T) {
	var m *map[string]int
	errs := ValidateMap(m, "gt=3")
	assert.Nil(t, errs)
}

// ──────────────────────────────────────────────
// HasRule
// ──────────────────────────────────────────────

func TestHasRule_Required(t *testing.T) {
	assert.True(t, HasRule("required"))
}

func TestHasRule_Email(t *testing.T) {
	assert.True(t, HasRule("email"))
}

func TestHasRule_Nonexistent(t *testing.T) {
	assert.False(t, HasRule("nonexistent"))
}

func TestHasRule_CaseInsensitive(t *testing.T) {
	assert.True(t, HasRule("REQUIRED"))
	assert.True(t, HasRule("Email"))
}

// ──────────────────────────────────────────────
// RegisteredRules
// ──────────────────────────────────────────────

func TestRegisteredRules(t *testing.T) {
	rules := RegisteredRules()
	require.NotNil(t, rules)
	assert.NotEmpty(t, rules)

	// Should contain the core built-in rules.
	assert.Contains(t, rules, "required")
	assert.Contains(t, rules, "email")
	assert.Contains(t, rules, "min")
	assert.Contains(t, rules, "max")

	// There are 30+ built-in rules.
	assert.Greater(t, len(rules), 20)
}

// ──────────────────────────────────────────────
// ResetRules
// ──────────────────────────────────────────────

func TestResetRules_RemovesCustomAndRestoresBuiltin(t *testing.T) {
	// Register a custom rule.
	AddRule("custom", func(value any, param string, parent any) error {
		return nil
	})
	assert.True(t, HasRule("custom"))

	// Reset rules.
	ResetRules()

	// Custom rule should be gone.
	assert.False(t, HasRule("custom"))

	// Built-in rules should be restored.
	assert.True(t, HasRule("required"))
	assert.True(t, HasRule("email"))
}

// ensureRuleNamesLowercase is a sanity check that all registered rule
// names are lowercase (the package lowercases names on registration).
func TestRegisteredRules_AllLowercase(t *testing.T) {
	for _, name := range RegisteredRules() {
		assert.Equal(t, strings.ToLower(name), name, "rule name %q should be lowercase", name)
	}
}

// ──────────────────────────────────────────────
// ValidateEmail
// ──────────────────────────────────────────────

func TestValidateEmail(t *testing.T) {
	assert.True(t, ValidateEmail("user@example.com"))
	assert.True(t, ValidateEmail("alice.bob+tag@sub.domain.org"))
	assert.False(t, ValidateEmail("not-an-email"))
	assert.False(t, ValidateEmail(""))
	assert.False(t, ValidateEmail("no-at-sign"))
}
