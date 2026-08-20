// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package meter provides usage metering and cost calculation for LLM API
// calls. It records per-call usage (tokens, images, audio seconds) and
// calculates cost from a pricing table.
//
// The pricing system supports two modes:
//   - Ratio mode (compatible with LingRein/newapi quota system): 1 ratio unit
//     = $0.002 / 1K tokens. A model's input ratio × $0.002/1K = input price.
//     Completion ratio × input ratio = output price.
//   - Direct USD mode: prices are specified directly in USD per 1M tokens.
package meter

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/ling-base/llm/types"
)

// ─── Ratio constants (compatible with LingRein/newapi) ──────────

const (
	USD2RMB = 7.3
	USD     = 500 // $0.002 = 1 ratio unit → $1 = 500
	RMB     = USD / USD2RMB
)

// ─── Usage types ─────────────────────────────────────────────────

// Usage holds metering data for a single API call.
type Usage struct {
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	TotalTokens   int     `json:"total_tokens"`
	CachedTokens  int     `json:"cached_tokens,omitempty"`
	ImageCount    int     `json:"image_count,omitempty"`
	AudioSeconds  float64 `json:"audio_seconds,omitempty"`
	VideoSeconds  float64 `json:"video_seconds,omitempty"`
	Cost          float64 `json:"cost,omitempty"` // in USD
	Source        string  `json:"source,omitempty"` // provider format
}

// Merge adds another Usage's counts into this one.
func (u *Usage) Merge(other Usage) {
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.TotalTokens += other.TotalTokens
	u.CachedTokens += other.CachedTokens
	u.ImageCount += other.ImageCount
	u.AudioSeconds += other.AudioSeconds
	u.VideoSeconds += other.VideoSeconds
	u.Cost += other.Cost
}

// mergeUsage is a helper that merges two Usage values and returns the result.
func mergeUsage(a, b Usage) Usage {
	a.Merge(b)
	return a
}

