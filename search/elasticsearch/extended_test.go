// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package elasticsearch

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/LingByte/ling-base/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// GetByID + Update
// ============================================================

func TestES_GetByID(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	err := eng.Index(ctx, search.Doc{
		ID:   "get-1",
		Type: "article",
		Fields: map[string]any{
			"title":   "Get By ID Test",
			"content": "Testing GetByID",
			"author":  "Tester",
		},
	})
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	doc, err := eng.(search.ExtendedEngine).GetByID(ctx, "get-1")
	require.NoError(t, err)
	assert.Equal(t, "get-1", doc.ID)
	assert.Equal(t, "Get By ID Test", doc.Fields["title"])
	assert.Equal(t, "article", doc.Type)
}

func TestES_GetByID_NotFound(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	_, err := eng.(search.ExtendedEngine).GetByID(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestES_Update(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.Index(ctx, search.Doc{
		ID:   "upd-1",
		Type: "article",
		Fields: map[string]any{
			"title":  "Original Title",
			"views":  10,
			"author": "Alice",
		},
	})
	time.Sleep(1 * time.Second)

	// Partial update
	err := eng.(search.ExtendedEngine).Update(ctx, "upd-1", map[string]any{
		"title": "Updated Title",
		"views": 100,
	})
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	doc, err := eng.(search.ExtendedEngine).GetByID(ctx, "upd-1")
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", doc.Fields["title"])
	assert.Equal(t, float64(100), doc.Fields["views"])
	// author should be preserved
	assert.Equal(t, "Alice", doc.Fields["author"])
}

// ============================================================
// BulkDelete + BulkUpdate
// ============================================================

func TestES_BulkDelete(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		eng.Index(ctx, search.Doc{
			ID:     fmt.Sprintf("bd-%d", i),
			Type:   "article",
			Fields: map[string]any{"title": fmt.Sprintf("Bulk Delete %d", i)},
		})
	}
	time.Sleep(1 * time.Second)

	count, _ := eng.DocCount(ctx)
	assert.Equal(t, uint64(5), count)

	err := eng.(search.ExtendedEngine).BulkDelete(ctx, []string{"bd-0", "bd-1", "bd-2"})
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	count, _ = eng.DocCount(ctx)
	assert.Equal(t, uint64(2), count)
}

func TestES_BulkUpdate(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		eng.Index(ctx, search.Doc{
			ID:     fmt.Sprintf("bu-%d", i),
			Type:   "article",
			Fields: map[string]any{"title": fmt.Sprintf("Bulk Update %d", i), "views": 0},
		})
	}
	time.Sleep(1 * time.Second)

	updates := map[string]map[string]any{
		"bu-0": {"views": 100},
		"bu-1": {"views": 200},
		"bu-2": {"views": 300},
	}
	err := eng.(search.ExtendedEngine).BulkUpdate(ctx, updates)
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	doc, _ := eng.(search.ExtendedEngine).GetByID(ctx, "bu-0")
	assert.Equal(t, float64(100), doc.Fields["views"])

	doc, _ = eng.(search.ExtendedEngine).GetByID(ctx, "bu-2")
	assert.Equal(t, float64(300), doc.Fields["views"])
}

// ============================================================
// DeleteByQuery + UpdateByQuery
// ============================================================

