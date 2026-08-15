// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package cron

import (
	"testing"
	"time"
)

// ──────────────────────────────────────────────
// Parse
// ──────────────────────────────────────────────

func TestParse_Standard5Field(t *testing.T) {
	e, err := Parse("*/5 * * * *")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if e.HasSeconds() {
		t.Fatal("5-field should not have seconds")
	}
}

func TestParse_6Field(t *testing.T) {
	e, err := Parse("0 */30 * * * *")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !e.HasSeconds() {
		t.Fatal("6-field should have seconds")
	}
}

func TestParse_Empty(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Fatal("Parse(\"\") should error")
	}
}

func TestParse_TooFewFields(t *testing.T) {
	_, err := Parse("* * *")
	if err == nil {
		t.Fatal("Parse with 3 fields should error")
	}
}

func TestParse_TooManyFields(t *testing.T) {
	_, err := Parse("* * * * * * *")
	if err == nil {
		t.Fatal("Parse with 7 fields should error")
	}
}

func TestParse_InvalidValue(t *testing.T) {
	_, err := Parse("60 * * * *") // minute 60 out of range
	if err == nil {
		t.Fatal("Parse with minute 60 should error")
	}
}

func TestParse_InvalidStep(t *testing.T) {
	_, err := Parse("*/0 * * * *")
	if err == nil {
		t.Fatal("Parse with step 0 should error")
	}
}

func TestParse_NamedMonth(t *testing.T) {
	_, err := Parse("* * * jan *")
	if err != nil {
		t.Fatalf("Parse with named month failed: %v", err)
	}
}

func TestParse_NamedDOW(t *testing.T) {
	_, err := Parse("* * * * mon")
	if err != nil {
		t.Fatalf("Parse with named DOW failed: %v", err)
	}
}

func TestParse_CommaList(t *testing.T) {
	_, err := Parse("1,3,5 * * * *")
	if err != nil {
		t.Fatalf("Parse comma list failed: %v", err)
	}
}

func TestParse_Range(t *testing.T) {
	_, err := Parse("1-5 * * * *")
	if err != nil {
		t.Fatalf("Parse range failed: %v", err)
	}
}

func TestParse_RangeWithStep(t *testing.T) {
	_, err := Parse("1-10/3 * * * *")
	if err != nil {
		t.Fatalf("Parse range with step failed: %v", err)
	}
}

func TestParse_LastDOM(t *testing.T) {
	e, err := Parse("* * L * *")
	if err != nil {
		t.Fatalf("Parse L failed: %v", err)
	}
	if !e.lastDOM {
		t.Fatal("lastDOM should be true")
	}
}

func TestParse_NearestWeekday(t *testing.T) {
	e, err := Parse("* * 15W * *")
	if err != nil {
		t.Fatalf("Parse 15W failed: %v", err)
	}
	if e.nearestW != 15 {
		t.Fatalf("nearestW = %d, want 15", e.nearestW)
	}
}

func TestParse_LastDOW(t *testing.T) {
	e, err := Parse("* * * * 5L")
	if err != nil {
		t.Fatalf("Parse 5L failed: %v", err)
	}
	if e.lastDOW != 5 {
		t.Fatalf("lastDOW = %d, want 5", e.lastDOW)
	}
}

func TestMustParse(t *testing.T) {
	e := MustParse("*/5 * * * *")
	if e == nil {
		t.Fatal("MustParse returned nil")
	}
}

func TestMustParse_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustParse with invalid expr should panic")
		}
	}()
	MustParse("invalid")
}

// ──────────────────────────────────────────────
// Descriptors
// ──────────────────────────────────────────────

