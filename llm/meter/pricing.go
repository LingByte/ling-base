// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package meter

// DefaultPricing returns a pricing table pre-populated with all model ratios
// and fixed prices from LingRein's default pricing data.
// 237 model ratios + 31 fixed-price models + 5 audio models.
func DefaultPricing() *PricingTable {
	t := NewPricingTable()

	// Model ratios (input ratio + completion ratio + cache ratio).
	// 1 ratio = $0.002/1K tokens = $2/1M tokens.
	for _, r := range defaultModelRatios {
		t.Set(r.model, ModelPricing{
			InputRatio:      r.inputRatio,
			CompletionRatio: r.compRatio,
			CacheRatio:      r.cacheRatio,
		})
	}

	// Fixed per-call prices (image models, video models, etc.).
	for _, r := range defaultFixedPrices {
		t.Set(r.model, ModelPricing{PricePerCall: r.price})
	}

	// Audio ratios (input ratio + output ratio).
	for _, r := range defaultAudioRatios {
		t.Set(r.model, ModelPricing{
			InputRatio:      r.inputRatio,
			CompletionRatio: r.outRatio,
		})
	}

	return t
}
