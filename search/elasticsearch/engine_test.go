// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package elasticsearch

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/LingByte/ling-base/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// esAddr returns the ES address from env or skips the test.
func esAddr(t *testing.T) string {
	addr := os.Getenv("ES_ADDR")
	if addr == "" {
		addr = "http://localhost:9200"
	}
	return addr
}

// skipIfNoES skips the test if ES is not reachable.
func skipIfNoES(t *testing.T) string {
	addr := esAddr(t)
	eng, err := New(Config{
		Addresses:       []string{addr},
		IndexName:       "test_ling_base_es",
		AutoCreateIndex: true,
		QueryTimeout:    5 * time.Second,
	})
	if err != nil {
		t.Skipf("ES not available at %s: %v", addr, err)
	}
	eng.Close()
	return addr
}

// cleanupIndex deletes all documents in the test index.

func newTestEngine(t *testing.T) search.Engine {
	addr := skipIfNoES(t)
	idxName := fmt.Sprintf("test_ling_base_es_%d", time.Now().UnixNano())

	eng, err := New(Config{
		Addresses:       []string{addr},
		IndexName:       idxName,
		AutoCreateIndex: true,
		QueryTimeout:    10 * time.Second,
		BatchSize:       50,
	})
	require.NoError(t, err)

	// Delete index on cleanup
	t.Cleanup(func() {
		// We can't delete the index via the Engine interface,
		// but ES test indices are ephemeral. Just close.
		eng.Close()
	})

	return eng
}

func TestES_IndexAndSearch(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	docs := []search.Doc{
		{ID: "1", Type: "article", Fields: map[string]any{
			"title":   "Go Programming Language",
			"content": "Go is a statically typed, compiled programming language designed at Google.",
			"author":  "Alice",
			"tags":    "go,programming,google",
		}},
		{ID: "2", Type: "article", Fields: map[string]any{
			"title":   "Python Programming",
			"content": "Python is a high-level, general-purpose programming language.",
			"author":  "Bob",
			"tags":    "python,programming",
		}},
		{ID: "3", Type: "article", Fields: map[string]any{
			"title":   "Rust Memory Safety",
			"content": "Rust provides memory safety without a garbage collector.",
			"author":  "Alice",
			"tags":    "rust,programming,memory",
		}},
	}

	for _, d := range docs {
		err := eng.Index(ctx, d)
		require.NoError(t, err)
	}

	// Wait for indexing
	time.Sleep(2 * time.Second)

	// Keyword search
	res, err := eng.Search(ctx, search.NewKeywordSearch("programming", 10))
	require.NoError(t, err)
	assert.Greater(t, res.Total, uint64(0))
	t.Logf("keyword search 'programming': %d hits", res.Total)

	// Term search (filter by author)
	res, err = eng.Search(ctx, search.NewTermSearch("author", "Alice", 10))
	require.NoError(t, err)
	assert.Equal(t, uint64(2), res.Total)

	// Match search on specific field
	res, err = eng.Search(ctx, search.NewMatchSearch("title", "programming", 10))
	require.NoError(t, err)
	assert.Greater(t, res.Total, uint64(0))

	// Phrase search
	res, err = eng.Search(ctx, search.NewPhraseSearch("content", "memory safety", 10))
	require.NoError(t, err)
	assert.Greater(t, res.Total, uint64(0))
}

func TestES_BatchIndex(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	docs := make([]search.Doc, 20)
	for i := 0; i < 20; i++ {
		docs[i] = search.Doc{
			ID:   fmt.Sprintf("batch-%d", i),
			Type: "article",
			Fields: map[string]any{
				"title":   fmt.Sprintf("Batch Document %d", i),
				"content": fmt.Sprintf("This is batch document number %d for testing bulk indexing.", i),
			},
		}
	}

	err := eng.IndexBatch(ctx, docs)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	count, err := eng.DocCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(20), count)
}

func TestES_Delete(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	err := eng.Index(ctx, search.Doc{
		ID:   "del-1",
		Type: "article",
		Fields: map[string]any{
			"title":   "To Be Deleted",
			"content": "This document will be deleted.",
		},
	})
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	count, err := eng.DocCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)

	err = eng.Delete(ctx, "del-1")
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	count, err = eng.DocCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), count)
}

func TestES_DeleteNonExistent(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	// Deleting a non-existent doc should not error (404 is OK)
	err := eng.Delete(ctx, "nonexistent-id")
	assert.NoError(t, err)
}

func TestES_Facets(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	docs := []search.Doc{
		{ID: "f1", Type: "article", Fields: map[string]any{"title": "Go 1", "author": "Alice", "category": "tech"}},
		{ID: "f2", Type: "article", Fields: map[string]any{"title": "Go 2", "author": "Alice", "category": "tech"}},
		{ID: "f3", Type: "article", Fields: map[string]any{"title": "Python 1", "author": "Bob", "category": "tech"}},
		{ID: "f4", Type: "article", Fields: map[string]any{"title": "Rust 1", "author": "Bob", "category": "systems"}},
	}

	for _, d := range docs {
		eng.Index(ctx, d)
	}

	time.Sleep(2 * time.Second)

	res, err := eng.Search(ctx, search.SearchRequest{
		Size: 10,
		Facets: []search.FacetRequest{
			{Name: "authors", Field: "author", Size: 10},
			{Name: "categories", Field: "category", Size: 10},
		},
	})
	require.NoError(t, err)

	assert.Contains(t, res.Facets, "authors")
	assert.Contains(t, res.Facets, "categories")

	// Alice should have 2 docs, Bob should have 2 docs
	authorFacet := res.Facets["authors"]
	foundAlice := false
	foundBob := false
	for _, term := range authorFacet.Terms {
		if term.Term == "Alice" {
			foundAlice = true
			assert.Equal(t, 2, term.Count)
		}
		if term.Term == "Bob" {
			foundBob = true
			assert.Equal(t, 2, term.Count)
		}
	}
	assert.True(t, foundAlice, "Alice should be in author facet")
	assert.True(t, foundBob, "Bob should be in author facet")
}

