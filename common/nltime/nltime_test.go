// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package nltime

import (
	"testing"
	"time"
)

// Fixed reference time for deterministic tests.
// 2026-08-15 10:30:00 UTC is a Saturday.
var refTime = time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)

// ──────────────────────────────────────────────
// Keywords
// ──────────────────────────────────────────────

func TestParse_Now(t *testing.T) {
	result, err := Parse("now", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !result.Equal(refTime) {
		t.Fatalf("now = %v, want %v", result, refTime)
	}
}

func TestParse_Today(t *testing.T) {
	result, err := Parse("today", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("today = %v, want %v", result, expected)
	}
}

func TestParse_Yesterday(t *testing.T) {
	result, err := Parse("yesterday", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("yesterday = %v, want %v", result, expected)
	}
}

func TestParse_Tomorrow(t *testing.T) {
	result, err := Parse("tomorrow", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("tomorrow = %v, want %v", result, expected)
	}
}

func TestParse_Noon(t *testing.T) {
	result, err := Parse("noon", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("noon = %v, want %v", result, expected)
	}
}

func TestParse_Midnight(t *testing.T) {
	result, err := Parse("midnight", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("midnight = %v, want %v", result, expected)
	}
}

func TestParse_ThisMorning(t *testing.T) {
	result, err := Parse("this morning", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Hour() != 8 {
		t.Fatalf("this morning hour = %d, want 8", result.Hour())
	}
}

func TestParse_ThisAfternoon(t *testing.T) {
	result, err := Parse("this afternoon", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Hour() != 14 {
		t.Fatalf("this afternoon hour = %d, want 14", result.Hour())
	}
}

func TestParse_ThisEvening(t *testing.T) {
	result, err := Parse("this evening", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Hour() != 18 {
		t.Fatalf("this evening hour = %d, want 18", result.Hour())
	}
}

// ──────────────────────────────────────────────
// Relative
// ──────────────────────────────────────────────

func TestParse_3DaysAgo(t *testing.T) {
	result, err := Parse("3 days ago", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := refTime.Add(-3 * 24 * time.Hour)
	if !result.Equal(expected) {
		t.Fatalf("3 days ago = %v, want %v", result, expected)
	}
}

func TestParse_In2Hours(t *testing.T) {
	result, err := Parse("in 2 hours", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := refTime.Add(2 * time.Hour)
	if !result.Equal(expected) {
		t.Fatalf("in 2 hours = %v, want %v", result, expected)
	}
}

func TestParse_5MinutesFromNow(t *testing.T) {
	result, err := Parse("5 minutes from now", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := refTime.Add(5 * time.Minute)
	if !result.Equal(expected) {
		t.Fatalf("5 minutes from now = %v, want %v", result, expected)
	}
}

func TestParse_10SecondsLater(t *testing.T) {
	result, err := Parse("10 seconds later", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := refTime.Add(10 * time.Second)
	if !result.Equal(expected) {
		t.Fatalf("10 seconds later = %v, want %v", result, expected)
	}
}

func TestParse_ShortUnits(t *testing.T) {
	tests := []struct {
		expr string
		want time.Duration
	}{
		{"3d ago", -3 * 24 * time.Hour},
		{"2h from now", 2 * time.Hour},
		{"5m ago", -5 * time.Minute},
		{"10s later", 10 * time.Second},
		{"1w ago", -7 * 24 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Parse(tt.expr, refTime)
			if err != nil {
				t.Fatalf("Parse(%q) failed: %v", tt.expr, err)
			}
			expected := refTime.Add(tt.want)
			if !result.Equal(expected) {
				t.Fatalf("Parse(%q) = %v, want %v", tt.expr, result, expected)
			}
		})
	}
}

func TestParse_1YearAgo(t *testing.T) {
	result, err := Parse("1 year ago", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := refTime.Add(-365 * 24 * time.Hour)
	if !result.Equal(expected) {
		t.Fatalf("1 year ago = %v, want %v", result, expected)
	}
}

func TestParse_2MonthsAgo(t *testing.T) {
	result, err := Parse("2 months ago", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := refTime.Add(-60 * 24 * time.Hour) // approximate
	if !result.Equal(expected) {
		t.Fatalf("2 months ago = %v, want %v", result, expected)
	}
}

// ──────────────────────────────────────────────
// Weekday references
// ──────────────────────────────────────────────

func TestParse_NextMonday(t *testing.T) {
	result, err := Parse("next monday", refTime) // refTime is Saturday
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Weekday() != time.Monday {
		t.Fatalf("next monday weekday = %v, want Monday", result.Weekday())
	}
	if result.Day() != 17 {
		t.Fatalf("next monday day = %d, want 17", result.Day())
	}
}

func TestParse_LastFriday(t *testing.T) {
	result, err := Parse("last friday", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Weekday() != time.Friday {
		t.Fatalf("last friday weekday = %v, want Friday", result.Weekday())
	}
	if result.Day() != 14 {
		t.Fatalf("last friday day = %d, want 14", result.Day())
	}
}

func TestParse_PreviousTuesday(t *testing.T) {
	result, err := Parse("previous tuesday", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Weekday() != time.Tuesday {
		t.Fatalf("previous tuesday weekday = %v, want Tuesday", result.Weekday())
	}
}

func TestParse_ThisWednesday(t *testing.T) {
	result, err := Parse("this wednesday", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Weekday() != time.Wednesday {
		t.Fatalf("this wednesday weekday = %v, want Wednesday", result.Weekday())
	}
}

func TestParse_NextWeekend(t *testing.T) {
	result, err := Parse("next weekend", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Weekday() != time.Saturday {
		t.Fatalf("next weekend weekday = %v, want Saturday", result.Weekday())
	}
}

func TestParse_ShortWeekday(t *testing.T) {
	result, err := Parse("next mon", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Weekday() != time.Monday {
		t.Fatalf("next mon weekday = %v, want Monday", result.Weekday())
	}
}

// ──────────────────────────────────────────────
// Combined date-time
// ──────────────────────────────────────────────

func TestParse_TomorrowAt3pm(t *testing.T) {
	result, err := Parse("tomorrow at 3pm", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := time.Date(2026, 8, 16, 15, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("tomorrow at 3pm = %v, want %v", result, expected)
	}
}

func TestParse_TodayAt1030(t *testing.T) {
	result, err := Parse("today at 10:30", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("today at 10:30 = %v, want %v", result, expected)
	}
}

func TestParse_YesterdayAtNoon(t *testing.T) {
	result, err := Parse("yesterday at noon", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("yesterday at noon = %v, want %v", result, expected)
	}
}

// ──────────────────────────────────────────────
// Time of day
// ──────────────────────────────────────────────

func TestParse_3pm(t *testing.T) {
	result, err := Parse("3pm", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Hour() != 15 {
		t.Fatalf("3pm hour = %d, want 15", result.Hour())
	}
}

func TestParse_10am(t *testing.T) {
	result, err := Parse("10am", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Hour() != 10 {
		t.Fatalf("10am hour = %d, want 10", result.Hour())
	}
}

func TestParse_1030am(t *testing.T) {
	result, err := Parse("10:30am", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Hour() != 10 || result.Minute() != 30 {
		t.Fatalf("10:30am = %v, want 10:30", result)
	}
}

func TestParse_12am(t *testing.T) {
	result, err := Parse("12am", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Hour() != 0 {
		t.Fatalf("12am hour = %d, want 0", result.Hour())
	}
}

func TestParse_12pm(t *testing.T) {
	result, err := Parse("12pm", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Hour() != 12 {
		t.Fatalf("12pm hour = %d, want 12", result.Hour())
	}
}

func TestParse_Time24Hour(t *testing.T) {
	result, err := Parse("15:30", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Hour() != 15 || result.Minute() != 30 {
		t.Fatalf("15:30 = %v, want 15:30", result)
	}
}

// ──────────────────────────────────────────────
// Error cases
// ──────────────────────────────────────────────

func TestParse_Empty(t *testing.T) {
	_, err := Parse("", refTime)
	if err == nil {
		t.Fatal("Parse(\"\") should error")
	}
}

func TestParse_Invalid(t *testing.T) {
	_, err := Parse("some random text", refTime)
	if err == nil {
		t.Fatal("Parse with invalid text should error")
	}
}

func TestParse_InvalidUnit(t *testing.T) {
	_, err := Parse("3 foos ago", refTime)
	if err == nil {
		t.Fatal("Parse with invalid unit should error")
	}
}

// ──────────────────────────────────────────────
// MustParse
// ──────────────────────────────────────────────

func TestMustParse(t *testing.T) {
	result := MustParse("tomorrow", refTime)
	if result.IsZero() {
		t.Fatal("MustParse returned zero")
	}
}

func TestMustParse_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustParse with invalid expr should panic")
		}
	}()
	MustParse("invalid", refTime)
}

// ──────────────────────────────────────────────
// ParseDuration
// ──────────────────────────────────────────────

func TestParseDuration_Compound(t *testing.T) {
	d, err := ParseDuration("2 hours 30 minutes")
	if err != nil {
		t.Fatalf("ParseDuration failed: %v", err)
	}
	expected := 2*time.Hour + 30*time.Minute
	if d != expected {
		t.Fatalf("ParseDuration = %v, want %v", d, expected)
	}
}

func TestParseDuration_ShortUnits(t *testing.T) {
	d, err := ParseDuration("1d 2h 30m")
	if err != nil {
		t.Fatalf("ParseDuration failed: %v", err)
	}
	expected := 24*time.Hour + 2*time.Hour + 30*time.Minute
	if d != expected {
		t.Fatalf("ParseDuration = %v, want %v", d, expected)
	}
}

func TestParseDuration_StandardGo(t *testing.T) {
	d, err := ParseDuration("1h30m")
	if err != nil {
		t.Fatalf("ParseDuration failed: %v", err)
	}
	if d != 90*time.Minute {
		t.Fatalf("ParseDuration = %v, want 90m", d)
	}
}

func TestParseDuration_SingleUnit(t *testing.T) {
	d, err := ParseDuration("45 minutes")
	if err != nil {
		t.Fatalf("ParseDuration failed: %v", err)
	}
	if d != 45*time.Minute {
		t.Fatalf("ParseDuration = %v, want 45m", d)
	}
}

func TestParseDuration_Empty(t *testing.T) {
	_, err := ParseDuration("")
	if err == nil {
		t.Fatal("ParseDuration(\"\") should error")
	}
}

func TestParseDuration_Invalid(t *testing.T) {
	_, err := ParseDuration("invalid")
	if err == nil {
		t.Fatal("ParseDuration(\"invalid\") should error")
	}
}

func TestParseDuration_UnknownUnit(t *testing.T) {
	_, err := ParseDuration("3 foos")
	if err == nil {
		t.Fatal("ParseDuration with unknown unit should error")
	}
}

func TestMustParseDuration(t *testing.T) {
	d := MustParseDuration("2 hours")
	if d != 2*time.Hour {
		t.Fatalf("MustParseDuration = %v, want 2h", d)
	}
}

func TestMustParseDuration_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustParseDuration with invalid should panic")
		}
	}()
	MustParseDuration("invalid")
}

// ──────────────────────────────────────────────
// Case insensitivity
// ──────────────────────────────────────────────

func TestParse_CaseInsensitive(t *testing.T) {
	result, err := Parse("3 DAYS AGO", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := refTime.Add(-3 * 24 * time.Hour)
	if !result.Equal(expected) {
		t.Fatalf("3 DAYS AGO = %v, want %v", result, expected)
	}
}

func TestParse_NextMonday_CaseInsensitive(t *testing.T) {
	result, err := Parse("NEXT MONDAY", refTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Weekday() != time.Monday {
		t.Fatalf("NEXT MONDAY weekday = %v", result.Weekday())
	}
}

// ──────────────────────────────────────────────
// Zero now fallback
// ──────────────────────────────────────────────

func TestParse_ZeroNowFallback(t *testing.T) {
	// Should not panic when now is zero.
	_, err := Parse("now", time.Time{})
	if err != nil {
		t.Fatalf("Parse with zero now failed: %v", err)
	}
}
