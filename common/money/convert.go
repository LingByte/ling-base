// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package money

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

// ErrInvalidRate is returned when an exchange rate is non-positive or non-finite.
var ErrInvalidRate = fmt.Errorf("money: invalid exchange rate")

// RateTable stores pairwise exchange rates in major units.
// A rate R for (FROM -> TO) means: 1 major unit of FROM = R major units of TO.
//
// Inspired by lingpan's usdExchangeRate model: keep rates explicit, convert
// through Round(..., mode), never silently mix currencies in arithmetic.
type RateTable struct {
	mu    sync.RWMutex
	rates map[string]map[string]float64 // from -> to -> rate
}

// NewRateTable creates an empty rate table.
func NewRateTable() *RateTable {
	return &RateTable{rates: make(map[string]map[string]float64)}
}

// Set stores the major-unit rate from -> to. Rate must be finite and > 0.
func (t *RateTable) Set(from, to string, rate float64) error {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	if from == "" || to == "" {
		return ErrUnknownCurrency
	}
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate <= 0 {
		return ErrInvalidRate
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.rates[from] == nil {
		t.rates[from] = make(map[string]float64)
	}
	t.rates[from][to] = rate
	return nil
}

// Get returns the stored rate for from -> to.
func (t *RateTable) Get(from, to string) (float64, bool) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	t.mu.RLock()
	defer t.mu.RUnlock()
	if m := t.rates[from]; m != nil {
		r, ok := m[to]
		return r, ok
	}
	return 0, false
}

// Convert converts m into toCurrency using a previously Set rate and rounding mode.
func (t *RateTable) Convert(m Money, toCurrency string, mode RoundingMode) (Money, error) {
	if !m.IsValid() {
		return Money{}, ErrUnknownCurrency
	}
	toCurrency = strings.ToUpper(strings.TrimSpace(toCurrency))
	if toCurrency == "" {
		return Money{}, ErrUnknownCurrency
	}
	if m.currency == toCurrency {
		return m, nil
	}
	rate, ok := t.Get(m.currency, toCurrency)
	if !ok {
		return Money{}, fmt.Errorf("%w: no rate %s -> %s", ErrInvalidRate, m.currency, toCurrency)
	}
	return m.Convert(toCurrency, rate, mode)
}

// Convert converts m to toCurrency using a major-unit exchange rate.
// Example: USD->CNY rate 7 means $1.00 becomes ¥7.00.
//
// Prefer ConvertRatio for payment-grade FX: float rates reintroduce
// approximation via Decimal()*rate (acceptable for display, not settlement).
func (m Money) Convert(toCurrency string, majorUnitRate float64, mode RoundingMode) (Money, error) {
	if !m.IsValid() {
		return Money{}, ErrUnknownCurrency
	}
	toCurrency = strings.ToUpper(strings.TrimSpace(toCurrency))
	if toCurrency == "" {
		return Money{}, ErrUnknownCurrency
	}
	if math.IsNaN(majorUnitRate) || math.IsInf(majorUnitRate, 0) || majorUnitRate <= 0 {
		return Money{}, ErrInvalidRate
	}
	if m.currency == toCurrency {
		return m, nil
	}
	// Display-oriented path: major * rate, then Round into target precision.
	major := m.Decimal()
	if major == 0 && m.amount != 0 {
		return Money{}, ErrOverflow
	}
	return Round(major*majorUnitRate, toCurrency, mode)
}

// ConvertRatio converts using an exact rational major-unit rate num/den.
// Example: 72/10 means 1 USD = 7.2 CNY. Settlement-safe (int64 only + Round).
func (m Money) ConvertRatio(toCurrency string, num, den int64, mode RoundingMode) (Money, error) {
	if !m.IsValid() {
		return Money{}, ErrUnknownCurrency
	}
	toCurrency = strings.ToUpper(strings.TrimSpace(toCurrency))
	if toCurrency == "" {
		return Money{}, ErrUnknownCurrency
	}
	if den <= 0 || num <= 0 {
		return Money{}, ErrInvalidRate
	}
	if m.currency == toCurrency {
		return m, nil
	}

	fromPrec := CurrencyPrecision(m.currency)
	toPrec := CurrencyPrecision(toCurrency)
	fromScale, err := pow10(fromPrec)
	if err != nil {
		return Money{}, err
	}
	toScale, err := pow10(toPrec)
	if err != nil {
		return Money{}, err
	}

	// target_minor ≈ amount * (num/den) * (toScale/fromScale)
	// = amount * num * toScale / (den * fromScale)
	neg := m.amount < 0
	absAmount := m.amount
	if neg {
		var nerr error
		absAmount, nerr = negInt64(m.amount)
		if nerr != nil {
			return Money{}, nerr
		}
	}

	step1, err := mulInt64(absAmount, num)
	if err != nil {
		return Money{}, err
	}
	step2, err := mulInt64(step1, toScale)
	if err != nil {
		return Money{}, err
	}
	denom, err := mulInt64(den, fromScale)
	if err != nil {
		return Money{}, err
	}

	quot := step2 / denom
	rem := step2 % denom
	// Build a temporary major float only for the fractional remainder via Round?
	// Better: apply rounding mode on (quot + rem/denom) in integer space.
	rounded, err := roundQuotient(quot, rem, denom, mode)
	if err != nil {
		return Money{}, err
	}
	if neg {
		rounded, err = negInt64(rounded)
		if err != nil {
			return Money{}, err
		}
	}
	return Money{amount: rounded, currency: toCurrency}, nil
}

// roundQuotient applies RoundingMode to quot + rem/denom (rem>=0, denom>0).
func roundQuotient(quot, rem, denom int64, mode RoundingMode) (int64, error) {
	if rem == 0 {
		return quot, nil
	}
	// Compare 2*rem ? denom for half decisions without float.
	twice := rem * 2 // rem < denom so 2*rem fits if denom <= MaxInt64/2; check:
	if rem > math.MaxInt64/2 {
		// Extremely large denom path: fall back to float comparison once.
		frac := float64(rem) / float64(denom)
		return applyFloatFrac(quot, frac, mode)
	}
	roundUp := false
	switch mode {
	case RoundHalfUp:
		roundUp = twice >= denom
	case RoundHalfDown:
		roundUp = twice > denom
	case RoundHalfEven:
		if twice > denom {
			roundUp = true
		} else if twice == denom {
			roundUp = quot%2 != 0
		}
	case RoundDown:
		roundUp = false
	case RoundUp:
		roundUp = rem != 0
	default:
		roundUp = twice >= denom
	}
	if !roundUp {
		return quot, nil
	}
	return addInt64(quot, 1)
}

func applyFloatFrac(quot int64, frac float64, mode RoundingMode) (int64, error) {
	roundUp := false
	switch mode {
	case RoundHalfUp:
		roundUp = frac >= 0.5
	case RoundHalfDown:
		roundUp = frac > 0.5
	case RoundHalfEven:
		if frac > 0.5 {
			roundUp = true
		} else if frac == 0.5 {
			roundUp = quot%2 != 0
		}
	case RoundDown:
	case RoundUp:
		roundUp = frac > 0
	default:
		roundUp = frac >= 0.5
	}
	if !roundUp {
		return quot, nil
	}
	return addInt64(quot, 1)
}
