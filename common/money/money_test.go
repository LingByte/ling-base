// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package money

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Construction
// ──────────────────────────────────────────────

func TestNew(t *testing.T) {
	m := New(199, "USD")
	assert.Equal(t, int64(199), m.Amount())
	assert.Equal(t, "USD", m.Currency())
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
		{0.005, "USD", 1}, // round half up
		{100, "JPY", 100},
		{1.234, "JPY", 1}, // 0 decimals
		{1.5, "JPY", 2},
		{-1.99, "USD", -199},
		{0, "USD", 0},
		{1.999, "BHD", 1999}, // 3 decimals
	}
	for _, tt := range tests {
		m := FromDecimal(tt.value, tt.currency)
		assert.Equal(t, tt.want, m.Amount(), "FromDecimal(%v, %s)", tt.value, tt.currency)
	}
}

func TestRound(t *testing.T) {
	// Use 0.125 (1/8, exactly representable) so 0.125*100 = 12.5 is a true tie.
	// RoundHalfUp: ties away from zero
	assert.Equal(t, int64(13), Round(0.125, "USD", RoundHalfUp).Amount())
	assert.Equal(t, int64(12), Round(0.124, "USD", RoundHalfUp).Amount())
	assert.Equal(t, int64(-13), Round(-0.125, "USD", RoundHalfUp).Amount())

	// RoundHalfDown: ties toward zero
	assert.Equal(t, int64(12), Round(0.125, "USD", RoundHalfDown).Amount())
	assert.Equal(t, int64(13), Round(0.126, "USD", RoundHalfDown).Amount())
	assert.Equal(t, int64(-12), Round(-0.125, "USD", RoundHalfDown).Amount())

	// RoundHalfEven: ties to even (banker's rounding)
	assert.Equal(t, int64(12), Round(0.125, "USD", RoundHalfEven).Amount())  // tie -> 12 (even)
	assert.Equal(t, int64(38), Round(0.375, "USD", RoundHalfEven).Amount())  // tie -> 38 (even)
	assert.Equal(t, int64(-12), Round(-0.125, "USD", RoundHalfEven).Amount())

	// RoundDown (truncation toward zero)
	assert.Equal(t, int64(12), Round(0.129, "USD", RoundDown).Amount())
	assert.Equal(t, int64(-12), Round(-0.129, "USD", RoundDown).Amount())

	// RoundUp (away from zero)
	assert.Equal(t, int64(13), Round(0.121, "USD", RoundUp).Amount())
	assert.Equal(t, int64(-13), Round(-0.121, "USD", RoundUp).Amount())

	// Unknown rounding mode defaults to half up
	assert.Equal(t, int64(13), Round(0.125, "USD", RoundingMode(99)).Amount())
}

func TestParse(t *testing.T) {
	tests := []struct {
		s        string
		currency string
		want     int64
		wantErr  bool
	}{
		{"1.99", "USD", 199, false},
		{"1.9", "USD", 190, false},
		{"1", "USD", 100, false},
		{"0.01", "USD", 1, false},
		{"-1.99", "USD", -199, false},
		{"+1.50", "USD", 150, false},
		{"  1.25  ", "USD", 125, false},
		{"100", "JPY", 100, false},
		{"100.50", "JPY", 100, false}, // truncate extra decimals
		{"1.234", "BHD", 1234, false}, // 3 decimals
		{"1.2", "BHD", 1200, false},
		{"", "USD", 0, true},
		{"  ", "USD", 0, true},
		{"abc", "USD", 0, true},
		{"1.2.3", "USD", 0, true},
		{"1.xx", "USD", 0, true},
		{"-1.2x", "USD", 0, true},
		{".99", "USD", 99, false},
		{"-.", "USD", 0, true},
		{"99999999999999999999999999", "USD", 0, true}, // overflow
	}
	for _, tt := range tests {
		m, err := Parse(tt.s, tt.currency)
		if tt.wantErr {
			assert.Error(t, err, "Parse(%q, %s) should error", tt.s, tt.currency)
			continue
		}
		require.NoError(t, err, "Parse(%q, %s)", tt.s, tt.currency)
		assert.Equal(t, tt.want, m.Amount(), "Parse(%q, %s)", tt.s, tt.currency)
	}
}

