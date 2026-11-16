// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package money

import (
	"fmt"
	"strings"
	"sync"
)

// Currency describes a currency's metadata.
type Currency struct {
	Code      string // ISO 4217 or custom code (stored uppercased)
	Precision int    // decimal places of the smallest unit (0..18)
	Symbol    string // display symbol; empty falls back to Code
}

var (
	currencyMu sync.RWMutex

	// builtinPrecision maps ISO 4217 currency codes to decimal places.
	builtinPrecision = map[string]int{
		"USD": 2, "EUR": 2, "GBP": 2, "CNY": 2, "JPY": 0, "KRW": 0,
		"CHF": 2, "CAD": 2, "AUD": 2, "NZD": 2, "HKD": 2, "SGD": 2,
		"INR": 2, "RUB": 2, "BRL": 2, "MXN": 2, "ZAR": 2, "SEK": 2,
		"NOK": 2, "DKK": 2, "PLN": 2, "TRY": 2, "THB": 2, "PHP": 2,
		"MYR": 2, "IDR": 2, "VND": 0, "TWD": 2, "ILS": 2, "AED": 2,
		"SAR": 2, "CLP": 0, "ISK": 0, "UGX": 0, "RWF": 0, "BIF": 0,
		"DJF": 0, "GNF": 0, "KMF": 0, "PYG": 0, "VUV": 0, "XAF": 0,
		"XOF": 0, "XPF": 0, "BHD": 3, "IQD": 3, "JOD": 3, "KWD": 3,
		"LYD": 3, "OMR": 3, "TND": 3,
	}

	builtinSymbol = map[string]string{
		"USD": "$", "EUR": "€", "GBP": "£", "CNY": "¥", "JPY": "¥",
		"KRW": "₩", "CHF": "CHF", "CAD": "C$", "AUD": "A$", "NZD": "NZ$",
		"HKD": "HK$", "SGD": "S$", "INR": "₹", "RUB": "₽", "BRL": "R$",
		"MXN": "$", "ZAR": "R", "SEK": "kr", "NOK": "kr", "DKK": "kr",
		"PLN": "zł", "TRY": "₺", "THB": "฿", "PHP": "₱", "MYR": "RM",
		"IDR": "Rp", "VND": "₫", "TWD": "NT$", "ILS": "₪", "AED": "د.إ",
		"SAR": "﷼", "BHD": ".د.ب", "KWD": "د.ك", "JOD": "د.ا", "OMR": "﷼",
	}

	// customCurrencies overlays / extends the builtin tables.
	customCurrencies = map[string]Currency{}
)

// RegisterCurrency registers a custom currency.
// Precision must be in [0, 18]. Code is normalized to upper-case.
// Builtin ISO codes cannot be overridden (prevents silently rescaling
// existing Money values of USD/CNY/…).
func RegisterCurrency(c Currency) error {
	code := strings.ToUpper(strings.TrimSpace(c.Code))
	if code == "" {
		return fmt.Errorf("%w: empty currency code", ErrUnknownCurrency)
	}
	if c.Precision < 0 || c.Precision > 18 {
		return fmt.Errorf("%w: precision %d out of range [0,18]", ErrInvalidAmount, c.Precision)
	}
	if _, builtin := builtinPrecision[code]; builtin {
		return fmt.Errorf("%w: cannot override builtin currency %s", ErrUnknownCurrency, code)
	}
	symbol := c.Symbol
	if symbol == "" {
		symbol = code
	}
	currencyMu.Lock()
	customCurrencies[code] = Currency{Code: code, Precision: c.Precision, Symbol: symbol}
	currencyMu.Unlock()
	return nil
}

// UnregisterCurrency removes a previously registered custom currency.
// Builtin ISO entries cannot be removed; unregistering them is a no-op that
// only clears an overlay if one was registered with the same code.
func UnregisterCurrency(code string) {
	code = strings.ToUpper(strings.TrimSpace(code))
	currencyMu.Lock()
	delete(customCurrencies, code)
	currencyMu.Unlock()
}

// LookupCurrency returns metadata for a currency code.
func LookupCurrency(code string) (Currency, bool) {
	code = strings.ToUpper(strings.TrimSpace(code))
	currencyMu.RLock()
	if c, ok := customCurrencies[code]; ok {
		currencyMu.RUnlock()
		return c, true
	}
	currencyMu.RUnlock()
	if p, ok := builtinPrecision[code]; ok {
		sym := builtinSymbol[code]
		if sym == "" {
			sym = code
		}
		return Currency{Code: code, Precision: p, Symbol: sym}, true
	}
	return Currency{}, false
}

// CurrencyPrecision returns decimal places for currency.
// Unknown currencies default to 2 (common case) for formatting helpers;
// prefer LookupCurrency / IsKnownCurrency when validating input.
func CurrencyPrecision(currency string) int {
	if c, ok := LookupCurrency(currency); ok {
		return c.Precision
	}
	return 2
}

// CurrencySymbol returns the display symbol (or the code itself).
func CurrencySymbol(currency string) string {
	if c, ok := LookupCurrency(currency); ok {
		return c.Symbol
	}
	code := strings.ToUpper(strings.TrimSpace(currency))
	if code == "" {
		return ""
	}
	return code
}

// IsKnownCurrency reports whether currency is builtin or registered.
func IsKnownCurrency(currency string) bool {
	_, ok := LookupCurrency(currency)
	return ok
}
