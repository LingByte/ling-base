// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package meter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsage_Merge(t *testing.T) {
	u := Usage{InputTokens: 100, OutputTokens: 50, ImageCount: 2}
	u.Merge(Usage{InputTokens: 200, OutputTokens: 100, ImageCount: 3})

	assert.Equal(t, 300, u.InputTokens)
	assert.Equal(t, 150, u.OutputTokens)
	assert.Equal(t, 5, u.ImageCount)
}

func TestMemoryMeter_RecordAndQuery(t *testing.T) {
	m := NewMemoryMeter()

	_ = m.Record(nil, &UsageRecord{
		Provider: "openai",
		Model:    "gpt-4o",
		Mode:     1,
		Usage:    Usage{InputTokens: 1000, OutputTokens: 500},
	})
	_ = m.Record(nil, &UsageRecord{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		Mode:     1,
		Usage:    Usage{InputTokens: 2000, OutputTokens: 1000},
	})
	_ = m.Record(nil, &UsageRecord{
		Provider: "claude",
		Model:    "claude-3-5-sonnet",
		Mode:     1,
		Usage:    Usage{InputTokens: 500, OutputTokens: 250, CachedTokens: 100},
	})

	// Query all.
	stats, err := m.Query(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalRecords)
	assert.Equal(t, 3500, stats.TotalUsage.InputTokens)
	assert.Equal(t, 1750, stats.TotalUsage.OutputTokens)
	assert.Equal(t, 100, stats.TotalUsage.CachedTokens)

	// Query by provider.
	stats, err = m.Query(nil, &UsageQuery{Provider: "openai"})
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalRecords)
	assert.Equal(t, 3000, stats.TotalUsage.InputTokens)

	// Query by model.
	stats, err = m.Query(nil, &UsageQuery{Model: "gpt-4o"})
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 1000, stats.TotalUsage.InputTokens)

	// Query by mode.
	stats, err = m.Query(nil, &UsageQuery{Mode: 1})
	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalRecords)
}

func TestMemoryMeter_Aggregate(t *testing.T) {
	m := NewMemoryMeter()

	_ = m.Record(nil, &UsageRecord{
		Provider: "openai", Model: "gpt-4o", Mode: 1,
		Usage: Usage{InputTokens: 1000},
	})
	_ = m.Record(nil, &UsageRecord{
		Provider: "openai", Model: "gpt-4o-mini", Mode: 1,
		Usage: Usage{InputTokens: 2000},
	})
	_ = m.Record(nil, &UsageRecord{
		Provider: "claude", Model: "claude-3-5-sonnet", Mode: 1,
		Usage: Usage{InputTokens: 500},
	})
	_ = m.Record(nil, &UsageRecord{
		Provider: "openai", Model: "dall-e-3", Mode: 3,
		Usage: Usage{ImageCount: 2},
	})

	// Aggregate by provider.
	agg, err := m.Aggregate(nil, nil, "provider")
	require.NoError(t, err)
	assert.Len(t, agg, 2)

	// Aggregate by model.
	agg, err = m.Aggregate(nil, nil, "model")
	require.NoError(t, err)
	assert.Len(t, agg, 4) // gpt-4o, gpt-4o-mini, claude-3-5-sonnet, dall-e-3

	// Aggregate by mode.
	agg, err = m.Aggregate(nil, nil, "mode")
	require.NoError(t, err)
	assert.Len(t, agg, 2) // chat + image
}

func TestMemoryMeter_ImageUsage(t *testing.T) {
	m := NewMemoryMeter()

	_ = m.Record(nil, &UsageRecord{
		Provider: "openai", Model: "dall-e-3", Mode: 3,
		Usage: Usage{ImageCount: 4},
	})

	stats, err := m.Query(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 4, stats.TotalUsage.ImageCount)
}

func TestMemoryMeter_AudioUsage(t *testing.T) {
	m := NewMemoryMeter()

	_ = m.Record(nil, &UsageRecord{
		Provider: "openai", Model: "whisper-1", Mode: 5,
		Usage: Usage{AudioSeconds: 120.5},
	})

	stats, err := m.Query(nil, nil)
	require.NoError(t, err)
	assert.InDelta(t, 120.5, stats.TotalUsage.AudioSeconds, 0.01)
}

func TestMemoryMeter_VideoUsage(t *testing.T) {
	m := NewMemoryMeter()

	_ = m.Record(nil, &UsageRecord{
		Provider: "kling", Model: "kling-v2", Mode: 10,
		Usage: Usage{VideoSeconds: 30.0},
	})

	stats, err := m.Query(nil, nil)
	require.NoError(t, err)
	assert.InDelta(t, 30.0, stats.TotalUsage.VideoSeconds, 0.01)
}

func TestMemoryMeter_CacheTokens(t *testing.T) {
	m := NewMemoryMeter()

	_ = m.Record(nil, &UsageRecord{
		Provider: "claude", Model: "claude-3-5-sonnet", Mode: 1,
		Usage: Usage{
			InputTokens:         1000,
			OutputTokens:        500,
			CachedTokens:        300,
			CacheCreationTokens: 200,
		},
	})

	stats, err := m.Query(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 300, stats.TotalUsage.CachedTokens)
	assert.Equal(t, 200, stats.TotalUsage.CacheCreationTokens)
}

func TestModeName(t *testing.T) {
	assert.Equal(t, "chat", modeName(1))
	assert.Equal(t, "embeddings", modeName(2))
	assert.Equal(t, "image", modeName(3))
	assert.Equal(t, "unknown", modeName(999))
}