// ──────────────────────────────────────────────
// Accessors & formatting
// ──────────────────────────────────────────────

func TestDecimal(t *testing.T) {
	assert.Equal(t, 1.99, New(199, "USD").Decimal())
	assert.Equal(t, 100.0, New(100, "JPY").Decimal())
	assert.Equal(t, 1.234, New(1234, "BHD").Decimal())
	assert.Equal(t, -1.50, New(-150, "USD").Decimal())
}

func TestString(t *testing.T) {
	assert.Equal(t, "USD 1.99", New(199, "USD").String())
	assert.Equal(t, "USD -1.99", New(-199, "USD").String())
	assert.Equal(t, "JPY 100", New(100, "JPY").String())
	assert.Equal(t, "USD 0.00", New(0, "USD").String())
	assert.Equal(t, "BHD 1.234", New(1234, "BHD").String())
}

func TestFormat(t *testing.T) {
	assert.Equal(t, "1.99", New(199, "USD").Format())
	assert.Equal(t, "0.05", New(5, "USD").Format())
	assert.Equal(t, "100", New(100, "JPY").Format())
	assert.Equal(t, "-1.50", New(-150, "USD").Format())
	assert.Equal(t, "0.00", New(0, "USD").Format())
}

// ──────────────────────────────────────────────
// Arithmetic
// ──────────────────────────────────────────────

func TestAdd(t *testing.T) {
	a := New(100, "USD")
	b := New(200, "USD")
	sum, err := a.Add(b)
	require.NoError(t, err)
	assert.Equal(t, int64(300), sum.Amount())

	_, err = a.Add(New(100, "EUR"))
	assert.ErrorIs(t, err, ErrCurrencyMismatch)
}

func TestSub(t *testing.T) {
	a := New(300, "USD")
	b := New(100, "USD")
	diff, err := a.Sub(b)
	require.NoError(t, err)
	assert.Equal(t, int64(200), diff.Amount())

	_, err = a.Sub(New(100, "EUR"))
	assert.ErrorIs(t, err, ErrCurrencyMismatch)
}

func TestMul(t *testing.T) {
	assert.Equal(t, int64(300), New(100, "USD").Mul(3).Amount())
	assert.Equal(t, int64(-200), New(100, "USD").Mul(-2).Amount())
	assert.Equal(t, int64(0), New(100, "USD").Mul(0).Amount())
}

func TestDiv(t *testing.T) {
	q, err := New(100, "USD").Div(3)
	require.NoError(t, err)
	assert.Equal(t, int64(33), q.Amount())

	_, err = New(100, "USD").Div(0)
	assert.ErrorIs(t, err, ErrDivisionByZero)
}

func TestDivMod(t *testing.T) {
	q, r, err := New(100, "USD").DivMod(3)
	require.NoError(t, err)
	assert.Equal(t, int64(33), q.Amount())
	assert.Equal(t, int64(1), r.Amount())

	_, _, err = New(100, "USD").DivMod(0)
	assert.ErrorIs(t, err, ErrDivisionByZero)
}

func TestNeg(t *testing.T) {
	assert.Equal(t, int64(-100), New(100, "USD").Neg().Amount())
	assert.Equal(t, int64(100), New(-100, "USD").Neg().Amount())
	assert.Equal(t, int64(0), New(0, "USD").Neg().Amount())
}

func TestAbs(t *testing.T) {
	assert.Equal(t, int64(100), New(-100, "USD").Abs().Amount())
	assert.Equal(t, int64(100), New(100, "USD").Abs().Amount())
	assert.Equal(t, int64(0), New(0, "USD").Abs().Amount())
}

