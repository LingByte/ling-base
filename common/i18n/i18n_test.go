// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager_NilConfig(t *testing.T) {
	m := NewManager(nil)
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if m.defaultLocale != DefaultLocale {
		t.Errorf("default = %s, want %s", m.defaultLocale, DefaultLocale)
	}
}

func TestNewManager_WithConfig(t *testing.T) {
	m := NewManager(&Config{
		DefaultLocale:    LocaleZhCN,
		SupportedLocales: []Locale{LocaleEn, LocaleZhCN, LocaleZhTW},
		FallbackLocale:   LocaleEn,
	})
	if m.defaultLocale != LocaleZhCN {
		t.Errorf("default = %s, want zh-CN", m.defaultLocale)
	}
	if len(m.supportedLocales) != 3 {
		t.Errorf("supported count = %d, want 3", len(m.supportedLocales))
	}
}

func TestManager_SetGetTranslation(t *testing.T) {
	m := NewManager(nil)
	m.SetTranslation(LocaleEn, "test.key", "Test Value")
	if v := m.GetTranslation(LocaleEn, "test.key"); v != "Test Value" {
		t.Errorf("got %q, want %q", v, "Test Value")
	}
}

func TestManager_GetTranslation_Fallback(t *testing.T) {
	m := NewManager(&Config{
		SupportedLocales: []Locale{LocaleEn, LocaleZhCN},
		FallbackLocale:   LocaleEn,
	})
	m.SetTranslation(LocaleEn, "key", "Fallback")
	if v := m.GetTranslation(LocaleZhCN, "key"); v != "Fallback" {
		t.Errorf("fallback got %q, want %q", v, "Fallback")
	}
}

func TestManager_GetTranslation_NotFound(t *testing.T) {
	m := NewManager(nil)
	if v := m.GetTranslation(LocaleEn, "nope"); v != "nope" {
		t.Errorf("not found got %q, want %q", v, "nope")
	}
}

func TestManager_T_WithArgs(t *testing.T) {
	m := NewManager(nil)
	m.SetTranslation(LocaleEn, "hello", "Hello, %s!")
	if v := m.T(LocaleEn, "hello", "World"); v != "Hello, World!" {
		t.Errorf("got %q, want %q", v, "Hello, World!")
	}
}

func TestManager_LoadTranslationFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	os.WriteFile(path, []byte(`{"k": "v"}`), 0644)

	m := NewManager(&Config{SupportedLocales: []Locale{LocaleEn}})
	if err := m.LoadTranslationFile(LocaleEn, path); err != nil {
		t.Fatalf("LoadTranslationFile: %v", err)
	}
	if v := m.GetTranslation(LocaleEn, "k"); v != "v" {
		t.Errorf("got %q, want %q", v, "v")
	}
}

func TestManager_LoadTranslationFile_UnsupportedLocale(t *testing.T) {
	m := NewManager(&Config{SupportedLocales: []Locale{LocaleEn}})
	if err := m.LoadTranslationFile(LocaleFrFR, "dummy.json"); err == nil {
		t.Fatal("expected error for unsupported locale")
	}
}

func TestManager_LoadTranslations_Dir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "messages.en.json"), []byte(`{"welcome": "Welcome"}`), 0644)
	os.WriteFile(filepath.Join(dir, "messages.zh-CN.json"), []byte(`{"welcome": "欢迎"}`), 0644)

	m := NewManager(&Config{
		SupportedLocales: []Locale{LocaleEn, LocaleZhCN},
		TranslationsPath: dir,
	})
	if v := m.GetTranslation(LocaleEn, "welcome"); v != "Welcome" {
		t.Errorf("en got %q", v)
	}
	if v := m.GetTranslation(LocaleZhCN, "welcome"); v != "欢迎" {
		t.Errorf("zh-CN got %q", v)
	}
}

func TestManager_LoadTranslations_NonExistentPath(t *testing.T) {
	m := NewManager(&Config{TranslationsPath: "/no/such/path"})
	// NewManager swallows the error, but translations should be empty
	if v := m.GetTranslation(LocaleEn, "welcome"); v != "welcome" {
		t.Errorf("got %q, want key fallback", v)
	}
}

func TestManager_IsSupportedLocale(t *testing.T) {
	m := NewManager(&Config{SupportedLocales: []Locale{LocaleEn, LocaleZhCN}})
	if !m.IsSupportedLocale(LocaleEn) {
		t.Error("en should be supported")
	}
	if m.IsSupportedLocale(LocaleFrFR) {
		t.Error("fr should not be supported")
	}
}

func TestManager_GetTranslations(t *testing.T) {
	m := NewManager(nil)
	m.SetTranslation(LocaleEn, "a", "1")
	m.SetTranslation(LocaleEn, "b", "2")
	trans := m.GetTranslations(LocaleEn)
	if len(trans) != 2 {
		t.Errorf("count = %d, want 2", len(trans))
	}
}

func TestManager_Translate_NoTranslator(t *testing.T) {
	m := NewManager(nil)
	_, err := m.Translate("hello", "en", "zh-CN")
	if err == nil {
		t.Fatal("expected error when no translator configured")
	}
}
