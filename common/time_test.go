// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 2026-08-15 10:30 UTC is a Saturday.
var refSat = time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)

// 2026-03-31 → 3 months ago should be 2025-12-31 via AddDate.
func TestParseNaturalTime_MonthsAddDate(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	got, err := ParseNaturalTime("3 months ago", now)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2025, 12, 31, 12, 0, 0, 0, time.UTC), got)

	got, err = ParseNaturalTime("in 1 month", now)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC), got) // Mar31 +1mo → May1 in Go
}

func TestParseNaturalTime_LeapYear(t *testing.T) {
	// Go AddDate normalizes 2023-02-29 → 2023-03-01 (not clamp to Feb 28).
	now := time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC)
	got, err := ParseNaturalTime("1 year ago", now)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2023, 3, 1, 12, 0, 0, 0, time.UTC), got)
}

func TestParseNaturalTime_InAgoMutex(t *testing.T) {
	_, err := ParseNaturalTime("in 3 months ago", refSat)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedNaturalTime))
}

func TestParseNaturalTime_ThisMonday(t *testing.T) {
	// Saturday → this monday = 2026-08-10
	got, err := ParseNaturalTime("this monday", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), got)

	monday := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	got, err = ParseNaturalTime("this monday", monday)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), got)

	got, err = ParseNaturalTime("next monday", monday)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC), got)

	got, err = ParseNaturalTime("last monday", monday)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), got)
}

func TestParseNaturalTime_Weekend(t *testing.T) {
	got, err := ParseNaturalTime("next weekend", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), got) // next Sat

	got, err = ParseNaturalTime("this weekend", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), got) // this Sat
}

func TestParseNaturalTime_Combined(t *testing.T) {
	got, err := ParseNaturalTime("tomorrow at 10:30pm", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 16, 22, 30, 0, 0, time.UTC), got)

	got, err = ParseNaturalTime("next monday at 3pm", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 17, 15, 0, 0, 0, time.UTC), got)

	got, err = ParseNaturalTime("next monday 10:00", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC), got)
}

func TestParseNaturalTime_Invalid13pm(t *testing.T) {
	_, err := ParseNaturalTime("13pm", refSat)
	assert.Error(t, err)
}

func TestParseNaturalTime_KeywordsAndRelative(t *testing.T) {
	got, err := ParseNaturalTime("today", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), got)

	got, err = ParseNaturalTime("3 days ago", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 12, 10, 30, 0, 0, time.UTC), got)

	got, err = ParseNaturalTime("in 2 hours", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 15, 12, 30, 0, 0, time.UTC), got)

	// bare "3 days" → future
	got, err = ParseNaturalTime("3 days", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 18, 10, 30, 0, 0, time.UTC), got)
}

func TestParseNaturalDuration(t *testing.T) {
	d, err := ParseNaturalDuration("2 hours 30 minutes")
	require.NoError(t, err)
	assert.Equal(t, 2*time.Hour+30*time.Minute, d)

	_, err = ParseNaturalDuration("2 months")
	assert.Error(t, err)
}

func TestParseNaturalRange(t *testing.T) {
	r, err := ParseNaturalRange("last 7 days", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 8, 10, 30, 0, 0, time.UTC), r.Start)
	assert.Equal(t, refSat, r.End)

	r, err = ParseNaturalRange("this month", refSat)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), r.Start)
}

func TestShiftNatural(t *testing.T) {
	got, err := ShiftNatural(refSat, "-3 months")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 5, 15, 10, 30, 0, 0, time.UTC), got)
}

func TestParseNaturalTimeInLocation(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	got, err := ParseNaturalTime("today", time.Date(2026, 8, 15, 1, 0, 0, 0, loc))
	require.NoError(t, err)
	assert.Equal(t, loc, got.Location())
	assert.Equal(t, 0, got.Hour())
}
