// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package timeutil

import (
	"sync/atomic"
	"testing"
	"time"
)

// ──────────────────────────────────────────────
// Timezone helpers
// ──────────────────────────────────────────────

func TestLocation(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"", false},
		{"Local", false},
		{"UTC", false},
		{"Asia/Shanghai", false},
		{"America/New_York", false},
		{"Invalid/Zone", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := Location(tt.name)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Location(%q) expected error, got nil", tt.name)
				}
			} else {
				if err != nil {
					t.Fatalf("Location(%q) unexpected error: %v", tt.name, err)
				}
				if loc == nil {
					t.Fatalf("Location(%q) returned nil", tt.name)
				}
			}
		})
	}
}

func TestMustLocation(t *testing.T) {
	loc := MustLocation("Asia/Shanghai")
	if loc == nil {
		t.Fatal("MustLocation returned nil")
	}
}

func TestMustLocation_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustLocation with invalid zone should panic")
		}
	}()
	MustLocation("Invalid/Zone")
}

func TestIn(t *testing.T) {
	now := time.Now()
	loc, _ := Location("Asia/Shanghai")
	result := In(now, loc)
	if result.Location() != loc {
		t.Fatal("In did not convert timezone")
	}
}

func TestInLocation(t *testing.T) {
	now := time.Now()
	result, err := InLocation(now, "Asia/Shanghai")
	if err != nil {
		t.Fatalf("InLocation failed: %v", err)
	}
	if result.Location().String() != "Asia/Shanghai" {
		t.Fatalf("InLocation timezone = %q, want Asia/Shanghai", result.Location())
	}
}

func TestInLocation_Error(t *testing.T) {
	_, err := InLocation(time.Now(), "Invalid/Zone")
	if err == nil {
		t.Fatal("InLocation with invalid zone should error")
	}
}

func TestMustInLocation(t *testing.T) {
	result := MustInLocation(time.Now(), "UTC")
	if result.Location() != time.UTC {
		t.Fatal("MustInLocation UTC failed")
	}
}

// ──────────────────────────────────────────────
// Formatting & parsing
// ──────────────────────────────────────────────

func TestFormat(t *testing.T) {
	tm := time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)
	result := Format(tm, LayoutDateTime)
	if result != "2026-08-15 10:30:00" {
		t.Fatalf("Format = %q, want %q", result, "2026-08-15 10:30:00")
	}
}

func TestFormatIn(t *testing.T) {
	tm := time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)
	result := FormatIn(tm, LayoutDateTime, "Asia/Shanghai")
	if result != "2026-08-15 18:30:00" {
		t.Fatalf("FormatIn = %q, want %q", result, "2026-08-15 18:30:00")
	}
}

func TestFormatUTC(t *testing.T) {
	tm := time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)
	result := FormatUTC(tm, LayoutDateTime)
	if result != "2026-08-15 10:30:00" {
		t.Fatalf("FormatUTC = %q", result)
	}
}

func TestParse(t *testing.T) {
	tm, err := Parse("2026-08-15 10:30:00", LayoutDateTime)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if tm.Year() != 2026 || tm.Month() != 8 || tm.Day() != 15 {
		t.Fatalf("Parse result = %v", tm)
	}
}

func TestParseIn(t *testing.T) {
	tm, err := ParseIn("2026-08-15 10:30:00", LayoutDateTime, "Asia/Shanghai")
	if err != nil {
		t.Fatalf("ParseIn failed: %v", err)
	}
	// 10:30 Shanghai = 02:30 UTC
	if tm.UTC().Hour() != 2 {
		t.Fatalf("ParseIn UTC hour = %d, want 2", tm.UTC().Hour())
	}
}

func TestParseIn_Error(t *testing.T) {
	_, err := ParseIn("invalid", LayoutDateTime, "Asia/Shanghai")
	if err == nil {
		t.Fatal("ParseIn with invalid value should error")
	}
	_, err = ParseIn("2026-08-15 10:30:00", LayoutDateTime, "Invalid/Zone")
	if err == nil {
		t.Fatal("ParseIn with invalid zone should error")
	}
}

