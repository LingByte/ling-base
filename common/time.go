// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Natural-language time parsing (formerly common/nltime).
//
// Time zone: entirely controlled by the `now` argument (or ParseNaturalTimeInLocation).
// Week semantics: "this <weekday>" uses WeekStart (default Monday) as the week boundary.
// Calendar month/year: relative month/year use AddDate — never a fixed 30/365 Duration.

var (
	// ErrEmptyNaturalTime is returned for empty expressions.
	ErrEmptyNaturalTime = errors.New("common: empty time expression")
	// ErrUnsupportedNaturalTime is returned when the expression cannot be parsed.
	ErrUnsupportedNaturalTime = errors.New("common: unsupported time expression")
	// ErrEmptyNaturalDuration is returned for empty duration strings.
	ErrEmptyNaturalDuration = errors.New("common: empty duration")
	// ErrUnsupportedNaturalDuration is returned when a duration cannot be parsed.
	ErrUnsupportedNaturalDuration = errors.New("common: unsupported duration")
)

// WeekStart defines the first day of the week for "this <weekday>" / "this weekend".
// Default is Monday (common in CN/ISO). Set to time.Sunday for US-style weeks.
var WeekStart = time.Monday

// Daypart hours for "this morning/afternoon/evening" (local to `now`).
var (
	MorningHour   = 8
	AfternoonHour = 14
	EveningHour   = 18
)

// TimeRange is defined in time_helpers.go (inclusive-start / exclusive-end window).

// ParseNaturalTime parses a natural-language time expression relative to now.
//
// Match priority (first hit wins):
//  1. keywords (now/today/…)
//  2. relative (3 days ago / in 2 hours / 3 months ago via AddDate)
//  3. weekday refs (next/last/this monday|weekend)
//  4. combined date+time (tomorrow at 3pm, next monday at 10:00)
//  5. time-of-day (3pm, 10:30)
func ParseNaturalTime(expr string, now time.Time) (time.Time, error) {
	expr = strings.TrimSpace(strings.ToLower(expr))
	if expr == "" {
		return time.Time{}, ErrEmptyNaturalTime
	}
	if now.IsZero() {
		now = time.Now()
	}

	if t, ok := parseNLKeyword(expr, now); ok {
		return t, nil
	}
	if t, ok := parseNLRelative(expr, now); ok {
		return t, nil
	}
	if t, ok := parseNLWeekdayRef(expr, now); ok {
		return t, nil
	}
	if t, ok := parseNLCombined(expr, now); ok {
		return t, nil
	}
	if t, ok := parseNLTimeOfDay(expr, now); ok {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("%w: %q", ErrUnsupportedNaturalTime, expr)
}

// ParseNaturalTimeInLocation parses expr relative to time.Now() in loc.
func ParseNaturalTimeInLocation(expr string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.Local
	}
	return ParseNaturalTime(expr, time.Now().In(loc))
}

// MustParseNaturalTime is like ParseNaturalTime but panics on error.
func MustParseNaturalTime(expr string, now time.Time) time.Time {
	t, err := ParseNaturalTime(expr, now)
	if err != nil {
		panic(err)
	}
	return t
}

// ParseNaturalRange parses window expressions such as:
//
//	"last 7 days", "last 2 weeks", "this week", "this month", "this year"
//
// End is exclusive (typically `now` or start of next period).
func ParseNaturalRange(expr string, now time.Time) (TimeRange, error) {
	expr = strings.TrimSpace(strings.ToLower(expr))
	if expr == "" {
		return TimeRange{}, ErrEmptyNaturalTime
	}
	if now.IsZero() {
		now = time.Now()
	}

	if m := lastNRe.FindStringSubmatch(expr); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil || n <= 0 {
			return TimeRange{}, fmt.Errorf("%w: %q", ErrUnsupportedNaturalTime, expr)
		}
		unit := normalizeNLUnit(m[2])
		start, ok := shiftCalendar(now, -n, unit)
		if !ok {
			return TimeRange{}, fmt.Errorf("%w: %q", ErrUnsupportedNaturalTime, expr)
		}
		return TimeRange{Start: start, End: now}, nil
	}

	switch expr {
	case "this week":
		start := thisWeekday(now, WeekStart)
		return TimeRange{Start: start, End: now}, nil
	case "this month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return TimeRange{Start: start, End: now}, nil
	case "this year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return TimeRange{Start: start, End: now}, nil
	case "today":
		start := startOfNLDay(now)
		return TimeRange{Start: start, End: now}, nil
	}
	return TimeRange{}, fmt.Errorf("%w: %q", ErrUnsupportedNaturalTime, expr)
}

