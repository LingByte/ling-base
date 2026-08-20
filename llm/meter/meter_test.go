// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package meter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricingTable_SetAndGet(t *testing.T) {
	pt := NewPricingTable()

	// Exact match.
	pt.Set("gpt-4o", ModelPricing{InputPricePer1M: 2.5, OutputPricePer1M: 10})
	p, ok := pt.Get("gpt-4o")
	require.True(t, ok)
	assert.Equal(t, 2.5, p.InputPricePer1M)
	assert.Equal(t, 10.0, p.OutputPricePer1M)

	// Wildcard match.
	pt.Set("gpt-4o-*", ModelPricing{InputPricePer1M: 1.25, OutputPricePer1M: 5})
	p, ok = pt.Get("gpt-4o-2024-08-06")
	require.True(t, ok)
	assert.Equal(t, 1.25, p.InputPricePer1M)

	// Exact match takes priority over wildcard.
	p, ok = pt.Get("gpt-4o")
	require.True(t, ok)
	assert.Equal(t, 2.5, p.InputPricePer1M)

	// No match.
	_, ok = pt.Get("nonexistent-model")
	assert.False(t, ok)
}

func TestPricingTable_Calculate_DirectUSD(t *testing.T) {
	pt := NewPricingTable()
	pt.Set("gpt-4o", ModelPricing{
		InputPricePer1M:  2.5,
		OutputPricePer1M: 10,
	})

	usage := Usage{InputTokens: 1_000_000, OutputTokens: 500_000}
	cost := pt.Calculate("gpt-4o", usage)
	// 1M input × $2.5/1M + 500K output × $10/1M = $2.5 + $5 = $7.5
	assert.InDelta(t, 7.5, cost, 0.0001)
}

func TestPricingTable_Calculate_RatioMode(t *testing.T) {
	pt := NewPricingTable()
	pt.Set("test-model", ModelPricing{
		InputRatio:      1.25, // $2.5/1M
		CompletionRatio: 4,    // output = 1.25 × 4 × $2 = $10/1M
	})

	usage := Usage{InputTokens: 1_000_000, OutputTokens: 500_000}
	cost := pt.Calculate("test-model", usage)
	// 1M × $2.5 + 500K × $10 = $7.5
	assert.InDelta(t, 7.5, cost, 0.0001)
}

func TestPricingTable_Calculate_FixedPrice(t *testing.T) {
	pt := NewPricingTable()
	pt.Set("dall-e-3", ModelPricing{PricePerCall: 0.04})

	usage := Usage{ImageCount: 3}
	cost := pt.Calculate("dall-e-3", usage)
	assert.InDelta(t, 0.12, cost, 0.0001)
}

func TestPricingTable_Calculate_CachedTokens(t *testing.T) {
	pt := NewPricingTable()
	pt.Set("gpt-4o", ModelPricing{
		InputPricePer1M:  2.5,
		OutputPricePer1M: 10,
		CachedInputPer1M: 1.25,
	})

	// 1M input, 500K cached → (1M - 500K) × $2.5 + 500K × $1.25 = $1.25 + $0.625 = $1.875
	usage := Usage{InputTokens: 1_000_000, CachedTokens: 500_000}
	cost := pt.Calculate("gpt-4o", usage)
	assert.InDelta(t, 1.875, cost, 0.0001)
}

func TestPricingTable_Calculate_NoPricing(t *testing.T) {
	pt := NewPricingTable()
	usage := Usage{InputTokens: 1000, OutputTokens: 500}
	cost := pt.Calculate("unknown-model", usage)
	assert.Equal(t, 0.0, cost)
}

func TestDefaultPricing_HasCommonModels(t *testing.T) {
	pt := DefaultPricing()

	models := []string{
		"gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-5",
		"claude-3-5-sonnet-20241022", "claude-3-opus-20240229",
		"gemini-2.5-pro", "gemini-2.0-flash",
		"deepseek-chat", "dall-e-3",
	}
	for _, m := range models {
		_, ok := pt.Get(m)
		assert.True(t, ok, "DefaultPricing should have model %s", m)
	}
}

func TestDefaultPricing_GPT4oCost(t *testing.T) {
	pt := DefaultPricing()
	usage := Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
	cost := pt.Calculate("gpt-4o", usage)
	// gpt-4o: input ratio=1.25 ($2.5/1M), completion ratio=0 → defaults to 1
	// 1M input × $2.5 + 1M output × $2.5 = $5
	assert.InDelta(t, 5.0, cost, 0.01)
}

func TestMemoryMeter_RecordAndQuery(t *testing.T) {
	m := NewMemoryMeter(nil)

	// Record some usage.
	_ = m.Record(nil, &UsageRecord{
		Provider: "openai",
		Model:    "gpt-4o",
		Usage:    Usage{InputTokens: 1000, OutputTokens: 500},
	})
	_ = m.Record(nil, &UsageRecord{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		Usage:    Usage{InputTokens: 2000, OutputTokens: 1000},
	})
	_ = m.Record(nil, &UsageRecord{
		Provider: "claude",
		Model:    "claude-3-5-sonnet-20241022",
		Usage:    Usage{InputTokens: 500, OutputTokens: 250},
	})

	// Query all.
	stats, err := m.Query(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalRecords)
	assert.Equal(t, 3500, stats.TotalUsage.InputTokens)
	assert.Equal(t, 1750, stats.TotalUsage.OutputTokens)

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
}

func TestMemoryMeter_Aggregate(t *testing.T) {
	m := NewMemoryMeter(nil)

	_ = m.Record(nil, &UsageRecord{
		Provider: "openai", Model: "gpt-4o",
		Usage: Usage{InputTokens: 1000},
	})
	_ = m.Record(nil, &UsageRecord{
		Provider: "openai", Model: "gpt-4o-mini",
		Usage: Usage{InputTokens: 2000},
	})
	_ = m.Record(nil, &UsageRecord{
		Provider: "claude", Model: "claude-3-5-sonnet",
		Usage: Usage{InputTokens: 500},
	})

	// Aggregate by provider.
	agg, err := m.Aggregate(nil, nil, "provider")
	require.NoError(t, err)
	assert.Len(t, agg, 2)

	// Aggregate by model.
	agg, err = m.Aggregate(nil, nil, "model")
	require.NoError(t, err)
	assert.Len(t, agg, 3)
}

func TestFormatMatchingModelName(t *testing.T) {
	assert.Equal(t, "gpt-4-gizmo-*", FormatMatchingModelName("gpt-4-gizmo-123"))
	assert.Equal(t, "gpt-4o-gizmo-*", FormatMatchingModelName("gpt-4o-gizmo-456"))
	assert.Equal(t, "gpt-4o", FormatMatchingModelName("gpt-4o"))
}