func TestParseLocal(t *testing.T) {
	tm, err := ParseLocal("2026-08-15 10:30:00", LayoutDateTime)
	if err != nil {
		t.Fatalf("ParseLocal failed: %v", err)
	}
	if tm.Location() != time.Local {
		t.Fatalf("ParseLocal location = %v, want Local", tm.Location())
	}
}

func TestParseUTC(t *testing.T) {
	tm, err := ParseUTC("2026-08-15 10:30:00", LayoutDateTime)
	if err != nil {
		t.Fatalf("ParseUTC failed: %v", err)
	}
	if tm.Location() != time.UTC {
		t.Fatalf("ParseUTC location = %v, want UTC", tm.Location())
	}
}

func TestMustParse(t *testing.T) {
	tm := MustParse("2026-08-15", LayoutDate)
	if tm.Year() != 2026 {
		t.Fatalf("MustParse year = %d", tm.Year())
	}
}

func TestMustParse_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustParse with invalid value should panic")
		}
	}()
	MustParse("invalid", LayoutDate)
}

func TestFormatTimestamp(t *testing.T) {
	tm := time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)
	result := FormatTimestamp(tm)
	if result != "20260815103000" {
		t.Fatalf("FormatTimestamp = %q", result)
	}
}

func TestFormatDate(t *testing.T) {
	tm := time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)
	if FormatDate(tm) != "2026-08-15" {
		t.Fatalf("FormatDate = %q", FormatDate(tm))
	}
}

func TestFormatDateTime(t *testing.T) {
	tm := time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)
	if FormatDateTime(tm) != "2026-08-15 10:30:00" {
		t.Fatalf("FormatDateTime = %q", FormatDateTime(tm))
	}
}

// ──────────────────────────────────────────────
// Time range helpers
// ──────────────────────────────────────────────

func TestStartOfDay(t *testing.T) {
	tm := time.Date(2026, 8, 15, 14, 30, 45, 123456, time.UTC)
	result := StartOfDay(tm)
	expected := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("StartOfDay = %v, want %v", result, expected)
	}
}

func TestEndOfDay(t *testing.T) {
	tm := time.Date(2026, 8, 15, 14, 30, 45, 0, time.UTC)
	result := EndOfDay(tm)
	if result.Day() != 15 || result.Hour() != 23 || result.Minute() != 59 {
		t.Fatalf("EndOfDay = %v", result)
	}
}

func TestStartOfHour(t *testing.T) {
	tm := time.Date(2026, 8, 15, 14, 30, 45, 0, time.UTC)
	result := StartOfHour(tm)
	if result.Minute() != 0 || result.Second() != 0 {
		t.Fatalf("StartOfHour = %v", result)
	}
}

func TestEndOfHour(t *testing.T) {
	tm := time.Date(2026, 8, 15, 14, 30, 45, 0, time.UTC)
	result := EndOfHour(tm)
	if result.Minute() != 59 || result.Second() != 59 {
		t.Fatalf("EndOfHour = %v", result)
	}
}

func TestStartOfMinute(t *testing.T) {
	tm := time.Date(2026, 8, 15, 14, 30, 45, 123, time.UTC)
	result := StartOfMinute(tm)
	if result.Second() != 0 || result.Nanosecond() != 0 {
		t.Fatalf("StartOfMinute = %v", result)
	}
}

func TestEndOfMinute(t *testing.T) {
	tm := time.Date(2026, 8, 15, 14, 30, 45, 0, time.UTC)
	result := EndOfMinute(tm)
	if result.Second() != 59 {
		t.Fatalf("EndOfMinute = %v", result)
	}
}

func TestStartOfWeek(t *testing.T) {
	// 2026-08-15 is a Saturday.
	tm := time.Date(2026, 8, 15, 14, 0, 0, 0, time.UTC)
	result := StartOfWeek(tm)
	// Should be Monday 2026-08-10.
	if result.Weekday() != time.Monday || result.Day() != 10 {
		t.Fatalf("StartOfWeek = %v (weekday=%v, day=%d), want Monday 10", result, result.Weekday(), result.Day())
	}
}