func TestParse_Hourly(t *testing.T) {
	e, err := Parse("@hourly")
	if err != nil {
		t.Fatalf("Parse @hourly failed: %v", err)
	}
	next, err := e.Next(time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if next.Hour() != 11 || next.Minute() != 0 {
		t.Fatalf("@hourly next = %v, want 11:00", next)
	}
}

func TestParse_Daily(t *testing.T) {
	e, err := Parse("@daily")
	if err != nil {
		t.Fatalf("Parse @daily failed: %v", err)
	}
	next, err := e.Next(time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if next.Day() != 16 || next.Hour() != 0 {
		t.Fatalf("@daily next = %v, want next day 00:00", next)
	}
}

func TestParse_Weekly(t *testing.T) {
	e, err := Parse("@weekly")
	if err != nil {
		t.Fatalf("Parse @weekly failed: %v", err)
	}
	next, err := e.Next(time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)) // Saturday
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if next.Weekday() != time.Sunday {
		t.Fatalf("@weekly next weekday = %v, want Sunday", next.Weekday())
	}
}

func TestParse_Monthly(t *testing.T) {
	e, err := Parse("@monthly")
	if err != nil {
		t.Fatalf("Parse @monthly failed: %v", err)
	}
	next, err := e.Next(time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if next.Day() != 1 || next.Month() != 9 {
		t.Fatalf("@monthly next = %v, want Sep 1", next)
	}
}

func TestParse_Yearly(t *testing.T) {
	e, err := Parse("@yearly")
	if err != nil {
		t.Fatalf("Parse @yearly failed: %v", err)
	}
	next, err := e.Next(time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if next.Month() != 1 || next.Day() != 1 || next.Year() != 2027 {
		t.Fatalf("@yearly next = %v, want Jan 1 2027", next)
	}
}

func TestParse_Annually(t *testing.T) {
	_, err := Parse("@annually")
	if err != nil {
		t.Fatalf("Parse @annually failed: %v", err)
	}
}

func TestParse_Midnight(t *testing.T) {
	_, err := Parse("@midnight")
	if err != nil {
		t.Fatalf("Parse @midnight failed: %v", err)
	}
}

func TestParse_Every(t *testing.T) {
	e, err := Parse("@every 10s")
	if err != nil {
		t.Fatalf("Parse @every 10s failed: %v", err)
	}
	if !e.IsEvery() {
		t.Fatal("IsEvery should be true")
	}
	if e.EveryDuration() != 10*time.Second {
		t.Fatalf("EveryDuration = %v, want 10s", e.EveryDuration())
	}
}

func TestParse_EveryInvalidDuration(t *testing.T) {
	_, err := Parse("@every invalid")
	if err == nil {
		t.Fatal("Parse @every invalid should error")
	}
}

func TestParse_EveryZero(t *testing.T) {
	_, err := Parse("@every 0s")
	if err == nil {
		t.Fatal("Parse @every 0s should error")
	}
}

func TestParse_EveryNoDuration(t *testing.T) {
	_, err := Parse("@every")
	if err == nil {
		t.Fatal("Parse @every without duration should error")
	}
}

func TestParse_UnknownDescriptor(t *testing.T) {
	_, err := Parse("@unknown")
	if err == nil {
		t.Fatal("Parse @unknown should error")
	}
}

// ──────────────────────────────────────────────
// Next
// ──────────────────────────────────────────────

func TestNext_Every5Minutes(t *testing.T) {
	e := MustParse("*/5 * * * *")
	now := time.Date(2026, 8, 15, 10, 32, 0, 0, time.UTC)
	next, err := e.Next(now)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	expected := time.Date(2026, 8, 15, 10, 35, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("Next = %v, want %v", next, expected)
	}
}

func TestNext_EveryMinute(t *testing.T) {
	e := MustParse("* * * * *")
	now := time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)
	next, err := e.Next(now)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	expected := time.Date(2026, 8, 15, 10, 31, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("Next = %v, want %v", next, expected)
	}
}

func TestNext_SpecificTime(t *testing.T) {
	e := MustParse("0 9 * * *") // every day at 9am
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	next, err := e.Next(now)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	expected := time.Date(2026, 8, 16, 9, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("Next = %v, want %v", next, expected)
	}
}

func TestNext_WithSeconds(t *testing.T) {
	e := MustParse("30 * * * * *") // every minute at 30 seconds
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	next, err := e.Next(now)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	expected := time.Date(2026, 8, 15, 10, 0, 30, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("Next = %v, want %v", next, expected)
	}
}

func TestNext_MonthTransition(t *testing.T) {
	e := MustParse("0 0 1 * *") // 1st of every month at midnight
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	next, err := e.Next(now)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	expected := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("Next = %v, want %v", next, expected)
	}
}

func TestNext_YearTransition(t *testing.T) {
	e := MustParse("0 0 1 1 *") // Jan 1st
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	next, err := e.Next(now)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	expected := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("Next = %v, want %v", next, expected)
	}
}

