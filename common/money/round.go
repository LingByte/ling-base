// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package money

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// FromDecimal creates a Money value from a decimal float using RoundHalfUp.
// Prefer Parse for untrusted string input. Large / scientific floats that
// cannot be represented safely return an error (never a silent zero).
func FromDecimal(value float64, currency string) (Money, error) {
	return Round(value, currency, RoundHalfUp)
}

// Round creates a Money value from a decimal float using the given rounding
// mode and the currency's precision.
//
// The value is formatted with strconv.FormatFloat(..., 'f', -1, 64) and then
// parsed with integer arithmetic. Scientific notation, non-finite values, and
// int64 overflows return ErrInvalidAmount / ErrOverflow — never a silent zero.
func Round(value float64, currency string, mode RoundingMode) (Money, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return Money{}, ErrUnknownCurrency
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return Money{}, fmt.Errorf("%w: non-finite float", ErrInvalidAmount)
	}

	precision := CurrencyPrecision(currency)
	scale, err := pow10(precision)
	if err != nil {
		return Money{}, err
	}

	// Force fixed-point formatting. If 'f' somehow still yields an exponent
	// (extreme magnitudes), reject rather than mis-parse.
	s := strconv.FormatFloat(value, 'f', -1, 64)
	if strings.ContainsAny(s, "eE") {
		return Money{}, fmt.Errorf("%w: scientific notation %q", ErrInvalidAmount, s)
	}
	if s == "NaN" || s == "Inf" || s == "-Inf" || s == "+Inf" {
		return Money{}, fmt.Errorf("%w: non-finite float", ErrInvalidAmount)
	}

	negative := false
	if s != "" && s[0] == '-' {
		negative = true
		s = s[1:]
	}

	var intPart, fracPart string
	if dot := strings.Index(s, "."); dot >= 0 {
		intPart = s[:dot]
		fracPart = s[dot+1:]
	} else {
		intPart = s
	}
	if intPart == "" {
		intPart = "0"
	}

	kept := fracPart
	if len(kept) > precision {
		kept = kept[:precision]
	}
	for len(kept) < precision {
		kept += "0"
	}
	rest := ""
	if len(fracPart) > precision {
		rest = fracPart[precision:]
	}

	intVal, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return Money{}, fmt.Errorf("%w: integer part: %v", ErrInvalidAmount, err)
	}
	var fracVal int64
	if kept != "" {
		fracVal, err = strconv.ParseInt(kept, 10, 64)
		if err != nil {
			return Money{}, fmt.Errorf("%w: fractional part: %v", ErrInvalidAmount, err)
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

	if len(rest) > 0 {
		roundUp := false
		switch mode {
		case RoundHalfUp:
			roundUp = rest[0] >= '5'
		case RoundHalfDown:
			if rest[0] > '5' {
				roundUp = true
			} else if rest[0] == '5' {
				roundUp = hasNonZero(rest[1:])
			}
		case RoundHalfEven:
			if rest[0] > '5' {
				roundUp = true
			} else if rest[0] == '5' {
				if hasNonZero(rest[1:]) {
					roundUp = true
				} else if fracVal%2 != 0 {
					roundUp = true
				}
			}
		case RoundDown:
			// never
		case RoundUp:
			roundUp = hasNonZero(rest)
		default:
			roundUp = rest[0] >= '5'
		}
		if roundUp {
			amount, err = addInt64(amount, 1)
			if err != nil {
				return Money{}, err
			}
		}
	}

	if negative {
		amount = -amount
	}
	return Money{amount: amount, currency: currency}, nil
}

func hasNonZero(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '0' {
			return true
		}
	}
	return false
}