// ──────────────────────────────────────────────
// Predicates & comparison
// ──────────────────────────────────────────────

func TestIsZero(t *testing.T) {
	assert.True(t, New(0, "USD").IsZero())
	assert.False(t, New(1, "USD").IsZero())
	assert.False(t, New(-1, "USD").IsZero())
}

func TestIsPositive(t *testing.T) {
	assert.True(t, New(1, "USD").IsPositive())
	assert.False(t, New(0, "USD").IsPositive())
	assert.False(t, New(-1, "USD").IsPositive())
}

func TestIsNegative(t *testing.T) {
	assert.True(t, New(-1, "USD").IsNegative())
	assert.False(t, New(0, "USD").IsNegative())
	assert.False(t, New(1, "USD").IsNegative())
}

func TestEqual(t *testing.T) {
	assert.True(t, New(100, "USD").Equal(New(100, "USD")))
	assert.False(t, New(100, "USD").Equal(New(101, "USD")))
	assert.False(t, New(100, "USD").Equal(New(100, "EUR")))
}

func TestCompare(t *testing.T) {
	assert.Equal(t, -1, New(100, "USD").Compare(New(200, "USD")))
	assert.Equal(t, 0, New(100, "USD").Compare(New(100, "USD")))
	assert.Equal(t, 1, New(200, "USD").Compare(New(100, "USD")))
	// different currencies: lexicographic on currency code
	assert.Equal(t, -1, New(100, "EUR").Compare(New(100, "USD")))
	assert.Equal(t, 1, New(100, "USD").Compare(New(100, "EUR")))
}

func TestLessThanGreaterThan(t *testing.T) {
	assert.True(t, New(100, "USD").LessThan(New(200, "USD")))
	assert.False(t, New(200, "USD").LessThan(New(100, "USD")))
	assert.True(t, New(200, "USD").GreaterThan(New(100, "USD")))
	assert.False(t, New(100, "USD").GreaterThan(New(200, "USD")))
	// different currency
	assert.True(t, New(100, "EUR").LessThan(New(100, "USD")))
	assert.True(t, New(100, "USD").GreaterThan(New(100, "EUR")))
}

// ──────────────────────────────────────────────
// Allocate
// ──────────────────────────────────────────────

func TestAllocate(t *testing.T) {
	// 100 cents across [1,1,1] => 34,33,33
	parts, err := New(100, "USD").Allocate([]int{1, 1, 1})
	require.NoError(t, err)
	require.Len(t, parts, 3)
	assert.Equal(t, int64(34), parts[0].Amount())
	assert.Equal(t, int64(33), parts[1].Amount())
	assert.Equal(t, int64(33), parts[2].Amount())
	// sum equals original
	sum := int64(0)
	for _, p := range parts {
		sum += p.Amount()
	}
	assert.Equal(t, int64(100), sum)

	// 100 cents across [1,1,2] => 25,25,50
	parts, err = New(100, "USD").Allocate([]int{1, 1, 2})
	require.NoError(t, err)
	sum = int64(0)
	for _, p := range parts {
		sum += p.Amount()
	}
	assert.Equal(t, int64(100), sum)

	// single part
	parts, err = New(99, "USD").Allocate([]int{5})
	require.NoError(t, err)
	assert.Equal(t, int64(99), parts[0].Amount())

	// uneven: 10 cents across [1,1,1] => 4,3,3
	parts, err = New(10, "USD").Allocate([]int{1, 1, 1})
	require.NoError(t, err)
	sum = int64(0)
	for _, p := range parts {
		sum += p.Amount()
	}
	assert.Equal(t, int64(10), sum)

	// negative amount allocation
	parts, err = New(-100, "USD").Allocate([]int{1, 1, 1})
	require.NoError(t, err)
	sum = int64(0)
	for _, p := range parts {
		sum += p.Amount()
	}
	assert.Equal(t, int64(-100), sum)

	// zero amount
	parts, err = New(0, "USD").Allocate([]int{1, 2, 3})
	require.NoError(t, err)
	for _, p := range parts {
		assert.Equal(t, int64(0), p.Amount())
	}

	// errors
	_, err = New(100, "USD").Allocate([]int{})
	assert.ErrorIs(t, err, ErrNoAllocationParts)

	_, err = New(100, "USD").Allocate(nil)
	assert.ErrorIs(t, err, ErrNoAllocationParts)

	_, err = New(100, "USD").Allocate([]int{0, 0, 0})
	assert.ErrorIs(t, err, ErrNoAllocationParts)

	_, err = New(100, "USD").Allocate([]int{1, -1, 1})
	assert.ErrorIs(t, err, ErrNegativeAllocationPart)
}

