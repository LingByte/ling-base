// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package money

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	m := New(199, "usd")
	assert.Equal(t, int64(199), m.Amount())
	assert.Equal(t, "USD", m.Currency())
	assert.True(t, m.IsValid())
	assert.False(t, Money{}.IsValid())
}

func TestFromDecimal(t *testing.T) {
	tests := []struct {
		value    float64
		currency string
		want     int64
	}{
		{1.99, "USD", 199},
		{10.50, "USD", 1050},
		{0.01, "USD", 1},
		{0.005, "USD", 1},
		{100, "JPY", 100},
		{1.234, "JPY", 1},
		{1.5, "JPY", 2},
		{-1.99, "USD", -199},
		{0, "USD", 0},
		{1.999, "BHD", 1999},
	}
	for _, tt := range tests {
		m, err := FromDecimal(tt.value, tt.currency)
		require.NoError(t, err, "FromDecimal(%v, %s)", tt.value, tt.currency)
		assert.Equal(t, tt.want, m.Amount(), "FromDecimal(%v, %s)", tt.value, tt.currency)
	}
}

func TestRound(t *testing.T) {
	must := func(v float64, mode RoundingMode) int64 {
		t.Helper()
		m, err := Round(v, "USD", mode)
		require.NoError(t, err)
		return m.Amount()
	}
	assert.Equal(t, int64(13), must(0.125, RoundHalfUp))
	assert.Equal(t, int64(12), must(0.124, RoundHalfUp))
	assert.Equal(t, int64(-13), must(-0.125, RoundHalfUp))

	assert.Equal(t, int64(12), must(0.125, RoundHalfDown))
	assert.Equal(t, int64(13), must(0.126, RoundHalfDown))
	assert.Equal(t, int64(-12), must(-0.125, RoundHalfDown))

	assert.Equal(t, int64(12), must(0.125, RoundHalfEven))
	assert.Equal(t, int64(38), must(0.375, RoundHalfEven))
	assert.Equal(t, int64(-12), must(-0.125, RoundHalfEven))

	assert.Equal(t, int64(12), must(0.129, RoundDown))
	assert.Equal(t, int64(-12), must(-0.129, RoundDown))

	assert.Equal(t, int64(13), must(0.121, RoundUp))
	assert.Equal(t, int64(-13), must(-0.121, RoundUp))

	assert.Equal(t, int64(13), must(0.125, RoundingMode(99)))
}

func TestRound_Errors(t *testing.T) {
	_, err := Round(math.NaN(), "USD", RoundHalfUp)
	assert.ErrorIs(t, err, ErrInvalidAmount)
	_, err = Round(math.Inf(1), "USD", RoundHalfUp)
	assert.ErrorIs(t, err, ErrInvalidAmount)
	_, err = Round(1e30, "USD", RoundHalfUp)
	assert.Error(t, err)
	_, err = Round(1.0, "", RoundHalfUp)
	assert.ErrorIs(t, err, ErrUnknownCurrency)
}

func TestParseStrictAndTrunc(t *testing.T) {
	m, err := Parse("1.99", "USD")
	require.NoError(t, err)
	assert.Equal(t, int64(199), m.Amount())

	_, err = Parse("1.2345", "USD")
	assert.ErrorIs(t, err, ErrInvalidAmount)

	m, err = ParseTrunc("1.2345", "USD")
	require.NoError(t, err)
	assert.Equal(t, int64(123), m.Amount())

	m, err = ParseTrunc("100.50", "JPY")
	require.NoError(t, err)
	assert.Equal(t, int64(100), m.Amount())

	_, err = Parse("100.50", "JPY")
	assert.ErrorIs(t, err, ErrInvalidAmount)

	_, err = Parse("", "USD")
	assert.ErrorIs(t, err, ErrInvalidAmount)
	_, err = Parse("abc", "USD")
	assert.ErrorIs(t, err, ErrInvalidAmount)
}

func TestArithmetic(t *testing.T) {
	sum, err := New(100, "USD").Add(New(200, "USD"))
	require.NoError(t, err)
	assert.Equal(t, int64(300), sum.Amount())
	_, err = New(100, "USD").Add(New(100, "EUR"))
	assert.ErrorIs(t, err, ErrCurrencyMismatch)

	diff, err := New(300, "USD").Sub(New(100, "USD"))
	require.NoError(t, err)
	assert.Equal(t, int64(200), diff.Amount())

	prod, err := New(100, "USD").Mul(3)
	require.NoError(t, err)
	assert.Equal(t, int64(300), prod.Amount())

	q, err := New(100, "USD").Div(3)
	require.NoError(t, err)
	assert.Equal(t, int64(33), q.Amount())
	_, err = New(100, "USD").Div(0)
	assert.ErrorIs(t, err, ErrDivisionByZero)
}

