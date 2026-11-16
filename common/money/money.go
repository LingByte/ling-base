// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package money provides precise monetary value handling using fixed-point
// integer arithmetic. Internally amounts are stored as int64 in the smallest
// currency unit (e.g. cents for USD/EUR, fen for CNY, whole yen for JPY) so
// that no floating-point rounding error is introduced during arithmetic.
//
// # Quick start
//
//	m := money.New(199, "USD")          // $1.99
//	m.Decimal()                         // 1.99
//	m.String()                          // "USD 1.99"
//
//	a, _ := money.FromDecimal(10.50, "USD")
//	b := money.New(250, "USD")
//	sum, _ := a.Add(b)                  // $13.00
//
//	parsed, _ := money.Parse("12.34", "USD") // strict: excess decimals error
//
//	alloc, _ := money.New(100, "USD").Allocate([]int{1, 1, 1}) // 34+33+33
//	r, _ := money.Round(2.345, "USD", money.RoundHalfUp)
//
//	_ = money.RegisterCurrency(money.Currency{Code: "POINTS", Precision: 0, Symbol: "P"})
//	rates := money.NewRateTable()
//	_ = rates.Set("USD", "CNY", 7.2)
//	cny, _ := rates.Convert(money.New(100, "USD"), "CNY", money.RoundHalfUp)
package money

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// ──────────────────────────────────────────────
// Errors
// ──────────────────────────────────────────────

var (
	// ErrCurrencyMismatch is returned when an arithmetic / amount comparison
	// is attempted between different currencies.
	ErrCurrencyMismatch = errors.New("money: currency mismatch")

	// ErrDivisionByZero is returned when a division operation has a zero divisor.
	ErrDivisionByZero = errors.New("money: division by zero")

	// ErrInvalidAmount is returned when a string / float cannot be parsed.
	ErrInvalidAmount = errors.New("money: invalid amount")

	// ErrUnknownCurrency is returned when a currency code is empty / invalid
	// for an operation that requires a known currency.
	ErrUnknownCurrency = errors.New("money: unknown currency")

	// ErrNoAllocationParts is returned when Allocate is called with no parts.
	ErrNoAllocationParts = errors.New("money: no allocation parts")

	// ErrNegativeAllocationPart is returned when Allocate is given a negative ratio.
	ErrNegativeAllocationPart = errors.New("money: negative allocation part")
)

// ──────────────────────────────────────────────
// RoundingMode
// ──────────────────────────────────────────────

// RoundingMode controls how a floating-point value is rounded to the smallest
// currency unit when constructing a Money value.
type RoundingMode int

const (
	// RoundHalfUp rounds to nearest, ties away from zero (0.5 -> 1).
	RoundHalfUp RoundingMode = iota
	// RoundHalfDown rounds to nearest, ties toward zero (0.5 -> 0).
	RoundHalfDown
	// RoundHalfEven rounds to nearest, ties to even (banker's rounding).
	RoundHalfEven
	// RoundDown always rounds toward zero (truncation).
	RoundDown
	// RoundUp always rounds away from zero.
	RoundUp
)

// ──────────────────────────────────────────────
// Money
// ──────────────────────────────────────────────

// Money represents a monetary amount stored as a fixed-point integer in the
// smallest unit of the given currency. The zero value has an empty currency;
// use IsValid to detect it. Prefer New / Parse / FromDecimal to construct.
type Money struct {
	amount   int64
	currency string
}

// New creates a Money value from an amount expressed in the smallest currency
// unit. For example New(100, "USD") represents $1.00.
func New(amount int64, currency string) Money {
	return Money{amount: amount, currency: strings.ToUpper(strings.TrimSpace(currency))}
}

// Amount returns the amount in the smallest currency unit.
func (m Money) Amount() int64 { return m.amount }

// Currency returns the currency code.
func (m Money) Currency() string { return m.currency }

// IsValid reports whether m has a non-empty currency code.
func (m Money) IsValid() bool { return m.currency != "" }

