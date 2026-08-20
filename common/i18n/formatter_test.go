// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package i18n

import (
	"testing"
	"time"
)

func TestFormatter_FormatNumber(t *testing.T) {
	f := NewFormatter(LocaleEn)
	if v := f.FormatNumber(1234.56, 2); v != "1,234.56" {
		t.Errorf("en got %q, want 1,234.56", v)
	}

	fCN := NewFormatter(LocaleZhCN)
	if v := fCN.FormatNumber(1234.56, 2); v != "1,234.56" {
		t.Errorf("zh-CN got %q, want 1,234.56", v)
	}

	fDE := NewFormatter(LocaleDeDE)
	if v := fDE.FormatNumber(1234.56, 2); v != "1.234,56" {
		t.Errorf("de got %q, want 1.234,56", v)
	}
}

func TestFormatter_FormatCurrency(t *testing.T) {
	f := NewFormatter(LocaleEn)
	if v := f.FormatCurrency(1234.56, "USD"); v != "$1,234.56" {
		t.Errorf("USD got %q, want $1,234.56", v)
	}

	fCN := NewFormatter(LocaleZhCN)
	if v := fCN.FormatCurrency(1234.56, "CNY"); v != "¥1,234.56" {
		t.Errorf("CNY got %q, want ¥1,234.56", v)
	}

	fDE := NewFormatter(LocaleDeDE)
	if v := fDE.FormatCurrency(1234.56, "EUR"); v != "1.234,56 €" {
		t.Errorf("EUR got %q, want 1.234,56 €", v)
	}
}

func TestFormatter_FormatDate(t *testing.T) {
	date := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	f := NewFormatter(LocaleEn)
	if v := f.FormatDate(date, "YYYY-MM-DD"); v != "2024-01-15" {
		t.Errorf("got %q, want 2024-01-15", v)
	}

	fCN := NewFormatter(LocaleZhCN)
	if v := fCN.FormatDate(date, ""); v != "2024-01-15" {
		t.Errorf("zh-CN default got %q, want 2024-01-15", v)
	}

	fUS := NewFormatter(LocaleEnUS)
	if v := fUS.FormatDate(date, ""); v != "01/15/2024" {
		t.Errorf("en-US default got %q, want 01/15/2024", v)
	}
}

func TestFormatter_FormatRelativeTime(t *testing.T) {
	f := NewFormatter(LocaleEn)
	now := time.Now()

	cases := []time.Duration{
		-30 * time.Second,
		-5 * time.Minute,
		-2 * time.Hour,
		-3 * 24 * time.Hour,
	}
	for _, d := range cases {
		if v := f.FormatRelativeTime(now.Add(d)); v == "" {
			t.Errorf("expected non-empty relative time for %v", d)
		}
	}

	fCN := NewFormatter(LocaleZhCN)
	if v := fCN.FormatRelativeTime(now.Add(-time.Hour)); v == "" {
		t.Error("zh-CN relative time should not be empty")
	}
}

func TestFormatter_AddThousandSeparators(t *testing.T) {
	f := NewFormatter(LocaleEn)
	if v := f.addThousandSeparators("1234567", ",", "."); v != "1,234,567" {
		t.Errorf("got %q, want 1,234,567", v)
	}
	if v := f.addThousandSeparators("1234.56", ",", "."); v != "1,234.56" {
		t.Errorf("got %q, want 1,234.56", v)
	}
}