func parseNLKeyword(expr string, now time.Time) (time.Time, bool) {
	switch expr {
	case "now":
		return now, true
	case "today":
		return startOfNLDay(now), true
	case "yesterday":
		return startOfNLDay(now.AddDate(0, 0, -1)), true
	case "tomorrow":
		return startOfNLDay(now.AddDate(0, 0, 1)), true
	case "noon":
		return time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location()), true
	case "midnight":
		return startOfNLDay(now), true
	case "this morning":
		return time.Date(now.Year(), now.Month(), now.Day(), MorningHour, 0, 0, 0, now.Location()), true
	case "this afternoon":
		return time.Date(now.Year(), now.Month(), now.Day(), AfternoonHour, 0, 0, 0, now.Location()), true
	case "this evening":
		return time.Date(now.Year(), now.Month(), now.Day(), EveningHour, 0, 0, 0, now.Location()), true
	}
	return time.Time{}, false
}

// relative: optional "in ", number, unit, optional ago/from now/later/...
// Bare "3 days" is treated as future (same as "in 3 days").
var (
	relativeRe = regexp.MustCompile(`^(?:in\s+)?(\d+)\s*([a-z]+)(?:\s+(ago|from\s+now|later|from\s+today|back))?$`)
	lastNRe    = regexp.MustCompile(`^last\s+(\d+)\s*([a-z]+)$`)
)