func TestStartOfWeek_Sunday(t *testing.T) {
	// 2026-08-16 is a Sunday.
	tm := time.Date(2026, 8, 16, 14, 0, 0, 0, time.UTC)
	result := StartOfWeek(tm)
	// Should be Monday 2026-08-10.
	if result.Weekday() != time.Monday || result.Day() != 10 {
		t.Fatalf("StartOfWeek(Sunday) = %v, want Monday 10", result)
	}
}

func TestEndOfWeek(t *testing.T) {
	tm := time.Date(2026, 8, 15, 14, 0, 0, 0, time.UTC)
	result := EndOfWeek(tm)
	// Should be Sunday 2026-08-16 23:59:59.
	if result.Weekday() != time.Sunday || result.Day() != 16 {
		t.Fatalf("EndOfWeek = %v, want Sunday 16", result)
	}
}

func TestStartOfMonth(t *testing.T) {
	tm := time.Date(2026, 8, 15, 14, 0, 0, 0, time.UTC)
	result := StartOfMonth(tm)
	if result.Day() != 1 || result.Month() != 8 {
		t.Fatalf("StartOfMonth = %v", result)
	}
}

func TestEndOfMonth(t *testing.T) {
	tm := time.Date(2026, 8, 15, 14, 0, 0, 0, time.UTC)
	result := EndOfMonth(tm)
	if result.Day() != 31 {
		t.Fatalf("EndOfMonth August day = %d, want 31", result.Day())
	}
}

func TestEndOfMonth_February(t *testing.T) {
	tm := time.Date(2026, 2, 15, 14, 0, 0, 0, time.UTC)
	result := EndOfMonth(tm)
	if result.Day() != 28 {
		t.Fatalf("EndOfMonth Feb 2026 day = %d, want 28", result.Day())
	}
}

func TestEndOfMonth_FebruaryLeap(t *testing.T) {
	tm := time.Date(2024, 2, 15, 14, 0, 0, 0, time.UTC)
	result := EndOfMonth(tm)
	if result.Day() != 29 {
		t.Fatalf("EndOfMonth Feb 2024 day = %d, want 29", result.Day())
	}
}

func TestStartOfYear(t *testing.T) {
	tm := time.Date(2026, 8, 15, 14, 0, 0, 0, time.UTC)
	result := StartOfYear(tm)
	if result.Month() != 1 || result.Day() != 1 {
		t.Fatalf("StartOfYear = %v", result)
	}
}

func TestEndOfYear(t *testing.T) {
	tm := time.Date(2026, 8, 15, 14, 0, 0, 0, time.UTC)
	result := EndOfYear(tm)
	if result.Month() != 12 || result.Day() != 31 {
		t.Fatalf("EndOfYear = %v", result)
	}
}

func TestDaysInMonth(t *testing.T) {
	tests := []struct {
		year, month, want int
	}{
		{2026, 1, 31},
		{2026, 2, 28},
		{2024, 2, 29},
		{2026, 4, 30},
		{2026, 12, 31},
	}
	for _, tt := range tests {
		tm := time.Date(tt.year, time.Month(tt.month), 15, 0, 0, 0, 0, time.UTC)
		if got := DaysInMonth(tm); got != tt.want {
			t.Fatalf("DaysInMonth(%d,%d) = %d, want %d", tt.year, tt.month, got, tt.want)
		}
	}
}

func TestIsLeapYear(t *testing.T) {
	tests := []struct {
		year int
		want bool
	}{
		{2024, true},
		{2026, false},
		{2000, true},
		{1900, false},
	}
	for _, tt := range tests {
		tm := time.Date(tt.year, 1, 1, 0, 0, 0, 0, time.UTC)
		if got := IsLeapYear(tm); got != tt.want {
			t.Fatalf("IsLeapYear(%d) = %v, want %v", tt.year, got, tt.want)
		}
	}
}

// ──────────────────────────────────────────────
// TimeRange
// ──────────────────────────────────────────────

func TestTimeRange_Duration(t *testing.T) {
	r := NewTimeRange(
		time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
	)
	if r.Duration() != 2*time.Hour {
		t.Fatalf("Duration = %v, want 2h", r.Duration())
	}
}