// UsageRecord is a stored metering event.
type UsageRecord struct {
	ID        string    `json:"id,omitempty"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Usage     Usage     `json:"usage"`
	Timestamp time.Time `json:"timestamp"`
}

// UsageQuery filters usage records.
type UsageQuery struct {
	Provider string
	Model    string
	Start    time.Time
	End      time.Time
}

// UsageStats is an aggregated usage summary.
type UsageStats struct {
	TotalRecords int               `json:"total_records"`
	TotalUsage   Usage             `json:"total_usage"`
	ByProvider   map[string]Usage  `json:"by_provider,omitempty"`
	ByModel      map[string]Usage  `json:"by_model,omitempty"`
	TimeRange    [2]time.Time      `json:"time_range,omitempty"`
}

// UsageAggregate is one bucket in a grouped aggregation.
type UsageAggregate struct {
	Key   string `json:"key"`
	Usage Usage  `json:"usage"`
	Count int    `json:"count"`
}

// ─── Meter interface ─────────────────────────────────────────────

// Meter is the interface for recording and querying usage.
type Meter interface {
	Record(ctx context.Context, record *UsageRecord) error
	Query(ctx context.Context, q *UsageQuery) (*UsageStats, error)
	Aggregate(ctx context.Context, q *UsageQuery, groupBy string) ([]UsageAggregate, error)
	Close() error
}

// ─── In-memory Meter ─────────────────────────────────────────────

// NewMemoryMeter creates an in-memory Meter with the given pricing table.
func NewMemoryMeter(pricing *PricingTable) Meter {
	if pricing == nil {
		pricing = DefaultPricing()
	}
	return &memoryMeter{pricing: pricing}
}

type memoryMeter struct {
	mu      sync.RWMutex
	records []UsageRecord
	pricing *PricingTable
}

func (m *memoryMeter) Record(ctx context.Context, record *UsageRecord) error {
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}
	if record.Usage.Cost == 0 {
		record.Usage.Cost = m.pricing.Calculate(record.Model, record.Usage)
	}
	m.mu.Lock()
	m.records = append(m.records, *record)
	m.mu.Unlock()
	return nil
}

func (m *memoryMeter) Query(ctx context.Context, q *UsageQuery) (*UsageStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stats := &UsageStats{
		ByProvider: make(map[string]Usage),
		ByModel:    make(map[string]Usage),
	}
	var minT, maxT time.Time
	for _, r := range m.records {
		if !matchQuery(r, q) {
			continue
		}
		stats.TotalRecords++
		stats.TotalUsage.Merge(r.Usage)
		stats.ByProvider[r.Provider] = mergeUsage(stats.ByProvider[r.Provider], r.Usage)
		stats.ByModel[r.Model] = mergeUsage(stats.ByModel[r.Model], r.Usage)
		if minT.IsZero() || r.Timestamp.Before(minT) {
			minT = r.Timestamp
		}
		if maxT.IsZero() || r.Timestamp.After(maxT) {
			maxT = r.Timestamp
		}
	}
	if !minT.IsZero() {
		stats.TimeRange = [2]time.Time{minT, maxT}
	}
	return stats, nil
}

func (m *memoryMeter) Aggregate(ctx context.Context, q *UsageQuery, groupBy string) ([]UsageAggregate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	groups := make(map[string]*UsageAggregate)
	for _, r := range m.records {
		if !matchQuery(r, q) {
			continue
		}
		var key string
		switch groupBy {
		case "provider":
			key = r.Provider
		case "model":
			key = r.Model
		default:
			key = r.Provider + "/" + r.Model
		}
		g, ok := groups[key]
		if !ok {
			g = &UsageAggregate{Key: key}
			groups[key] = g
		}
		g.Usage = mergeUsage(g.Usage, r.Usage)
		g.Count++
	}
	result := make([]UsageAggregate, 0, len(groups))
	for _, g := range groups {
		result = append(result, *g)
	}
	return result, nil
}

func (m *memoryMeter) Close() error { return nil }

func matchQuery(r UsageRecord, q *UsageQuery) bool {
	if q == nil {
		return true
	}
	if q.Provider != "" && r.Provider != q.Provider {
		return false
	}
	if q.Model != "" && r.Model != q.Model {
		return false
	}
	if !q.Start.IsZero() && r.Timestamp.Before(q.Start) {
		return false
	}
	if !q.End.IsZero() && r.Timestamp.After(q.End) {
		return false
	}
	return true
}

// ─── Pricing Table ───────────────────────────────────────────────

// ModelPricing defines the cost structure for a single model.
// All prices are in USD.
type ModelPricing struct {
	// Ratio-mode pricing (compatible with LingRein/newapi):
	// 1 ratio = $0.002 / 1K tokens = $2 / 1M tokens
	InputRatio         float64 `json:"input_ratio,omitempty"`          // input token ratio
	CompletionRatio    float64 `json:"completion_ratio,omitempty"`     // output/input ratio (default 1)
	CacheRatio         float64 `json:"cache_ratio,omitempty"`          // cache hit ratio relative to input

	// Direct USD pricing (overrides ratio mode if set):
	InputPricePer1M    float64 `json:"input_price_per_1m,omitempty"`
	OutputPricePer1M   float64 `json:"output_price_per_1m,omitempty"`
	CachedInputPer1M   float64 `json:"cached_input_per_1m,omitempty"`

	// Fixed price per call (for image models, etc.):
	PricePerCall       float64 `json:"price_per_call,omitempty"`

	// Audio pricing (USD per second):
	AudioPricePerSec   float64 `json:"audio_price_per_sec,omitempty"`
	AudioOutputPerSec  float64 `json:"audio_output_per_sec,omitempty"`
}

// PricingTable maps model names (or glob patterns) to their pricing.
type PricingTable struct {
	mu      sync.RWMutex
	entries []pricingEntry
}

type pricingEntry struct {
	pattern string
	pricing ModelPricing
}

// NewPricingTable creates an empty pricing table.
func NewPricingTable() *PricingTable {
	return &PricingTable{}
}

// Set adds or replaces pricing for a model name or glob pattern.
func (t *PricingTable) Set(model string, p ModelPricing) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, e := range t.entries {
		if e.pattern == model {
			t.entries[i].pricing = p
			return
		}
	}
	t.entries = append(t.entries, pricingEntry{pattern: model, pricing: p})
}

// SetRatio sets ratio-mode pricing for a model.
func (t *PricingTable) SetRatio(model string, inputRatio, completionRatio float64) {
	t.Set(model, ModelPricing{InputRatio: inputRatio, CompletionRatio: completionRatio})
}

// SetPrice sets direct USD pricing for a model.
func (t *PricingTable) SetPrice(model string, inputPer1M, outputPer1M float64) {
	t.Set(model, ModelPricing{InputPricePer1M: inputPer1M, OutputPricePer1M: outputPer1M})
}

// SetFixedPrice sets a fixed per-call price.
func (t *PricingTable) SetFixedPrice(model string, price float64) {
	t.Set(model, ModelPricing{PricePerCall: price})
}

// Get returns the pricing for a model. Tries exact match first, then
// wildcard patterns (longest match wins).
func (t *PricingTable) Get(model string) (ModelPricing, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	// Exact match.
	for _, e := range t.entries {
		if e.pattern == model {
			return e.pricing, true
		}
	}
	// Wildcard match (longest pattern wins).
	var best *pricingEntry
	var bestLen int
	for i := range t.entries {
		e := &t.entries[i]
		if strings.HasSuffix(e.pattern, "*") {
			prefix := strings.TrimSuffix(e.pattern, "*")
			if strings.HasPrefix(model, prefix) && len(e.pattern) > bestLen {
				best = e
				bestLen = len(e.pattern)
			}
		}
	}
	if best != nil {
		return best.pricing, true
	}
	return ModelPricing{}, false
}

// Calculate computes the cost (in USD) for a given model and usage.
func (t *PricingTable) Calculate(model string, usage Usage) float64 {
	p, ok := t.Get(model)
	if !ok {
		return 0
	}

	var cost float64

	// Fixed per-call price (image models, etc.)
	if p.PricePerCall > 0 && usage.ImageCount > 0 {
		return float64(usage.ImageCount) * p.PricePerCall
	}

	// Token-based cost.
	if usage.InputTokens > 0 {
		inputPrice := p.InputPricePer1M
		if inputPrice == 0 && p.InputRatio > 0 {
			// Ratio mode: 1 ratio = $0.002/1K = $2/1M
			inputPrice = p.InputRatio * 2.0
		}
		cost += float64(usage.InputTokens) / 1_000_000 * inputPrice
	}

	if usage.OutputTokens > 0 {
		outputPrice := p.OutputPricePer1M
		if outputPrice == 0 && p.InputRatio > 0 {
			compRatio := p.CompletionRatio
			if compRatio == 0 {
				compRatio = 1
			}
			outputPrice = p.InputRatio * compRatio * 2.0
		}
		cost += float64(usage.OutputTokens) / 1_000_000 * outputPrice
	}

	// Cached tokens are cheaper.
	if usage.CachedTokens > 0 {
		cachedPrice := p.CachedInputPer1M
		if cachedPrice == 0 && p.CacheRatio > 0 && p.InputRatio > 0 {
			cachedPrice = p.InputRatio * p.CacheRatio * 2.0
		}
		if cachedPrice > 0 {
			// Subtract full input price for cached tokens, add cached price.
			inputPrice := p.InputPricePer1M
			if inputPrice == 0 {
				inputPrice = p.InputRatio * 2.0
			}
			cost -= float64(usage.CachedTokens) / 1_000_000 * inputPrice
			cost += float64(usage.CachedTokens) / 1_000_000 * cachedPrice
		}
	}

	// Audio cost.
	if usage.AudioSeconds > 0 {
		if p.AudioPricePerSec > 0 {
			cost += usage.AudioSeconds * p.AudioPricePerSec
		}
	}

	return cost
}

// FormatMatchingModelName normalizes model names for pricing lookup.
// Adapted from LingRein's FormatMatchingModelName.
func FormatMatchingModelName(name string) string {
	if strings.HasPrefix(name, "gpt-4-gizmo") {
		return "gpt-4-gizmo-*"
	}
	if strings.HasPrefix(name, "gpt-4o-gizmo") {
		return "gpt-4o-gizmo-*"
	}
	return name
}

// Ensure types.RWMap is referenced (used for concurrent pricing maps in
// advanced usage).
var _ = types.NewRWMap[string, float64]
