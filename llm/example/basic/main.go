// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Example: basic chat completion with OpenAI provider and usage metering.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/LingByte/ling-base/llm"
	"github.com/LingByte/ling-base/llm/meter"
	"github.com/LingByte/ling-base/llm/provider/openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create a client with OpenAI provider and in-memory meter.
	client := llm.New(
		llm.WithProvider(openai.NewProvider(apiKey)),
		llm.WithMeter(meter.NewMemoryMeter(nil)),
	)

	ctx := context.Background()

	// Send a chat completion request.
	resp, err := client.Chat(ctx, &llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: "user", Content: "Say hello in one sentence."},
		},
	})
	if err != nil {
		log.Fatalf("chat failed: %v", err)
	}

	// Print the response.
	if len(resp.Choices) > 0 {
		fmt.Printf("Response: %v\n", resp.Choices[0].Message.Content)
	}
	fmt.Printf("Usage: %+v\n", resp.Usage)

	// Query aggregated usage stats.
	stats, err := client.Meter().Query(ctx, nil)
	if err != nil {
		log.Fatalf("query failed: %v", err)
	}
	fmt.Printf("\nAggregated stats:\n")
	fmt.Printf("  Total records: %d\n", stats.TotalRecords)
	fmt.Printf("  Total input tokens: %d\n", stats.TotalUsage.InputTokens)
	fmt.Printf("  Total output tokens: %d\n", stats.TotalUsage.OutputTokens)
	fmt.Printf("  Total cost: $%.6f\n", stats.TotalUsage.Cost)

	// Demonstrate pricing lookup.
	pricing := meter.DefaultPricing()
	if p, ok := pricing.Get("gpt-4o"); ok {
		fmt.Printf("\nGPT-4o pricing: input ratio=%.2f, completion ratio=%.2f\n",
			p.InputRatio, p.CompletionRatio)
		fmt.Printf("  → Input price: $%.2f/1M tokens\n", p.InputRatio*2.0)
		fmt.Printf("  → Output price: $%.2f/1M tokens\n", p.InputRatio*p.CompletionRatio*2.0)
	}

	// Demonstrate cost calculation.
	usage := meter.Usage{InputTokens: 1000, OutputTokens: 500}
	cost := pricing.Calculate("gpt-4o", usage)
	fmt.Printf("\nCost for 1K input + 500 output tokens (gpt-4o): $%.6f\n", cost)

	// Pretty-print the response as JSON.
	jsonData, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("\nFull response JSON:\n%s\n", jsonData)
}
