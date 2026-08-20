// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package meter

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUsage_Merge_AllFields verifies that Merge accumulates every field.
func TestUsage_Merge_AllFields(t *testing.T) {
	u := Usage{
		InputTokens:         10,
		OutputTokens:        5,
		TotalTokens:         15,
		CachedTokens:        2,
		CacheCreationTokens: 3,
		CacheCreation5m:     1,
		CacheCreation1h:     1,
		ReasoningTokens:     4,
		ImageCount:          2,
		AudioSeconds:        1.5,
		VideoSeconds:        2.5,
		RequestCount:        1,
		Source:              "openai",
	}

	u.Merge(Usage{
		InputTokens:         20,
		OutputTokens:        10,
		TotalTokens:         30,
		CachedTokens:        4,
		CacheCreationTokens: 6,
		CacheCreation5m:     2,
		CacheCreation1h:     3,
		ReasoningTokens:     8,
		ImageCount:          3,
		AudioSeconds:        3.5,
		VideoSeconds:        4.5,
		RequestCount:        2,
	})

	assert.Equal(t, 30, u.InputTokens)
	assert.Equal(t, 15, u.OutputTokens)
	assert.Equal(t, 45, u.TotalTokens)
	assert.Equal(t, 6, u.CachedTokens)
	assert.Equal(t, 9, u.CacheCreationTokens)
	assert.Equal(t, 3, u.CacheCreation5m)
	assert.Equal(t, 4, u.CacheCreation1h)
	assert.Equal(t, 12, u.ReasoningTokens)
	assert.Equal(t, 5, u.ImageCount)
	assert.InDelta(t, 5.0, u.AudioSeconds, 0.001)
	assert.InDelta(t, 7.0, u.VideoSeconds, 0.001)
	assert.Equal(t, 3, u.RequestCount)
}

// TestMemoryMeter_RecordToMemory verifies a single record is stored and queryable.
func TestMemoryMeter_RecordToMemory(t *testing.T) {
	m := NewMemoryMeter()
	ctx := context.Background()

	rec := &UsageRecord{
		Provider: "openai",
		Model:    "gpt-4o",
		Mode:     1,
		Usage:    Usage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150},
	}
	require.NoError(t, m.Record(ctx, rec))

	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 100, stats.TotalUsage.InputTokens)
	assert.Equal(t, 50, stats.TotalUsage.OutputTokens)
	assert.Equal(t, 150, stats.TotalUsage.TotalTokens)

	// ByProvider / ByModel / ByMode maps should be populated.
	assert.Contains(t, stats.ByProvider, "openai")
	assert.Contains(t, stats.ByModel, "gpt-4o")
	assert.Contains(t, stats.ByMode, 1)
}

// TestMemoryMeter_ConcurrentRecord verifies that concurrent Record calls are
// safe and all records are persisted.
func TestMemoryMeter_ConcurrentRecord(t *testing.T) {
	m := NewMemoryMeter()
	ctx := context.Background()

	const goroutines = 50
	const perG = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				_ = m.Record(ctx, &UsageRecord{
					Provider: "openai",
					Model:    "gpt-4o",
					Mode:     1,
					Usage:    Usage{InputTokens: 1, RequestCount: 1},
				})
			}
		}(i)
	}
	wg.Wait()

	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, goroutines*perG, stats.TotalRecords)
	assert.Equal(t, goroutines*perG, stats.TotalUsage.InputTokens)
	assert.Equal(t, goroutines*perG, stats.TotalUsage.RequestCount)
}

// TestMemoryMeter_QueryFilters verifies filtering by provider, model, and mode.
func TestMemoryMeter_QueryFilters(t *testing.T) {
	m := NewMemoryMeter()
	ctx := context.Background()

	records := []*UsageRecord{
		{Provider: "openai", Model: "gpt-4o", Mode: 1, Usage: Usage{InputTokens: 100}},
		{Provider: "openai", Model: "gpt-4o-mini", Mode: 1, Usage: Usage{InputTokens: 50}},
		{Provider: "claude", Model: "claude-3-5-sonnet", Mode: 1, Usage: Usage{InputTokens: 200}},
		{Provider: "openai", Model: "dall-e-3", Mode: 3, Usage: Usage{ImageCount: 1}},
	}
	for _, r := range records {
		require.NoError(t, m.Record(ctx, r))
	}

	// Filter by provider.
	stats, err := m.Query(ctx, &UsageQuery{Provider: "openai"})
	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalRecords)

	// Filter by model.
	stats, err = m.Query(ctx, &UsageQuery{Model: "gpt-4o"})
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 100, stats.TotalUsage.InputTokens)

	// Filter by mode.
	stats, err = m.Query(ctx, &UsageQuery{Mode: 3})
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 1, stats.TotalUsage.ImageCount)

	// Combined filter.
	stats, err = m.Query(ctx, &UsageQuery{Provider: "openai", Mode: 1})
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalRecords)

	// Non-matching filter returns zero records.
	stats, err = m.Query(ctx, &UsageQuery{Provider: "gemini"})
	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalRecords)
}