func TestTimeRange_Contains(t *testing.T) {
	r := NewTimeRange(
		time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
	)
	if !r.Contains(time.Date(2026, 8, 15, 11, 0, 0, 0, time.UTC)) {
		t.Fatal("Contains should be true for 11:00")
	}
	if r.Contains(time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("Contains should be false for 12:00 (half-open)")
	}
	if r.Contains(time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)) {
		t.Fatal("Contains should be false for 09:00")
	}
}

func TestTimeRange_ContainsInclusive(t *testing.T) {
	r := NewTimeRange(
		time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
	)
	if !r.ContainsInclusive(time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("ContainsInclusive should be true for 12:00")
	}
}

func TestTimeRange_Overlaps(t *testing.T) {
	r1 := NewTimeRange(
		time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
	)
	r2 := NewTimeRange(
		time.Date(2026, 8, 15, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 15, 13, 0, 0, 0, time.UTC),
	)
	if !r1.Overlaps(r2) {
		t.Fatal("Overlaps should be true")
	}
	r3 := NewTimeRange(
		time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 15, 14, 0, 0, 0, time.UTC),
	)
	if r1.Overlaps(r3) {
		t.Fatal("Overlaps should be false (adjacent, half-open)")
	}
}

func TestTimeRange_Intersect(t *testing.T) {
	r1 := NewTimeRange(
		time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 15, 14, 0, 0, 0, time.UTC),
	)
	r2 := NewTimeRange(
		time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 15, 16, 0, 0, 0, time.UTC),
	)
	intersect, ok := r1.Intersect(r2)
	if !ok {
		t.Fatal("Intersect should succeed")
	}
	if !intersect.Start.Equal(time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("Intersect start = %v", intersect.Start)
	}
	if !intersect.End.Equal(time.Date(2026, 8, 15, 14, 0, 0, 0, time.UTC)) {
		t.Fatalf("Intersect end = %v", intersect.End)
	}
}

func TestTimeRange_Intersect_NoOverlap(t *testing.T) {
	r1 := NewTimeRange(
		time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
	)
	r2 := NewTimeRange(
		time.Date(2026, 8, 15, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 15, 16, 0, 0, 0, time.UTC),
	)
	_, ok := r1.Intersect(r2)
	if ok {
		t.Fatal("Intersect should fail for non-overlapping ranges")
	}
}

func TestTimeRange_Split(t *testing.T) {
	r := NewTimeRange(
		time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
	)
	parts := r.Split(4)
	if len(parts) != 4 {
		t.Fatalf("Split(4) length = %d", len(parts))
	}
	if parts[0].Duration() != 30*time.Minute {
		t.Fatalf("Split part[0] duration = %v", parts[0].Duration())
	}
	// Last part should end at the range end.
	if !parts[3].End.Equal(r.End) {
		t.Fatalf("Split last end = %v, want %v", parts[3].End, r.End)
	}
}

func TestTimeRange_Split_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Split(0) should panic")
		}
	}()
	r := NewTimeRange(time.Now(), time.Now().Add(time.Hour))
	r.Split(0)
}

func TestTodayRange(t *testing.T) {
	r := TodayRange()
	now := time.Now()
	if !r.Contains(now) {
		t.Fatal("TodayRange should contain now")
	}
}

func TestYesterdayRange(t *testing.T) {
	r := YesterdayRange()
	now := time.Now()
	if r.Contains(now) {
		t.Fatal("YesterdayRange should not contain now")
	}
}

func TestThisWeekRange(t *testing.T) {
	r := ThisWeekRange()
	now := time.Now()
	if !r.Contains(now) {
		t.Fatal("ThisWeekRange should contain now")
	}
}

func TestThisMonthRange(t *testing.T) {
	r := ThisMonthRange()
	now := time.Now()
	if !r.Contains(now) {
		t.Fatal("ThisMonthRange should contain now")
	}
}

func TestThisYearRange(t *testing.T) {
	r := ThisYearRange()
	now := time.Now()
	if !r.Contains(now) {
		t.Fatal("ThisYearRange should contain now")
	}
}