func TestES_Highlight(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.Index(ctx, search.Doc{
		ID:   "hl-1",
		Type: "article",
		Fields: map[string]any{
			"title":   "Go Programming",
			"content": "Go is a great programming language for systems development.",
		},
	})

	time.Sleep(2 * time.Second)

	res, err := eng.Search(ctx, search.SearchRequest{
		Keyword:         "programming",
		Size:            10,
		Highlight:       true,
		HighlightFields: []string{"title", "content"},
	})
	require.NoError(t, err)
	require.Greater(t, len(res.Hits), 0)

	// At least one hit should have fragments
	hasFragments := false
	for _, hit := range res.Hits {
		if len(hit.Fragments) > 0 {
			hasFragments = true
			break
		}
	}
	assert.True(t, hasFragments, "at least one hit should have highlight fragments")
}

func TestES_PrefixSearch(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	docs := []search.Doc{
		{ID: "p1", Type: "article", Fields: map[string]any{"title": "Golang Tutorial", "content": "Learn Go"}},
		{ID: "p2", Type: "article", Fields: map[string]any{"title": "Goroutines Guide", "content": "Concurrency in Go"}},
		{ID: "p3", Type: "article", Fields: map[string]any{"title": "Python Guide", "content": "Learn Python"}},
	}
	for _, d := range docs {
		eng.Index(ctx, d)
	}

	time.Sleep(2 * time.Second)

	res, err := eng.Search(ctx, search.SearchRequest{
		Prefixes: []search.ClausePrefix{{Field: "title", Prefix: "Go"}},
		Size:     10,
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(2), res.Total)
}

func TestES_FuzzySearch(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.Index(ctx, search.Doc{
		ID:   "fz1",
		Type: "article",
		Fields: map[string]any{
			"title":   "Python Programming",
			"content": "Python is great for data science.",
		},
	})

	time.Sleep(2 * time.Second)

	// "Pyton" with fuzziness should match "Python"
	res, err := eng.Search(ctx, search.SearchRequest{
		Fuzzies: []search.ClauseFuzzy{{Field: "title", Term: "Pyton", Fuzziness: 2}},
		Size:    10,
	})
	require.NoError(t, err)
	assert.Greater(t, res.Total, uint64(0))
}

func TestES_NumericRange(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	docs := []search.Doc{
		{ID: "n1", Type: "article", Fields: map[string]any{"title": "Low Views", "views": 10}},
		{ID: "n2", Type: "article", Fields: map[string]any{"title": "Mid Views", "views": 500}},
		{ID: "n3", Type: "article", Fields: map[string]any{"title": "High Views", "views": 10000}},
	}
	for _, d := range docs {
		eng.Index(ctx, d)
	}

	time.Sleep(2 * time.Second)

	gte := 100.0
	lte := 5000.0
	res, err := eng.Search(ctx, search.SearchRequest{
		NumericRanges: []search.NumericRangeFilter{
			{Field: "views", GTE: &gte, LTE: &lte},
		},
		Size: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(1), res.Total)
}

func TestES_DocCount(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	count, err := eng.DocCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), count)

	eng.Index(ctx, search.Doc{ID: "c1", Type: "article", Fields: map[string]any{"title": "Count Test"}})
	time.Sleep(1 * time.Second)

	count, err = eng.DocCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)
}

func TestES_Stats(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.Index(ctx, search.Doc{ID: "s1", Type: "article", Fields: map[string]any{"title": "Stats Test"}})
	eng.Search(ctx, search.NewKeywordSearch("stats", 10))
	eng.Delete(ctx, "s1")

	stats := eng.Stats()
	assert.Equal(t, "elasticsearch", stats["backend"])
	assert.Equal(t, uint64(1), stats["indexOps"])
	assert.Equal(t, uint64(1), stats["searchOps"])
	assert.Equal(t, uint64(1), stats["deleteOps"])
}

func TestES_Suggestions(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	docs := []search.Doc{
		{ID: "sg1", Type: "article", Fields: map[string]any{"title": "Go Programming Guide", "content": "Learn Go"}},
		{ID: "sg2", Type: "article", Fields: map[string]any{"title": "Goroutines Tutorial", "content": "Go concurrency"}},
	}
	for _, d := range docs {
		eng.Index(ctx, d)
	}

	time.Sleep(2 * time.Second)

	// Search suggestions
	suggestions, err := eng.GetSearchSuggestions(ctx, "Go")
	require.NoError(t, err)
	assert.Greater(t, len(suggestions), 0)
	t.Logf("suggestions: %v", suggestions)
}

func TestES_Close(t *testing.T) {
	eng := newTestEngine(t)

	err := eng.Close()
	assert.NoError(t, err)

	// Double close should be safe
	err = eng.Close()
	assert.NoError(t, err)

	// Operations after close should fail
	err = eng.Index(context.Background(), search.Doc{ID: "x", Type: "test", Fields: map[string]any{"title": "test"}})
	assert.Error(t, err)
	assert.True(t, err != nil)
}

func TestES_EmptyDocID(t *testing.T) {
	eng := newTestEngine(t)
	err := eng.Index(context.Background(), search.Doc{ID: "", Type: "test", Fields: map[string]any{"title": "test"}})
	assert.Error(t, err)
	assert.Equal(t, search.ErrEmptyDocID, err)
}

func TestES_New_NoAddresses(t *testing.T) {
	_, err := New(Config{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one address")
}