func TestAllocateLargeRemainder(t *testing.T) {
	// 7 across [1,1,1] => 3,2,2
	parts, err := New(7, "USD").Allocate([]int{1, 1, 1})
	require.NoError(t, err)
	sum := int64(0)
	for _, p := range parts {
		sum += p.Amount()
	}
	assert.Equal(t, int64(7), sum)
}

// ──────────────────────────────────────────────
// Currency metadata
// ──────────────────────────────────────────────

func TestCurrencyPrecision(t *testing.T) {
	assert.Equal(t, 2, CurrencyPrecision("USD"))
	assert.Equal(t, 0, CurrencyPrecision("JPY"))
	assert.Equal(t, 3, CurrencyPrecision("BHD"))
	assert.Equal(t, 2, CurrencyPrecision("UNKNOWN")) // default
}

func TestCurrencySymbol(t *testing.T) {
	assert.Equal(t, "$", CurrencySymbol("USD"))
	assert.Equal(t, "€", CurrencySymbol("EUR"))
	assert.Equal(t, "¥", CurrencySymbol("CNY"))
	assert.Equal(t, "¥", CurrencySymbol("JPY"))
	assert.Equal(t, "UNKNOWN", CurrencySymbol("UNKNOWN"))
}

// ──────────────────────────────────────────────
// Rounding helpers (internal)
// ──────────────────────────────────────────────

func TestRoundHalfDownInternal(t *testing.T) {
	assert.Equal(t, 2.0, roundHalfDown(2.5))
	assert.Equal(t, 3.0, roundHalfDown(2.6))
	assert.Equal(t, -2.0, roundHalfDown(-2.5))
	assert.Equal(t, -3.0, roundHalfDown(-2.6))
}

func TestRoundHalfEvenInternal(t *testing.T) {
	assert.Equal(t, 2.0, roundHalfEven(2.5))
	assert.Equal(t, 4.0, roundHalfEven(3.5))
	assert.Equal(t, -2.0, roundHalfEven(-2.5))
	assert.Equal(t, -4.0, roundHalfEven(-3.5))
	assert.Equal(t, 2.0, roundHalfEven(2.4))
	assert.Equal(t, 3.0, roundHalfEven(2.6))
}

func TestRoundUpInternal(t *testing.T) {
	assert.Equal(t, 3.0, roundUp(2.1))
	assert.Equal(t, -3.0, roundUp(-2.1))
	assert.Equal(t, 2.0, roundUp(2.0))
}

func TestPow10(t *testing.T) {
	assert.Equal(t, int64(1), pow10(0))
	assert.Equal(t, int64(10), pow10(1))
	assert.Equal(t, int64(100), pow10(2))
	assert.Equal(t, int64(1000), pow10(3))
}

func TestLeftPad(t *testing.T) {
	assert.Equal(t, "005", leftPad("5", 3))
	assert.Equal(t, "123", leftPad("123", 3))
	assert.Equal(t, "00123", leftPad("123", 5))
}

// Ensure Decimal matches math for a known value to keep the math import used.
func TestDecimalFloatAccuracy(t *testing.T) {
	m := New(123456, "USD")
	assert.InDelta(t, 1234.56, m.Decimal(), math.Pow(10, -6))
}