func parseNLRelative(expr string, now time.Time) (time.Time, bool) {
	m := relativeRe.FindStringSubmatch(expr)
	if m == nil {
		return time.Time{}, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 0 {
		return time.Time{}, false
	}
	unit := normalizeNLUnit(m[2])
	suffix := m[3]

	hasIn := strings.HasPrefix(expr, "in ")
	hasAgo := suffix == "ago" || suffix == "back"
	if hasIn && hasAgo {
		return time.Time{}, false // "in 3 months ago" is illegal
	}
	past := hasAgo

	switch unit {
	case "year", "month", "day", "week", "hour", "minute", "second":
		if past {
			t, ok := shiftCalendar(now, -n, unit)
			return t, ok
		}
		t, ok := shiftCalendar(now, n, unit)
		return t, ok
	default:
		return time.Time{}, false
	}
}

// shiftCalendar applies signed calendar/duration shifts.
// year/month use AddDate; week/day/hour/minute/second use Duration.
func shiftCalendar(now time.Time, n int, unit string) (time.Time, bool) {
	switch unit {
	case "year":
		return now.AddDate(n, 0, 0), true
	case "month":
		return now.AddDate(0, n, 0), true
	case "week":
		return now.AddDate(0, 0, 7*n), true
	case "day":
		return now.AddDate(0, 0, n), true
	case "hour":
		return now.Add(time.Duration(n) * time.Hour), true
	case "minute":
		return now.Add(time.Duration(n) * time.Minute), true
	case "second":
		return now.Add(time.Duration(n) * time.Second), true
	}
	return time.Time{}, false
}

func normalizeNLUnit(u string) string {
	switch strings.ToLower(u) {
	case "s", "sec", "secs", "second", "seconds":
		return "second"
	case "m", "min", "mins", "minute", "minutes":
		return "minute"
	case "h", "hr", "hrs", "hour", "hours":
		return "hour"
	case "d", "day", "days":
		return "day"
	case "w", "wk", "wks", "week", "weeks":
		return "week"
	case "mo", "mos", "month", "months":
		return "month"
	case "y", "yr", "yrs", "year", "years":
		return "year"
	}
	return u
}

var weekdayRefRe = regexp.MustCompile(`^(next|last|this|previous)\s+(\w+)$`)

func parseNLWeekdayRef(expr string, now time.Time) (time.Time, bool) {
	m := weekdayRefRe.FindStringSubmatch(expr)
	if m == nil {
		return time.Time{}, false
	}
	dir := m[1]
	name := m[2]

	if name == "weekend" {
		t, ok := weekendPoint(now, dir)
		return t, ok
	}
	wd, ok := parseNLWeekdayName(name)
	if !ok {
		return time.Time{}, false
	}
	switch dir {
	case "next":
		return nextNLWeekday(now, wd), true
	case "last", "previous":
		return lastNLWeekday(now, wd), true
	case "this":
		return thisWeekday(now, wd), true
	}
	return time.Time{}, false
}

func parseNLWeekdayName(s string) (time.Weekday, bool) {
	weekdays := map[string]time.Weekday{
		"sunday": time.Sunday, "sun": time.Sunday,
		"monday": time.Monday, "mon": time.Monday,
		"tuesday": time.Tuesday, "tue": time.Tuesday, "tues": time.Tuesday,
		"wednesday": time.Wednesday, "wed": time.Wednesday,
		"thursday": time.Thursday, "thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday,
		"friday": time.Friday, "fri": time.Friday,
		"saturday": time.Saturday, "sat": time.Saturday,
	}
	wd, ok := weekdays[s]
	return wd, ok
}

func nextNLWeekday(now time.Time, wd time.Weekday) time.Time {
	daysAhead := (int(wd) - int(now.Weekday()) + 7) % 7
	if daysAhead == 0 {
		daysAhead = 7
	}
	return startOfNLDay(now.AddDate(0, 0, daysAhead))
}

func lastNLWeekday(now time.Time, wd time.Weekday) time.Time {
	daysBack := (int(now.Weekday()) - int(wd) + 7) % 7
	if daysBack == 0 {
		daysBack = 7
	}
	return startOfNLDay(now.AddDate(0, 0, -daysBack))
}

// thisWeekday returns wd in the current week defined by WeekStart.
// If today is that weekday, returns today (start of day).
func thisWeekday(now time.Time, wd time.Weekday) time.Time {
	toOffset := func(d time.Weekday) int {
		return (int(d) - int(WeekStart) + 7) % 7
	}
	delta := toOffset(wd) - toOffset(now.Weekday())
	return startOfNLDay(now.AddDate(0, 0, delta))
}

// weekendPoint returns the Saturday that anchors the weekend for dir.
// "this weekend" → Saturday of the current WeekStart-based week.
func weekendPoint(now time.Time, dir string) (time.Time, bool) {
	switch dir {
	case "next":
		return nextNLWeekday(now, time.Saturday), true
	case "last", "previous":
		return lastNLWeekday(now, time.Saturday), true
	case "this":
		return thisWeekday(now, time.Saturday), true
	}
	return time.Time{}, false
}

// Combined: "<dateExpr> at <time>" or "<weekdayRef> <time>" (no "at").
func parseNLCombined(expr string, now time.Time) (time.Time, bool) {
	var datePart, timePart string
	if i := strings.LastIndex(expr, " at "); i >= 0 {
		datePart = strings.TrimSpace(expr[:i])
		timePart = strings.TrimSpace(expr[i+4:])
	} else {
		// "next monday 10:00" / "tomorrow 3pm"
		fields := strings.Fields(expr)
		if len(fields) < 2 {
			return time.Time{}, false
		}
		timePart = fields[len(fields)-1]
		datePart = strings.Join(fields[:len(fields)-1], " ")
		if _, _, ok := parseNLTimeString(timePart); !ok {
			return time.Time{}, false
		}
	}

	base, ok := resolveNLDatePart(datePart, now)
	if !ok {
		return time.Time{}, false
	}
	hour, minute, ok := parseNLTimeString(timePart)
	if !ok {
		return time.Time{}, false
	}
	return time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, now.Location()), true
}

func resolveNLDatePart(datePart string, now time.Time) (time.Time, bool) {
	if t, ok := parseNLKeyword(datePart, now); ok {
		return t, true
	}
	if t, ok := parseNLWeekdayRef(datePart, now); ok {
		return t, true
	}
	return time.Time{}, false
}

