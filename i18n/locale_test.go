// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package i18n

import "testing"

func TestManager_ResolveLocale_Exact(t *testing.T) {
	m := NewManager(&Config{
		SupportedLocales: []Locale{LocaleEn, LocaleZhCN, LocaleZhTW, LocaleJaJP},
		DefaultLocale:    LocaleEn,
	})
	cases := []struct {
		in  string
		out Locale
	}{
		{"en", LocaleEn},
		{"zh-CN", LocaleZhCN},
		{"zh-TW", LocaleZhTW},
		{"ja-JP", LocaleJaJP},
		{"EN", LocaleEn}, // case-insensitive
	}
	for _, c := range cases {
		if got := m.ResolveLocale(c.in); got != c.out {
			t.Errorf("ResolveLocale(%q) = %s, want %s", c.in, got, c.out)
		}
	}
}

func TestManager_ResolveLocale_LangPrefix(t *testing.T) {
	m := NewManager(&Config{
		SupportedLocales: []Locale{LocaleEnUS, LocaleZhCN},
		DefaultLocale:    LocaleEnUS,
	})
	if got := m.ResolveLocale("en"); got != LocaleEnUS {
		t.Errorf("ResolveLocale(en) = %s, want en-US", got)
	}
	if got := m.ResolveLocale("zh"); got != LocaleZhCN {
		t.Errorf("ResolveLocale(zh) = %s, want zh-CN", got)
	}
}

func TestManager_ResolveLocale_Underscore(t *testing.T) {
	m := NewManager(&Config{
		SupportedLocales: []Locale{LocaleZhCN},
		DefaultLocale:    LocaleZhCN,
	})
	if got := m.ResolveLocale("zh_CN"); got != LocaleZhCN {
		t.Errorf("ResolveLocale(zh_CN) = %s, want zh-CN", got)
	}
}

func TestManager_ResolveLocale_Unknown(t *testing.T) {
	m := NewManager(&Config{
		SupportedLocales: []Locale{LocaleEn},
		DefaultLocale:    LocaleEn,
	})
	if got := m.ResolveLocale("klingon"); got != LocaleEn {
		t.Errorf("ResolveLocale(klingon) = %s, want en (default)", got)
	}
}

func TestManager_ResolveLocale_Empty(t *testing.T) {
	m := NewManager(&Config{
		SupportedLocales: []Locale{LocaleEn},
		DefaultLocale:    LocaleEn,
	})
	if got := m.ResolveLocale(""); got != LocaleEn {
		t.Errorf("ResolveLocale('') = %s, want en", got)
	}
}

func TestManager_ParseAcceptLanguage(t *testing.T) {
	m := NewManager(&Config{
		SupportedLocales: []Locale{LocaleEn, LocaleZhCN, LocaleZhTW, LocaleFrFR},
		DefaultLocale:    LocaleEn,
	})
	cases := []struct {
		in  string
		out Locale
	}{
		{"en-US,en;q=0.9", LocaleEn},
		{"zh-CN,zh;q=0.9,en;q=0.8", LocaleZhCN},
		{"fr-FR,fr;q=0.9,en;q=0.8", LocaleFrFR},
		{"", LocaleEn},
		{"klingon,en;q=0.9", LocaleEn},
	}
	for _, c := range cases {
		if got := m.ParseAcceptLanguage(c.in); got != c.out {
			t.Errorf("ParseAcceptLanguage(%q) = %s, want %s", c.in, got, c.out)
		}
	}
}