// ──────────────────────────────────────────────
// Duration helpers
// ──────────────────────────────────────────────

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"1h30m", 90 * time.Minute},
		{"500ms", 500 * time.Millisecond},
		{"1d", 24 * time.Hour},
		{"3d", 72 * time.Hour},
		{"1w", 7 * 24 * time.Hour},
		{"2w", 14 * 24 * time.Hour},
		{"1d2h", 26 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDuration_Error(t *testing.T) {
	_, err := ParseDuration("")
	if err == nil {
		t.Fatal("ParseDuration(\"\") should error")
	}
	_, err = ParseDuration("invalid")
	if err == nil {
		t.Fatal("ParseDuration(\"invalid\") should error")
	}
}

func TestDurationToHuman(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0ns"},
		{500 * time.Millisecond, "500ms"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{2 * time.Hour, "2h"},
		{26 * time.Hour, "1d2h"},
		{7 * 24 * time.Hour, "1w"},
		{14 * 24 * time.Hour, "2w"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := DurationToHuman(tt.d)
			if got != tt.want {
				t.Fatalf("DurationToHuman(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestDurationToHuman_Negative(t *testing.T) {
	got := DurationToHuman(-30 * time.Second)
	if got != "-30s" {
		t.Fatalf("DurationToHuman(-30s) = %q, want -30s", got)
	}
}

// ──────────────────────────────────────────────
// Stopwatch
// ──────────────────────────────────────────────

func TestStopwatch(t *testing.T) {
	sw := NewStopwatch()
	if sw.IsRunning() {
		t.Fatal("new stopwatch should not be running")
	}
	sw.Start()
	if !sw.IsRunning() {
		t.Fatal("stopwatch should be running after Start")
	}
	time.Sleep(10 * time.Millisecond)
	elapsed := sw.Stop()
	if elapsed < 5*time.Millisecond {
		t.Fatalf("Elapsed = %v, want >= 5ms", elapsed)
	}
	if sw.IsRunning() {
		t.Fatal("stopwatch should not be running after Stop")
	}
}

func TestStopwatch_ElapsedWhileRunning(t *testing.T) {
	sw := NewStopwatch()
	sw.Start()
	time.Sleep(10 * time.Millisecond)
	elapsed := sw.Elapsed()
	if elapsed < 5*time.Millisecond {
		t.Fatalf("Elapsed while running = %v", elapsed)
	}
	_ = sw.Stop()
}

func TestStopwatch_Reset(t *testing.T) {
	sw := NewStopwatch()
	sw.Start()
	time.Sleep(10 * time.Millisecond)
	sw.Stop()
	sw.Reset()
	if sw.Elapsed() != 0 {
		t.Fatalf("Elapsed after Reset = %v, want 0", sw.Elapsed())
	}
}

func TestStopwatch_Lap(t *testing.T) {
	sw := NewStopwatch()
	sw.Start()
	time.Sleep(10 * time.Millisecond)
	lap1 := sw.Lap()
	if lap1 < 5*time.Millisecond {
		t.Fatalf("Lap1 = %v, want >= 5ms", lap1)
	}
	time.Sleep(10 * time.Millisecond)
	lap2 := sw.Lap()
	if lap2 < 5*time.Millisecond {
		t.Fatalf("Lap2 = %v, want >= 5ms", lap2)
	}
	laps := sw.Laps()
	if len(laps) != 2 {
		t.Fatalf("Laps count = %d, want 2", len(laps))
	}
}

func TestStopwatch_LapNotRunning(t *testing.T) {
	sw := NewStopwatch()
	if sw.Lap() != 0 {
		t.Fatal("Lap when not running should return 0")
	}
}

func TestStopwatch_StartTwice(t *testing.T) {
	sw := NewStopwatch()
	sw.Start()
	sw.Start() // should be no-op
	if !sw.IsRunning() {
		t.Fatal("should still be running")
	}
	sw.Stop()
}

// ──────────────────────────────────────────────
// Countdown
// ──────────────────────────────────────────────

func TestCountdown(t *testing.T) {
	fired := false
	c := NewCountdown(50*time.Millisecond, func() {
		fired = true
	})
	<-c.Done()
	if !fired {
		t.Fatal("countdown did not fire")
	}
}

func TestCountdown_Stop(t *testing.T) {
	fired := false
	c := NewCountdown(1*time.Second, func() {
		fired = true
	})
	if !c.Stop() {
		t.Fatal("Stop should return true")
	}
	time.Sleep(50 * time.Millisecond)
	if fired {
		t.Fatal("countdown fired after Stop")
	}
}

// ──────────────────────────────────────────────
// Debounce
// ──────────────────────────────────────────────

func TestDebounce(t *testing.T) {
	var count int64
	debounced := Debounce(50*time.Millisecond, func() {
		atomic.AddInt64(&count, 1)
	})
	// Call multiple times rapidly.
	debounced()
	debounced()
	debounced()
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt64(&count) != 1 {
		t.Fatalf("debounce count = %d, want 1", atomic.LoadInt64(&count))
	}
}

func TestDebounce_OnlyLast(t *testing.T) {
	var count int64
	debounced := Debounce(50*time.Millisecond, func() {
		atomic.AddInt64(&count, 1)
	})
	debounced()
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt64(&count) != 1 {
		t.Fatalf("count = %d, want 1", atomic.LoadInt64(&count))
	}
	debounced()
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt64(&count) != 2 {
		t.Fatalf("count = %d, want 2", atomic.LoadInt64(&count))
	}
}

// ──────────────────────────────────────────────
// Business day calculations
// ──────────────────────────────────────────────

func TestIsBusinessDay(t *testing.T) {
	tests := []struct {
		date string
		want bool
	}{
		{"2026-08-14", true},  // Friday
		{"2026-08-15", false}, // Saturday
		{"2026-08-16", false}, // Sunday
		{"2026-08-17", true},  // Monday
		{"2026-08-21", true},  // Friday
	}
	for _, tt := range tests {
		tm := MustParse(tt.date, LayoutDate)
		if got := IsBusinessDay(tm); got != tt.want {
			t.Fatalf("IsBusinessDay(%s) = %v, want %v", tt.date, got, tt.want)
		}
	}
}

func TestIsWeekend(t *testing.T) {
	sat := MustParse("2026-08-15", LayoutDate) // Saturday
	if !IsWeekend(sat) {
		t.Fatal("Saturday should be weekend")
	}
	mon := MustParse("2026-08-17", LayoutDate) // Monday
	if IsWeekend(mon) {
		t.Fatal("Monday should not be weekend")
	}
}

func TestAddBusinessDays(t *testing.T) {
	// Friday 2026-08-14 + 1 business day = Monday 2026-08-17.
	fri := MustParse("2026-08-14", LayoutDate)
	result := AddBusinessDays(fri, 1)
	if result.Weekday() != time.Monday || result.Day() != 17 {
		t.Fatalf("AddBusinessDays(Fri, 1) = %v, want Mon 17", result)
	}
	// Monday + 5 business days = next Monday.
	mon := MustParse("2026-08-17", LayoutDate)
	result = AddBusinessDays(mon, 5)
	if result.Weekday() != time.Monday || result.Day() != 24 {
		t.Fatalf("AddBusinessDays(Mon, 5) = %v, want Mon 24", result)
	}
}

func TestAddBusinessDays_Negative(t *testing.T) {
	// Monday 2026-08-17 - 1 business day = Friday 2026-08-14.
	mon := MustParse("2026-08-17", LayoutDate)
	result := AddBusinessDays(mon, -1)
	if result.Weekday() != time.Friday || result.Day() != 14 {
		t.Fatalf("AddBusinessDays(Mon, -1) = %v, want Fri 14", result)
	}
}

func TestNextBusinessDay(t *testing.T) {
	fri := MustParse("2026-08-14", LayoutDate)
	result := NextBusinessDay(fri)
	if result.Weekday() != time.Monday {
		t.Fatalf("NextBusinessDay(Fri) = %v, want Monday", result)
	}
}

func TestPreviousBusinessDay(t *testing.T) {
	mon := MustParse("2026-08-17", LayoutDate)
	result := PreviousBusinessDay(mon)
	if result.Weekday() != time.Friday {
		t.Fatalf("PreviousBusinessDay(Mon) = %v, want Friday", result)
	}
}

func TestBusinessDaysBetween(t *testing.T) {
	start := MustParse("2026-08-17", LayoutDate) // Monday
	end := MustParse("2026-08-22", LayoutDate)   // Saturday
	count := BusinessDaysBetween(start, end)
	// Mon→Tue, Wed, Thu, Fri = 4 business days between.
	if count != 4 {
		t.Fatalf("BusinessDaysBetween = %d, want 4", count)
	}
}

func TestBusinessDaysBetween_NoOverlap(t *testing.T) {
	start := MustParse("2026-08-22", LayoutDate)
	end := MustParse("2026-08-17", LayoutDate)
	if BusinessDaysBetween(start, end) != 0 {
		t.Fatal("BusinessDaysBetween with start >= end should be 0")
	}
}

// ──────────────────────────────────────────────
// Misc helpers
// ──────────────────────────────────────────────

func TestNow(t *testing.T) {
	n := Now()
	if n.IsZero() {
		t.Fatal("Now() is zero")
	}
}

func TestNowUTC(t *testing.T) {
	n := NowUTC()
	if n.Location() != time.UTC {
		t.Fatalf("NowUTC location = %v", n.Location())
	}
}

func TestUnixNow(t *testing.T) {
	n := UnixNow()
	if n == 0 {
		t.Fatal("UnixNow() is 0")
	}
}

func TestMillisNow(t *testing.T) {
	n := MillisNow()
	if n == 0 {
		t.Fatal("MillisNow() is 0")
	}
}

func TestMicrosNow(t *testing.T) {
	n := MicrosNow()
	if n == 0 {
		t.Fatal("MicrosNow() is 0")
	}
}

func TestFromUnix(t *testing.T) {
	tm := FromUnix(0)
	if !tm.Equal(time.Unix(0, 0)) {
		t.Fatal("FromUnix(0) mismatch")
	}
}

func TestFromMillis(t *testing.T) {
	tm := FromMillis(1000)
	if tm.Unix() != 1 {
		t.Fatalf("FromMillis(1000) Unix = %d, want 1", tm.Unix())
	}
}

func TestFromMicros(t *testing.T) {
	tm := FromMicros(1000000)
	if tm.Unix() != 1 {
		t.Fatalf("FromMicros(1000000) Unix = %d, want 1", tm.Unix())
	}
}

func TestAge(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	age := Age(past)
	if age < 59*time.Minute || age > 61*time.Minute {
		t.Fatalf("Age = %v, want ~1h", age)
	}
}

func TestIsExpired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	if !IsExpired(past) {
		t.Fatal("IsExpired(past) should be true")
	}
	future := time.Now().Add(time.Hour)
	if IsExpired(future) {
		t.Fatal("IsExpired(future) should be false")
	}
}

func TestIsZero(t *testing.T) {
	if !IsZero(time.Time{}) {
		t.Fatal("IsZero(zero) should be true")
	}
	if IsZero(time.Now()) {
		t.Fatal("IsZero(now) should be false")
	}
}

func TestClamp(t *testing.T) {
	start := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 15, 20, 0, 0, 0, time.UTC)
	before := time.Date(2026, 8, 15, 5, 0, 0, 0, time.UTC)
	after := time.Date(2026, 8, 15, 23, 0, 0, 0, time.UTC)
	middle := time.Date(2026, 8, 15, 15, 0, 0, 0, time.UTC)

	if !Clamp(before, start, end).Equal(start) {
		t.Fatal("Clamp(before) should return start")
	}
	if !Clamp(after, start, end).Equal(end) {
		t.Fatal("Clamp(after) should return end")
	}
	if !Clamp(middle, start, end).Equal(middle) {
		t.Fatal("Clamp(middle) should return middle")
	}
}

func TestMax(t *testing.T) {
	a := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	b := time.Date(2026, 8, 15, 20, 0, 0, 0, time.UTC)
	if !Max(a, b).Equal(b) {
		t.Fatal("Max should return later time")
	}
}

func TestMin(t *testing.T) {
	a := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	b := time.Date(2026, 8, 15, 20, 0, 0, 0, time.UTC)
	if !Min(a, b).Equal(a) {
		t.Fatal("Min should return earlier time")
	}
}
