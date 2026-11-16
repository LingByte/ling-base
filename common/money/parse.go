// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package money

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse parses a decimal string into Money using the currency's precision.
// Excess fractional digits are rejected with ErrInvalidAmount (strict).
// Use ParseTrunc when truncation is intentional.
func Parse(s, currency string) (Money, error) {
	return parseDecimal(s, currency, false)
}

// ParseTrunc is like Parse but truncates excess fractional digits instead of
// returning an error.
func ParseTrunc(s, currency string) (Money, error) {
	return parseDecimal(s, currency, true)
}

func parseDecimal(s, currency string, truncate bool) (Money, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return Money{}, ErrUnknownCurrency
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return Money{}, ErrInvalidAmount
	}

	negative := false
	if s[0] == '-' {
		negative = true
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}
	if s == "" {
		return Money{}, ErrInvalidAmount
	}

	dot := strings.Index(s, ".")
	precision := CurrencyPrecision(currency)

	var intPart, fracPart string
	if dot == -1 {
		intPart = s
		fracPart = ""
	} else {
		intPart = s[:dot]
		fracPart = s[dot+1:]
	}
	if intPart == "" {
		intPart = "0"
	}

	// Reject lone "."
	if intPart == "0" && fracPart == "" && dot != -1 {
		return Money{}, ErrInvalidAmount
	}

	for _, c := range intPart {
		if c < '0' || c > '9' {
			return Money{}, ErrInvalidAmount
		}
	}
	for _, c := range fracPart {
		if c < '0' || c > '9' {
			return Money{}, ErrInvalidAmount
		}
	}

	if len(fracPart) > precision {
		if !truncate {
			return Money{}, fmt.Errorf("%w: too many decimal places (got %d, max %d)", ErrInvalidAmount, len(fracPart), precision)
		}
		fracPart = fracPart[:precision]
	} else {
		fracPart += strings.Repeat("0", precision-len(fracPart))
	}

	scale, err := pow10(precision)
	if err != nil {
		return Money{}, err
	}

	intVal, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return Money{}, ErrInvalidAmount
	}
	var fracVal int64
	if fracPart != "" {
		fracVal, err = strconv.ParseInt(fracPart, 10, 64)
		if err != nil {
			return Money{}, ErrInvalidAmount
		}
	}

	scaledInt, err := mulInt64(intVal, scale)
	if err != nil {
		return Money{}, err
	}
	amount, err := addInt64(scaledInt, fracVal)
	if err != nil {
		return Money{}, err
	}
	if negative {
		amount = -amount
	}
	return Money{amount: amount, currency: currency}, nil
}