func TestNext_Weekday(t *testing.T) {
	e := MustParse("0 0 * * 1")                          // every Monday at midnight
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC) // Saturday
	next, err := e.Next(now)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if next.Weekday() != time.Monday {
		t.Fatalf("Next weekday = %v, want Monday", next.Weekday())
	}
}

func TestNext_DOMAndDOW(t *testing.T) {
	// Both DOM and DOW restricted: match if EITHER matches.
	e := MustParse("0 0 15 * 1")                         // 15th OR Monday at midnight
	now := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC) // Monday
	next, err := e.Next(now)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	// Aug 15 (Saturday) matches DOM=15, comes before next Monday (Aug 17).
	if next.Day() != 15 {
		t.Fatalf("Next day = %d, want 15 (DOM=15 matches before next Monday)", next.Day())
	}
}

func TestNext_LastDOM(t *testing.T) {
	e := MustParse("0 0 L * *") // last day of month at midnight
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	next, err := e.Next(now)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if next.Day() != 31 {
		t.Fatalf("Next day = %d, want 31 (last day of August)", next.Day())
	}
}

func TestNext_LastDOW(t *testing.T) {
	e := MustParse("0 0 * * 5L") // last Friday at midnight
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	next, err := e.Next(now)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	// Last Friday of August 2026 is August 28.
	if next.Day() != 28 || next.Weekday() != time.Friday {
		t.Fatalf("Next = %v (weekday=%v), want Aug 28 Friday", next, next.Weekday())
	}
}

func TestNext_EveryDuration(t *testing.T) {
	e := MustParse("@every 30s")
	now := time.Date(2026, 8, 15, 10, 0, 15, 0, time.UTC)
	next, err := e.Next(now)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	expected := time.Date(2026, 8, 15, 10, 0, 30, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("Next = %v, want %v", next, expected)
	}
}

func TestNext_ZeroTime(t *testing.T) {
	e := MustParse("*/5 * * * *")
	next, err := e.Next(time.Time{})
	if err != nil {
		t.Fatalf("Next with zero time failed: %v", err)
	}
	if next.IsZero() {
		t.Fatal("Next should not be zero")
	}
}

// ──────────────────────────────────────────────
// Schedule
// ──────────────────────────────────────────────

func TestSchedule(t *testing.T) {
	e := MustParse("*/5 * * * *")
	s := NewSchedule(e)

	next1, err := s.Next()
	if err != nil {
		t.Fatalf("Schedule.Next failed: %v", err)
	}
	next2, err := s.Next()
	if err != nil {
		t.Fatalf("Schedule.Next 2 failed: %v", err)
	}
	if !next2.After(next1) {
		t.Fatalf("Schedule.Next 2 (%v) should be after Next 1 (%v)", next2, next1)
	}
}

func TestSchedule_Reset(t *testing.T) {
	e := MustParse("*/5 * * * *")
	s := NewSchedule(e)
	_, _ = s.Next()
	s.Reset()
	// After reset, next should be based on now.
	next, err := s.Next()
	if err != nil {
		t.Fatalf("Schedule.Next after Reset failed: %v", err)
	}
	if next.IsZero() {
		t.Fatal("Next after Reset should not be zero")
	}
}

// ──────────────────────────────────────────────
// Validate
// ──────────────────────────────────────────────

func TestValidate_Valid(t *testing.T) {
	if err := Validate("*/5 * * * *"); err != nil {
		t.Fatalf("Validate valid expr failed: %v", err)
	}
}

func TestValidate_Invalid(t *testing.T) {
	if err := Validate("invalid"); err == nil {
		t.Fatal("Validate invalid expr should error")
	}
}

// ──────────────────────────────────────────────
// String
// ──────────────────────────────────────────────

func TestExpression_String(t *testing.T) {
	e := MustParse("*/5 * * * *")
	if e.String() != "*/5 * * * *" {
		t.Fatalf("String = %q", e.String())
	}
}
