// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package redisbloom

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== parseBoolArray =====

func TestParseBoolArray_Valid(t *testing.T) {
	val := []interface{}{int64(1), int64(0), int64(1)}
	result, err := parseBoolArray(val, 3, "MADD")
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false, true}, result)
}

func TestParseBoolArray_AllTrue(t *testing.T) {
	val := []interface{}{int64(1), int64(1), int64(1)}
	result, err := parseBoolArray(val, 3, "MEXISTS")
	require.NoError(t, err)
	assert.Equal(t, []bool{true, true, true}, result)
}

func TestParseBoolArray_AllFalse(t *testing.T) {
	val := []interface{}{int64(0), int64(0)}
	result, err := parseBoolArray(val, 2, "MADD")
	require.NoError(t, err)
	assert.Equal(t, []bool{false, false}, result)
}

func TestParseBoolArray_ShorterThanExpected(t *testing.T) {
	// When the array has fewer elements than expected, the missing slots
	// default to false (zero value).
	val := []interface{}{int64(1)}
	result, err := parseBoolArray(val, 3, "MADD")
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false, false}, result)
}

func TestParseBoolArray_LongerThanExpected(t *testing.T) {
	// Extra elements beyond expected are ignored.
	val := []interface{}{int64(1), int64(0), int64(1), int64(0)}
	result, err := parseBoolArray(val, 2, "MADD")
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false}, result)
}

func TestParseBoolArray_Empty(t *testing.T) {
	val := []interface{}{}
	result, err := parseBoolArray(val, 0, "MADD")
	require.NoError(t, err)
	assert.Equal(t, []bool{}, result)
}

func TestParseBoolArray_WrongType(t *testing.T) {
	// A non-array value should produce an error.
	_, err := parseBoolArray("not-an-array", 3, "MADD")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected MADD response type")
}

func TestParseBoolArray_WrongElementType(t *testing.T) {
	// An array element that is not int64 should produce an error.
	val := []interface{}{int64(1), "not-int", int64(0)}
	_, err := parseBoolArray(val, 3, "MEXISTS")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected MEXISTS element type")
}

func TestParseBoolArray_NilValue(t *testing.T) {
	_, err := parseBoolArray(nil, 3, "MADD")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected MADD response type")
}

// ===== isItemExistsError =====

func TestIsItemExistsError_True(t *testing.T) {
	err := errors.New("ERR item exists")
	assert.True(t, isItemExistsError(err))
}

func TestIsItemExistsError_ExactMatch(t *testing.T) {
	err := errors.New("item exists")
	assert.True(t, isItemExistsError(err))
}

func TestIsItemExistsError_FalseOtherError(t *testing.T) {
	err := errors.New("connection refused")
	assert.False(t, isItemExistsError(err))
}

func TestIsItemExistsError_FalseNil(t *testing.T) {
	assert.False(t, isItemExistsError(nil))
}

func TestIsItemExistsError_FalseEmpty(t *testing.T) {
	err := errors.New("")
	assert.False(t, isItemExistsError(err))
}

// ===== applyOptions / config defaults =====

func TestApplyOptions_Defaults(t *testing.T) {
	cfg := applyOptions()
	assert.Equal(t, "", cfg.key)
	assert.Equal(t, uint64(1000), cfg.capacity)
	assert.Equal(t, 0.001, cfg.errorRate)
	assert.Equal(t, int64(DefaultExpansion), cfg.expansion)
	assert.False(t, cfg.nonScaling)
	assert.False(t, cfg.noCreate)
	assert.Zero(t, cfg.ttl)
}

func TestApplyOptions_NilOptionIgnored(t *testing.T) {
	cfg := applyOptions(WithKey("k"), nil, WithCapacity(500, 0.01))
	assert.Equal(t, "k", cfg.key)
	assert.Equal(t, uint64(500), cfg.capacity)
	assert.Equal(t, 0.01, cfg.errorRate)
}

func TestApplyOptions_AllOptions(t *testing.T) {
	cfg := applyOptions(
		WithKey("mykey"),
		WithCapacity(2000, 0.005),
		WithTTL(10_000_000_000), // 10s in ns
		WithExpansion(3),
		WithNonScaling(),
		WithNoCreate(),
	)
	assert.Equal(t, "mykey", cfg.key)
	assert.Equal(t, uint64(2000), cfg.capacity)
	assert.Equal(t, 0.005, cfg.errorRate)
	assert.Equal(t, int64(3), cfg.expansion)
	assert.True(t, cfg.nonScaling)
	assert.True(t, cfg.noCreate)
}

func TestApplyOptions_WithExpansionInvalid(t *testing.T) {
	// WithExpansion ignores values < 1, keeping the default.
	cfg := applyOptions(WithExpansion(0))
	assert.Equal(t, int64(DefaultExpansion), cfg.expansion)

	cfg2 := applyOptions(WithExpansion(-5))
	assert.Equal(t, int64(DefaultExpansion), cfg2.expansion)
}

func TestApplyOptions_WithErrorRateOnly(t *testing.T) {
	cfg := applyOptions(WithErrorRate(0.02))
	assert.Equal(t, 0.02, cfg.errorRate)
	// capacity keeps default
	assert.Equal(t, uint64(1000), cfg.capacity)
}

func TestApplyOptions_WithExpectedCapacityOnly(t *testing.T) {
	cfg := applyOptions(WithExpectedCapacity(777))
	assert.Equal(t, uint64(777), cfg.capacity)
	// errorRate keeps default
	assert.Equal(t, 0.001, cfg.errorRate)
}

// ===== config.validate =====

func TestConfigValidate_Valid(t *testing.T) {
	cfg := applyOptions(WithKey("k"), WithCapacity(100, 0.01))
	assert.NoError(t, cfg.validate())
}

func TestConfigValidate_EmptyKey(t *testing.T) {
	cfg := applyOptions(WithCapacity(100, 0.01))
	err := cfg.validate()
	assert.Error(t, err)
}

func TestConfigValidate_ZeroCapacity(t *testing.T) {
	cfg := applyOptions(WithKey("k"), WithExpectedCapacity(0))
	err := cfg.validate()
	assert.Error(t, err)
}

func TestConfigValidate_ErrorRateZero(t *testing.T) {
	cfg := applyOptions(WithKey("k"), WithExpectedCapacity(100), WithErrorRate(0))
	err := cfg.validate()
	assert.Error(t, err)
}

func TestConfigValidate_ErrorRateOne(t *testing.T) {
	cfg := applyOptions(WithKey("k"), WithExpectedCapacity(100), WithErrorRate(1))
	err := cfg.validate()
	assert.Error(t, err)
}

func TestConfigValidate_ErrorRateNegative(t *testing.T) {
	cfg := applyOptions(WithKey("k"), WithExpectedCapacity(100), WithErrorRate(-0.1))
	err := cfg.validate()
	assert.Error(t, err)
}
