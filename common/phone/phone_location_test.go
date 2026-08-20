// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package phone

import (
	"strings"
	"testing"
)

func TestFormatPhoneLocation(t *testing.T) {
	got := FormatPhoneLocation("四川", "成都", "中国移动")
	if got != "四川成都(中国移动)" {
		t.Fatalf("got %q", got)
	}
	got = FormatPhoneLocation("上海", "上海", "中国电信")
	if got != "上海(中国电信)" {
		t.Fatalf("duplicate province/city: got %q", got)
	}
	got = FormatPhoneLocation("河南", "郑州", "中国广电")
	if got != "河南郑州(中国广电)" {
		t.Fatalf("china radio diff city got %q", got)
	}
	got = FormatPhoneLocation("重庆", "重庆", "中国广电")
	if got != "重庆(中国广电)" {
		t.Fatalf("china radio same city got %q", got)
	}
	if FormatPhoneLocation("", "", "") != "" {
		t.Fatal("empty")
	}
}

func TestLookupPhoneLocation(t *testing.T) {
	got := LookupPhoneLocation("19511899044")
	if got == "" {
		t.Fatal("expected lookup result")
	}
	if !strings.Contains(got, "成都") {
		t.Fatalf("got %q", got)
	}

	radioPhone := "19208101234"
	got = LookupPhoneLocation(radioPhone)
	if got == "" {
		t.Fatalf("广电号码没有解析出结果")
	}
	if !strings.Contains(got, "成都") || !strings.Contains(got, "中国广电") {
		t.Fatalf("广电解析错误 got = %s", got)
	}

	if LookupPhoneLocation("123") != "" {
		t.Fatal("short number")
	}
	if LookupPhoneLocation("192") != "" {
		t.Fatal("short radio number should empty")
	}
}

func TestNormalizePhoneDigits(t *testing.T) {
	if got := NormalizePhoneDigits("+86 138-0013-8000"); got != "8613800138000" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizePhoneDigits("  13800138000  "); got != "13800138000" {
		t.Fatalf("got %q", got)
	}
}
