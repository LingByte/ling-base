// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package mymemory

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTranslate_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"responseData": {"translatedText": "Hello", "match": 0.95},
			"quotaFinished": false,
			"mtLangSupported": true,
			"responseDetails": "",
			"responseStatus": 200
		}`))
	}))
	defer server.Close()

	tr := New("")
	tr.baseURL = server.URL

	result, err := tr.Translate("你好", "zh-CN", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello" {
		t.Errorf("got %q, want Hello", result)
	}
}

func TestTranslate_SameLanguage(t *testing.T) {
	tr := New("")
	result, err := tr.Translate("Hello", "en", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello" {
		t.Errorf("got %q, want Hello", result)
	}
}

func TestTranslate_EmptyText(t *testing.T) {
	tr := New("")
	_, err := tr.Translate("", "en", "zh-CN")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestTranslate_QuotaFinished(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"responseData": {"translatedText": "", "match": 0},
			"quotaFinished": true,
			"mtLangSupported": true,
			"responseDetails": "",
			"responseStatus": 200
		}`))
	}))
	defer server.Close()

	tr := New("")
	tr.baseURL = server.URL

	_, err := tr.Translate("Hello", "en", "zh-CN")
	if err == nil {
		t.Fatal("expected error for quota finished")
	}
}

func TestTranslate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`internal error`))
	}))
	defer server.Close()

	tr := New("")
	tr.baseURL = server.URL

	_, err := tr.Translate("Hello", "en", "zh-CN")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestTranslateBatch(t *testing.T) {
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"responseData": {"translatedText": "Hello", "match": 0.95},
			"quotaFinished": false,
			"mtLangSupported": true,
			"responseDetails": "",
			"responseStatus": 200
		}`))
	}))
	defer server.Close()

	tr := New("")
	tr.baseURL = server.URL

	results, err := tr.TranslateBatch([]string{"你好", "世界"}, "zh-CN", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
	if count != 2 {
		t.Errorf("got %d API calls, want 2", count)
	}
}

func TestNormalizeLangCode(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"en", "en"},
		{"EN", "en"},
		{"zh", "zh-CN"},
		{"zh-CN", "zh-CN"},
		{"zh-cn", "zh-CN"},
		{"zh-TW", "zh-TW"},
		{"en-US", "en"},
		{"fr-FR", "fr"},
		{"unknown", "unknown"},
	}
	for _, c := range cases {
		if got := normalizeLangCode(c.in); got != c.out {
			t.Errorf("normalizeLangCode(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}

func TestSupportedLanguages(t *testing.T) {
	langs := SupportedLanguages()
	if len(langs) == 0 {
		t.Fatal("expected non-empty language list")
	}
	hasEn, hasZh := false, false
	for _, l := range langs {
		if l == "en" {
			hasEn = true
		}
		if l == "zh-CN" {
			hasZh = true
		}
	}
	if !hasEn {
		t.Error("expected English in supported languages")
	}
	if !hasZh {
		t.Error("expected Chinese in supported languages")
	}
}