// Decimal converts the amount to a float64 in major units (e.g. dollars).
// MinInt64 amounts return 0 (cannot form a safe absolute value in int64).
func (m Money) Decimal() float64 {
	precision := CurrencyPrecision(m.currency)
	scale, err := pow10(precision)
	if err != nil {
		return 0
	}

	negative := m.amount < 0
	abs := m.amount
	if negative {
		if m.amount == math.MinInt64 {
			return 0
		}
		abs = -abs
	}

	intPart := abs / scale
	fracPart := abs % scale

	var s string
	if precision == 0 {
		s = strconv.FormatInt(intPart, 10)
	} else {
		s = strconv.FormatInt(intPart, 10) + "." + leftPad(strconv.FormatInt(fracPart, 10), precision)
	}
	if negative {
		s = "-" + s
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// String formats the money as "CUR X.YY".
func (m Money) String() string {
	if m.currency == "" {
		return m.Format()
	}
	return m.currency + " " + m.Format()
}

// Format formats just the numeric amount using the currency's precision.
// MinInt64 amounts return an empty numeric marker ("0") rather than wrapping.
func (m Money) Format() string {
	precision := CurrencyPrecision(m.currency)
	scale, err := pow10(precision)
	if err != nil {
		return "0"
	}

	negative := m.amount < 0
	abs := m.amount
	if negative {
		if m.amount == math.MinInt64 {
			return "0"
		}
		abs = -abs
	}

	intPart := abs / scale
	fracPart := abs % scale

	var s string
	if precision == 0 {
		s = strconv.FormatInt(intPart, 10)
	} else {
		s = strconv.FormatInt(intPart, 10) + "." + leftPad(strconv.FormatInt(fracPart, 10), precision)
	}
	if negative {
		s = "-" + s
	}
	return s
}

func leftPad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return strings.Repeat("0", n-len(s)) + s
}

// Add returns the sum of m and other. Currencies must match; overflow errors.
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}
	sum, err := addInt64(m.amount, other.amount)
	if err != nil {
		return Money{}, err
	}
	return Money{amount: sum, currency: m.currency}, nil
}

// Sub returns m - other. Currencies must match; overflow errors.
func (m Money) Sub(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}
	diff, err := subInt64(m.amount, other.amount)
	if err != nil {
		return Money{}, err
	}
	return Money{amount: diff, currency: m.currency}, nil
}

// Mul returns m multiplied by n, detecting int64 overflow.
func (m Money) Mul(n int64) (Money, error) {
	prod, err := mulInt64(m.amount, n)
	if err != nil {
		return Money{}, err
	}
	return Money{amount: prod, currency: m.currency}, nil
}

// Div divides m by n (toward zero). Returns ErrDivisionByZero if n is 0.
// MinInt64 / -1 is rejected as overflow.
func (m Money) Div(n int64) (Money, error) {
	if n == 0 {
		return Money{}, ErrDivisionByZero
	}
	if m.amount == math.MinInt64 && n == -1 {
		return Money{}, ErrOverflow
	}
	return Money{amount: m.amount / n, currency: m.currency}, nil
}

// DivMod divides m by n and returns quotient and remainder.
func (m Money) DivMod(n int64) (Money, Money, error) {
	if n == 0 {
		return Money{}, Money{}, ErrDivisionByZero
	}
	if m.amount == math.MinInt64 && n == -1 {
		return Money{}, Money{}, ErrOverflow
	}
	return Money{amount: m.amount / n, currency: m.currency},
		Money{amount: m.amount % n, currency: m.currency}, nil
}

// Neg returns the additive inverse of m. MinInt64 cannot be negated in int64.
func (m Money) Neg() (Money, error) {
	neg, err := negInt64(m.amount)
	if err != nil {
		return Money{}, err
	}
	return Money{amount: neg, currency: m.currency}, nil
}

// Abs returns the absolute value of m. MinInt64 cannot be absolutized in int64.
func (m Money) Abs() (Money, error) {
	if m.amount >= 0 {
		return m, nil
	}
	return m.Neg()
}

// IsZero reports whether m represents zero.
func (m Money) IsZero() bool { return m.amount == 0 }

// IsPositive reports whether m is strictly greater than zero.
func (m Money) IsPositive() bool { return m.amount > 0 }

// IsNegative reports whether m is strictly less than zero.
func (m Money) IsNegative() bool { return m.amount < 0 }

// Equal reports whether m and other have the same currency and amount.
func (m Money) Equal(other Money) bool {
	return m.currency == other.currency && m.amount == other.amount
}

// LessThan compares amounts and requires matching currencies.
func (m Money) LessThan(other Money) (bool, error) {
	cmp, err := m.CompareAmount(other)
	if err != nil {
		return false, err
	}
	return cmp < 0, nil
}

// GreaterThan compares amounts and requires matching currencies.
func (m Money) GreaterThan(other Money) (bool, error) {
	cmp, err := m.CompareAmount(other)
	if err != nil {
		return false, err
	}
	return cmp > 0, nil
}

// CompareAmount returns -1/0/+1 for amount ordering. Currencies must match.
func (m Money) CompareAmount(other Money) (int, error) {
	if m.currency != other.currency {
		return 0, ErrCurrencyMismatch
	}
	switch {
	case m.amount < other.amount:
		return -1, nil
	case m.amount > other.amount:
		return 1, nil
	default:
		return 0, nil
	}
}

// Compare returns -1, 0, or +1. Same-currency values compare by amount;
// different currencies fall back to lexicographic currency ordering so Money
// remains sortable. Prefer CompareAmount / LessThan / GreaterThan for business
// logic — those reject cross-currency comparisons.
func (m Money) Compare(other Money) int {
	if m.currency != other.currency {
		if m.currency < other.currency {
			return -1
		}
		return 1
	}
	switch {
	case m.amount < other.amount:
		return -1
	case m.amount > other.amount:
		return 1
	default:
		return 0
	}
}