func parseNLTimeOfDay(expr string, now time.Time) (time.Time, bool) {
	hour, minute, ok := parseNLTimeString(expr)
	if !ok {
		return time.Time{}, false
	}
	return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()), true
}

var timeOfDayRe = regexp.MustCompile(`^(\d{1,2})(?::(\d{2}))?\s*(am|pm)?$`)

func parseNLTimeString(s string) (int, int, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "noon" {
		return 12, 0, true
	}
	if s == "midnight" {
		return 0, 0, true
	}
	m := timeOfDayRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	hour, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, false
	}
	minute := 0
	if m[2] != "" {
		minute, err = strconv.Atoi(m[2])
		if err != nil || minute < 0 || minute > 59 {
			return 0, 0, false
		}
	}
	ampm := m[3]
	if ampm != "" {
		// 12-hour clock: hour must be 1..12
		if hour < 1 || hour > 12 {
			return 0, 0, false
		}
		if ampm == "pm" && hour < 12 {
			hour += 12
		} else if ampm == "am" && hour == 12 {
			hour = 0
		}
	} else if hour < 0 || hour > 23 {
		return 0, 0, false
	}
	return hour, minute, true
}

var durationPartRe = regexp.MustCompile(`(\d+)\s*([a-z]+)`)

// ParseNaturalDuration parses compound durations like "2 hours 30 minutes" or "1d 2h".
// Month/year units are rejected (Duration cannot represent calendar months/years).
func ParseNaturalDuration(expr string) (time.Duration, error) {
	expr = strings.ToLower(strings.TrimSpace(expr))
	if expr == "" {
		return 0, ErrEmptyNaturalDuration
	}
	if d, err := time.ParseDuration(expr); err == nil {
		return d, nil
	}
	matches := durationPartRe.FindAllStringSubmatch(expr, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("%w: %q", ErrUnsupportedNaturalDuration, expr)
	}
	var total time.Duration
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, fmt.Errorf("%w: invalid number %q", ErrUnsupportedNaturalDuration, m[1])
		}
		unit := normalizeNLUnit(m[2])
		switch unit {
		case "month", "year":
			return 0, fmt.Errorf("%w: calendar %s cannot be a Duration; use ParseNaturalTime", ErrUnsupportedNaturalDuration, unit)
		case "second":
			total += time.Duration(n) * time.Second
		case "minute":
			total += time.Duration(n) * time.Minute
		case "hour":
			total += time.Duration(n) * time.Hour
		case "day":
			total += time.Duration(n) * 24 * time.Hour
		case "week":
			total += time.Duration(n) * 7 * 24 * time.Hour
		default:
			return 0, fmt.Errorf("%w: unknown unit %q", ErrUnsupportedNaturalDuration, m[2])
		}
	}
	return total, nil
}

// MustParseNaturalDuration panics on error.
func MustParseNaturalDuration(expr string) time.Duration {
	d, err := ParseNaturalDuration(expr)
	if err != nil {
		panic(err)
	}
	return d
}

// ShiftNatural applies a signed relative expression to t (e.g. "-3 months", "2 weeks").
func ShiftNatural(t time.Time, expr string) (time.Time, error) {
	expr = strings.TrimSpace(strings.ToLower(expr))
	if expr == "" {
		return time.Time{}, ErrEmptyNaturalTime
	}
	sign := 1
	if strings.HasPrefix(expr, "+") {
		expr = strings.TrimSpace(expr[1:])
	} else if strings.HasPrefix(expr, "-") {
		sign = -1
		expr = strings.TrimSpace(expr[1:])
	}
	m := regexp.MustCompile(`^(\d+)\s*([a-z]+)$`).FindStringSubmatch(expr)
	if m == nil {
		return time.Time{}, fmt.Errorf("%w: %q", ErrUnsupportedNaturalTime, expr)
	}
	n, _ := strconv.Atoi(m[1])
	unit := normalizeNLUnit(m[2])
	out, ok := shiftCalendar(t, sign*n, unit)
	if !ok {
		return time.Time{}, fmt.Errorf("%w: %q", ErrUnsupportedNaturalTime, expr)
	}
	return out, nil
}

func startOfNLDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