func TestES_DeleteByQuery(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	docs := []search.Doc{
		{ID: "dqb-1", Type: "article", Fields: map[string]any{"title": "Delete Me", "author": "Alice"}},
		{ID: "dqb-2", Type: "article", Fields: map[string]any{"title": "Delete Me", "author": "Alice"}},
		{ID: "dqb-3", Type: "article", Fields: map[string]any{"title": "Keep Me", "author": "Bob"}},
	}
	for _, d := range docs {
		eng.Index(ctx, d)
	}
	time.Sleep(2 * time.Second)

	deleted, err := eng.(search.ExtendedEngine).DeleteByQuery(ctx, search.SearchRequest{
		MustTerms: map[string][]string{"author": {"Alice"}},
		Size:      100,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	time.Sleep(1 * time.Second)

	count, _ := eng.DocCount(ctx)
	assert.Equal(t, uint64(1), count)
}

func TestES_UpdateByQuery(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	docs := []search.Doc{
		{ID: "ubq-1", Type: "article", Fields: map[string]any{"title": "Update Me", "status": "draft", "author": "Alice"}},
		{ID: "ubq-2", Type: "article", Fields: map[string]any{"title": "Update Me", "status": "draft", "author": "Alice"}},
		{ID: "ubq-3", Type: "article", Fields: map[string]any{"title": "No Update", "status": "published", "author": "Bob"}},
	}
	for _, d := range docs {
		eng.Index(ctx, d)
	}
	time.Sleep(2 * time.Second)

	updated, err := eng.(search.ExtendedEngine).UpdateByQuery(ctx, search.SearchRequest{
		MustTerms: map[string][]string{"status": {"draft"}},
		Size:      100,
	}, map[string]any{"status": "published"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), updated)
	time.Sleep(1 * time.Second)

	// Verify
	doc, _ := eng.(search.ExtendedEngine).GetByID(ctx, "ubq-1")
	assert.Equal(t, "published", doc.Fields["status"])
}

// ============================================================
// Scroll API
// ============================================================

func TestES_Scroll(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	// Index 25 docs
	for i := 0; i < 25; i++ {
		eng.Index(ctx, search.Doc{
			ID:     fmt.Sprintf("sc-%d", i),
			Type:   "article",
			Fields: map[string]any{"title": fmt.Sprintf("Scroll Doc %d", i)},
		})
	}
	time.Sleep(2 * time.Second)

	// First scroll page (size=10)
	result, err := eng.(search.ExtendedEngine).Scroll(ctx, search.SearchRequest{
		Size:   10,
		SortBy: []string{"_doc"},
	}, 1*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ScrollID)
	assert.Len(t, result.Hits, 10)
	assert.Equal(t, uint64(25), result.Total)

	// Second page
	result2, err := eng.(search.ExtendedEngine).ScrollNext(ctx, result.ScrollID, 1*time.Minute)
	require.NoError(t, err)
	assert.Len(t, result2.Hits, 10)

	// Third page
	result3, err := eng.(search.ExtendedEngine).ScrollNext(ctx, result2.ScrollID, 1*time.Minute)
	require.NoError(t, err)
	assert.Len(t, result3.Hits, 5)

	// Clear scroll
	err = eng.(search.ExtendedEngine).ClearScroll(ctx, result2.ScrollID)
	assert.NoError(t, err)
}

// ============================================================
// SearchAfter
// ============================================================

func TestES_SearchAfter(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		eng.Index(ctx, search.Doc{
			ID:     fmt.Sprintf("sa-%02d", i),
			Type:   "article",
			Fields: map[string]any{"title": fmt.Sprintf("SearchAfter %d", i)},
		})
	}
	time.Sleep(2 * time.Second)

	// First page
	res, err := eng.(search.ExtendedEngine).SearchAfter(ctx, search.SearchRequest{
		Size:   5,
		SortBy: []string{"_doc"},
	}, nil)
	require.NoError(t, err)
	assert.Len(t, res.Hits, 5)
	require.NotEmpty(t, res.Hits[4].Sort)

	// Next page using sort values from last hit
	res2, err := eng.(search.ExtendedEngine).SearchAfter(ctx, search.SearchRequest{
		Size:   5,
		SortBy: []string{"_doc"},
	}, res.Hits[4].Sort)
	require.NoError(t, err)
	assert.Len(t, res2.Hits, 5)
}

// ============================================================
// Aggregations
// ============================================================

func TestES_Aggregations_Terms(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	docs := []search.Doc{
		{ID: "agg-1", Type: "article", Fields: map[string]any{"title": "A", "author": "Alice", "views": 100}},
		{ID: "agg-2", Type: "article", Fields: map[string]any{"title": "B", "author": "Alice", "views": 200}},
		{ID: "agg-3", Type: "article", Fields: map[string]any{"title": "C", "author": "Bob", "views": 50}},
	}
	for _, d := range docs {
		eng.Index(ctx, d)
	}
	time.Sleep(2 * time.Second)

	res, err := eng.Search(ctx, search.SearchRequest{
		Size: 0,
		Aggregations: []search.AggregationRequest{
			{Name: "by_author", Type: "terms", Field: "author", Size: 10},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, res.Aggregations, "by_author")
	assert.Len(t, res.Aggregations["by_author"].Buckets, 2)
}

func TestES_Aggregations_Stats(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	docs := []search.Doc{
		{ID: "st-1", Type: "article", Fields: map[string]any{"title": "A", "views": 100}},
		{ID: "st-2", Type: "article", Fields: map[string]any{"title": "B", "views": 200}},
		{ID: "st-3", Type: "article", Fields: map[string]any{"title": "C", "views": 300}},
	}
	for _, d := range docs {
		eng.Index(ctx, d)
	}
	time.Sleep(2 * time.Second)

	res, err := eng.Search(ctx, search.SearchRequest{
		Size: 0,
		Aggregations: []search.AggregationRequest{
			{Name: "view_stats", Type: "stats", Field: "views"},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, res.Aggregations, "view_stats")
	require.NotNil(t, res.Aggregations["view_stats"].Stats)
	assert.Equal(t, int64(3), res.Aggregations["view_stats"].Stats.Count)
	assert.Equal(t, 200.0, res.Aggregations["view_stats"].Stats.Avg)
	assert.Equal(t, 600.0, res.Aggregations["view_stats"].Stats.Sum)
}

func TestES_Aggregations_Cardinality(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	docs := []search.Doc{
		{ID: "ca-1", Type: "article", Fields: map[string]any{"title": "A", "author": "Alice"}},
		{ID: "ca-2", Type: "article", Fields: map[string]any{"title": "B", "author": "Alice"}},
		{ID: "ca-3", Type: "article", Fields: map[string]any{"title": "C", "author": "Bob"}},
		{ID: "ca-4", Type: "article", Fields: map[string]any{"title": "D", "author": "Charlie"}},
	}
	for _, d := range docs {
		eng.Index(ctx, d)
	}
	time.Sleep(2 * time.Second)

	res, err := eng.Search(ctx, search.SearchRequest{
		Size: 0,
		Aggregations: []search.AggregationRequest{
			{Name: "unique_authors", Type: "cardinality", Field: "author"},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, res.Aggregations, "unique_authors")
	assert.Equal(t, int64(3), res.Aggregations["unique_authors"].Cardinality)
}

// ============================================================
// Refresh + Flush
// ============================================================

func TestES_Refresh(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.Index(ctx, search.Doc{ID: "rf-1", Type: "article", Fields: map[string]any{"title": "Refresh Test"}})

	// Without refresh, the doc might not be visible immediately.
	// With explicit refresh, it should be.
	err := eng.(search.ExtendedEngine).Refresh(ctx)
	assert.NoError(t, err)

	count, _ := eng.DocCount(ctx)
	assert.Equal(t, uint64(1), count)
}

func TestES_Flush(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.Index(ctx, search.Doc{ID: "fl-1", Type: "article", Fields: map[string]any{"title": "Flush Test"}})

	err := eng.(search.ExtendedEngine).Flush(ctx)
	assert.NoError(t, err)
}

// ============================================================
// HealthCheck
// ============================================================

func TestES_HealthCheck(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	err := eng.(search.ExtendedEngine).HealthCheck(ctx)
	assert.NoError(t, err)
}

// ============================================================
// Index Management
// ============================================================

func TestES_IndexManagement(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	testIdx := fmt.Sprintf("test_mgmt_%d", time.Now().UnixNano())

	// Create
	err := eng.(*engine).CreateIndex(ctx, testIdx, nil)
	assert.NoError(t, err)

	// Exists
	exists, err := eng.(*engine).IndexExists(ctx, testIdx)
	require.NoError(t, err)
	assert.True(t, exists)

	// Delete
	err = eng.(*engine).DeleteIndex(ctx, testIdx)
	assert.NoError(t, err)

	// Should not exist
	exists, err = eng.(*engine).IndexExists(ctx, testIdx)
	require.NoError(t, err)
	assert.False(t, exists)
}

// ============================================================
// Alias Management
// ============================================================

func TestES_AliasManagement(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	aliasName := fmt.Sprintf("test_alias_%d", time.Now().UnixNano())

	// Add alias
	err := eng.(*engine).AddAlias(ctx, aliasName)
	assert.NoError(t, err)

	// Remove alias
	err = eng.(*engine).RemoveAlias(ctx, aliasName)
	assert.NoError(t, err)
}

// ============================================================
// Cluster Health + Index Stats
// ============================================================

func TestES_ClusterHealth(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	health, err := eng.(*engine).ClusterHealth(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, health)
	assert.Contains(t, health, "status")
	t.Logf("cluster status: %v", health["status"])
}

func TestES_IndexStats(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.Index(ctx, search.Doc{ID: "is-1", Type: "article", Fields: map[string]any{"title": "Stats"}})

	stats, err := eng.(*engine).IndexStats(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, stats)
}

// ============================================================
// Multi-Index Search
// ============================================================

func TestES_MultiIndexSearch(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	// Create a second index
	secondIdx := fmt.Sprintf("test_multi_%d", time.Now().UnixNano())
	err := eng.(*engine).CreateIndex(ctx, secondIdx, nil)
	require.NoError(t, err)
	defer eng.(*engine).DeleteIndex(ctx, secondIdx)

	// Index into second index via a new engine pointing to it
	eng2, err := New(Config{
		Addresses:       []string{esAddr(t)},
		IndexName:       secondIdx,
		AutoCreateIndex: false,
		QueryTimeout:    10 * time.Second,
	})
	require.NoError(t, err)
	eng2.Index(ctx, search.Doc{ID: "multi-1", Type: "article", Fields: map[string]any{"title": "Multi Index Doc"}})
	eng2.(search.ExtendedEngine).Refresh(ctx)
	eng2.Close()

	// Search across both indices
	res, err := eng.(*engine).MultiIndexSearch(ctx, []string{eng.(*engine).cfg.IndexName, secondIdx}, search.SearchRequest{
		Keyword: "multi",
		Size:    10,
	})
	require.NoError(t, err)
	assert.Greater(t, res.Total, uint64(0))
}

// ============================================================
// ExtendedEngine interface compliance
// ============================================================

func TestES_ImplementsExtendedEngine(t *testing.T) {
	var _ search.ExtendedEngine = (*engine)(nil)
}