func TestCompareAmountAndOrdering(t *testing.T) {
	cmp, err := New(100, "USD").CompareAmount(New(200, "USD"))
	require.NoError(t, err)
	assert.Equal(t, -1, cmp)

	_, err = New(100, "USD").CompareAmount(New(100, "EUR"))
	assert.ErrorIs(t, err, ErrCurrencyMismatch)

	lt, err := New(100, "USD").LessThan(New(200, "USD"))
	require.NoError(t, err)
	assert.True(t, lt)

	_, err = New(100, "USD").LessThan(New(100, "EUR"))
	assert.ErrorIs(t, err, ErrCurrencyMismatch)

	// Compare still sorts cross-currency lexicographically.
	assert.Equal(t, -1, New(100, "EUR").Compare(New(100, "USD")))
}

func TestAllocate(t *testing.T) {
	parts, err := New(100, "USD").Allocate([]int{1, 1, 1})
	require.NoError(t, err)
	require.Len(t, parts, 3)
	assert.Equal(t, int64(34), parts[0].Amount())
	assert.Equal(t, int64(33), parts[1].Amount())
	assert.Equal(t, int64(33), parts[2].Amount())

	sum := int64(0)
	for _, p := range parts {
		sum += p.Amount()
	}
	assert.Equal(t, int64(100), sum)

	parts, err = New(7, "USD").Allocate([]int{1, 1, 1})
	require.NoError(t, err)
	sum = 0
	for _, p := range parts {
		sum += p.Amount()
	}
	assert.Equal(t, int64(7), sum)

	_, err = New(100, "USD").Allocate([]int{})
	assert.ErrorIs(t, err, ErrNoAllocationParts)
	_, err = New(100, "USD").Allocate([]int{1, -1})
	assert.ErrorIs(t, err, ErrNegativeAllocationPart)
}

func TestRegisterCurrencyAndConvert(t *testing.T) {
	require.NoError(t, RegisterCurrency(Currency{Code: "POINTS", Precision: 0, Symbol: "P"}))
	t.Cleanup(func() { UnregisterCurrency("POINTS") })

	assert.True(t, IsKnownCurrency("POINTS"))
	assert.Equal(t, 0, CurrencyPrecision("POINTS"))
	assert.Equal(t, "P", CurrencySymbol("POINTS"))

	m, err := Parse("10", "POINTS")
	require.NoError(t, err)
	assert.Equal(t, int64(10), m.Amount())

	rates := NewRateTable()
	require.NoError(t, rates.Set("USD", "CNY", 7))
	cny, err := rates.Convert(New(100, "USD"), "CNY", RoundHalfUp) // $1.00 -> ¥7.00
	require.NoError(t, err)
	assert.Equal(t, "CNY", cny.Currency())
	assert.Equal(t, int64(700), cny.Amount())

	_, err = rates.Convert(New(100, "USD"), "EUR", RoundHalfUp)
	assert.Error(t, err)
	_, err = New(100, "USD").Convert("CNY", -1, RoundHalfUp)
	assert.ErrorIs(t, err, ErrInvalidRate)
}

func TestFormatAndDecimal(t *testing.T) {
	assert.Equal(t, "USD 1.99", New(199, "USD").String())
	assert.Equal(t, "1.99", New(199, "USD").Format())
	assert.Equal(t, 1.99, New(199, "USD").Decimal())
	assert.Equal(t, "100", New(100, "JPY").Format())
}

func TestPow10Bounds(t *testing.T) {
	v, err := pow10(0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), v)
	_, err = pow10(19)
	assert.ErrorIs(t, err, ErrOverflow)
	_, err = pow10(-1)
	assert.ErrorIs(t, err, ErrOverflow)
}

func TestOverflowMulAdd(t *testing.T) {
	_, err := New(math.MaxInt64, "USD").Add(New(1, "USD"))
	assert.ErrorIs(t, err, ErrOverflow)
	_, err = New(math.MaxInt64/2+1, "USD").Mul(3)
	assert.ErrorIs(t, err, ErrOverflow)
}

func TestFromDecimal_LargeAmount(t *testing.T) {
	m, err := FromDecimal(123456789.99, "USD")
	require.NoError(t, err)
	assert.Equal(t, int64(12345678999), m.Amount())
}

func TestRound_HalfEvenEdges(t *testing.T) {
	m, err := Round(0.1255, "USD", RoundHalfEven)
	require.NoError(t, err)
	assert.Equal(t, int64(13), m.Amount())
	m, err = Round(0.126, "USD", RoundHalfEven)
	require.NoError(t, err)
	assert.Equal(t, int64(13), m.Amount())
	m, err = Round(0.1255, "USD", RoundHalfDown)
	require.NoError(t, err)
	assert.Equal(t, int64(13), m.Amount())
}

func TestHasNonZero(t *testing.T) {
	assert.False(t, hasNonZero(""))
	assert.False(t, hasNonZero("000"))
	assert.True(t, hasNonZero("001"))
}
