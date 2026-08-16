// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package search

import (
	"context"
	"errors"
)

// Sentinel errors for the search engine.
var (
	ErrClosed       = errors.New("search engine closed")
	ErrEmptyDocID   = errors.New("document id cannot be empty")
	ErrNilDoc       = errors.New("document cannot be nil")
	ErrIndexNotOpen = errors.New("index is not open")
)

// Engine defines the full-text search interface.
// Implementations include:
//   - bleve (in-process, disk or memory)
//   - elasticsearch (remote cluster)
type Engine interface {
	// Index adds or updates a single document.
	Index(ctx context.Context, doc Doc) error
	// IndexBatch adds or updates multiple documents in batches.
	IndexBatch(ctx context.Context, docs []Doc) error
	// Delete removes a document by ID.
	Delete(ctx context.Context, id string) error
	// Search executes a search request and returns matching hits.
	Search(ctx context.Context, req SearchRequest) (SearchResult, error)
	// GetAutoCompleteSuggestions returns prefix-based suggestions.
	GetAutoCompleteSuggestions(ctx context.Context, keyword string) ([]string, error)
	// GetSearchSuggestions returns match-based suggestions.
	GetSearchSuggestions(ctx context.Context, keyword string) ([]string, error)
	// DocCount returns the total number of documents in the index.
	DocCount(ctx context.Context) (uint64, error)
	// Stats returns index statistics.
	Stats() map[string]any
	// Close releases index resources.
	Close() error
}