// TestMemoryMeter_AggregateByDimensions verifies aggregation by provider,
// model, and mode.
func TestMemoryMeter_AggregateByDimensions(t *testing.T) {
	m := NewMemoryMeter()
	ctx := context.Background()

	records := []*UsageRecord{
		{Provider: "openai", Model: "gpt-4o", Mode: 1, Usage: Usage{InputTokens: 100}},
		{Provider: "openai", Model: "gpt-4o", Mode: 1, Usage: Usage{InputTokens: 50}},
		{Provider: "openai", Model: "gpt-4o-mini", Mode: 1, Usage: Usage{InputTokens: 25}},
		{Provider: "claude", Model: "claude-3-5-sonnet", Mode: 1, Usage: Usage{InputTokens: 200}},
		{Provider: "openai", Model: "dall-e-3", Mode: 3, Usage: Usage{ImageCount: 2}},
	}
	for _, r := range records {
		require.NoError(t, m.Record(ctx, r))
	}

	// Aggregate by provider.
	agg, err := m.Aggregate(ctx, nil, "provider")
	require.NoError(t, err)
	require.Len(t, agg, 2)
	byKey := map[string]UsageAggregate{}
	for _, a := range agg {
		byKey[a.Key] = a
	}
	assert.Equal(t, 4, byKey["openai"].Count)
	assert.Equal(t, 175, byKey["openai"].Usage.InputTokens)
	assert.Equal(t, 1, byKey["claude"].Count)
	assert.Equal(t, 200, byKey["claude"].Usage.InputTokens)

	// Aggregate by model.
	agg, err = m.Aggregate(ctx, nil, "model")
	require.NoError(t, err)
	// gpt-4o, gpt-4o-mini, claude-3-5-sonnet, dall-e-3 => 4 distinct models.
	require.Len(t, agg, 4)

	// Aggregate by mode.
	agg, err = m.Aggregate(ctx, nil, "mode")
	require.NoError(t, err)
	require.Len(t, agg, 2) // chat + image
	modeByKey := map[string]UsageAggregate{}
	for _, a := range agg {
		modeByKey[a.Key] = a
	}
	assert.Equal(t, 4, modeByKey["chat"].Count)
	assert.Equal(t, 1, modeByKey["image"].Count)
	assert.Equal(t, 2, modeByKey["image"].Usage.ImageCount)
}

// TestMemoryMeter_EmptyQueryReturnsAll verifies that a nil query returns all
// records.
func TestMemoryMeter_EmptyQueryReturnsAll(t *testing.T) {
	m := NewMemoryMeter()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, m.Record(ctx, &UsageRecord{
			Provider: "openai",
			Model:    "gpt-4o",
			Mode:     1,
			Usage:    Usage{InputTokens: 10},
		}))
	}

	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 5, stats.TotalRecords)
	assert.Equal(t, 50, stats.TotalUsage.InputTokens)
}

// TestMemoryMeter_ZeroValueUsage verifies that a zero-value Usage record is
// stored and counted without error.
func TestMemoryMeter_ZeroValueUsage(t *testing.T) {
	m := NewMemoryMeter()
	ctx := context.Background()

	require.NoError(t, m.Record(ctx, &UsageRecord{
		Provider: "openai",
		Model:    "gpt-4o",
		Mode:     1,
		Usage:    Usage{},
	}))

	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 0, stats.TotalUsage.InputTokens)
	assert.Equal(t, 0, stats.TotalUsage.OutputTokens)
	assert.Equal(t, 0, stats.TotalUsage.TotalTokens)
}
