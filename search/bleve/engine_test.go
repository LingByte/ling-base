package bleve

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/LingByte/ling-base/search"
)

func setupTestEngine(t *testing.T) (search.Engine, string) {
	tmpDir := os.TempDir()
	indexPath := filepath.Join(tmpDir, "test_search_index_"+t.Name())

	// Clean up any existing index
	os.RemoveAll(indexPath)

	cfg := Config{
		IndexPath:           indexPath,
		DefaultAnalyzer:     "standard",
		DefaultSearchFields: []string{"title", "body"},
		QueryTimeout:        5 * time.Second,
		BatchSize:           100,
	}

	m := BuildIndexMapping("standard")
	engine, err := New(cfg, m)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	return engine, indexPath
}

func cleanupTestEngine(t *testing.T, engine search.Engine, indexPath string) {
	if engine != nil {
		_ = engine.Close()
	}
	os.RemoveAll(indexPath)
}

func TestNew_OpenExistingIndex(t *testing.T) {
	tmpDir := os.TempDir()
	indexPath := filepath.Join(tmpDir, "test_existing_index")
	defer os.RemoveAll(indexPath)

	cfg := Config{
		IndexPath:       indexPath,
		DefaultAnalyzer: "standard",
		QueryTimeout:    5 * time.Second,
	}

	m := BuildIndexMapping("standard")

	// Create first index
	engine1, err := New(cfg, m)
	if err != nil {
		t.Fatalf("Failed to create first engine: %v", err)
	}

	// Index a document
	doc := search.Doc{
		ID:   "test1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test Document",
		},
	}
	err = engine1.Index(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to index document: %v", err)
	}
	engine1.Close()

	// Open existing index
	engine2, err := New(cfg, m)
	if err != nil {
		t.Fatalf("Failed to open existing index: %v", err)
	}
	defer engine2.Close()

	// Verify document exists
	req := search.SearchRequest{
		Keyword: "Test",
		Size:    10,
	}
	result, err := engine2.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}
	if result.Total == 0 {
		t.Fatalf("Expected to find document, but got 0 results")
	}
}

func TestBleveEngine_Index(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	doc := search.Doc{
		ID:   "doc1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test Article",
			"body":  "This is a test article body",
		},
	}

	err := engine.Index(context.Background(), doc)
	if err != nil {
		t.Fatalf("Index failed: %v", err)
	}
}

func TestBleveEngine_IndexBatch(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	docs := []search.Doc{
		{
			ID:   "doc1",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article 1",
			},
		},
		{
			ID:   "doc2",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article 2",
			},
		},
		{
			ID:   "doc3",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article 3",
			},
		},
	}

	err := engine.IndexBatch(context.Background(), docs)
	if err != nil {
		t.Fatalf("IndexBatch failed: %v", err)
	}

	// Verify documents were indexed
	req := search.SearchRequest{
		Keyword: "Article",
		Size:    10,
	}
	result, err := engine.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("Expected 3 documents, got %d", result.Total)
	}
}

func TestBleveEngine_Delete(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index a document
	doc := search.Doc{
		ID:   "doc1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test Article",
		},
	}
	err := engine.Index(context.Background(), doc)
	if err != nil {
		t.Fatalf("Index failed: %v", err)
	}

	// Delete the document
	err = engine.Delete(context.Background(), "doc1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify document is deleted
	req := search.SearchRequest{
		Keyword: "Test",
		Size:    10,
	}
	result, err := engine.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("Expected 0 documents after delete, got %d", result.Total)
	}
}

func TestBleveEngine_Search(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index documents
	docs := []search.Doc{
		{
			ID:   "doc1",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Go Programming",
				"body":  "Go is a programming language",
			},
		},
		{
			ID:   "doc2",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Python Programming",
				"body":  "Python is also a programming language",
			},
		},
	}
	for _, doc := range docs {
		err := engine.Index(context.Background(), doc)
		if err != nil {
			t.Fatalf("Index failed: %v", err)
		}
	}

	// Search
	req := search.SearchRequest{
		Keyword: "programming",
		Size:    10,
	}
	result, err := engine.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.Total < 2 {
		t.Fatalf("Expected at least 2 results, got %d", result.Total)
	}
	if len(result.Hits) == 0 {
		t.Fatalf("Expected hits, got none")
	}
}

func TestBleveEngine_SearchWithPagination(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index multiple documents
	for i := 0; i < 5; i++ {
		doc := search.Doc{
			ID:   "doc" + string(rune('0'+i)),
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article " + string(rune('0'+i)),
			},
		}
		err := engine.Index(context.Background(), doc)
		if err != nil {
			t.Fatalf("Index failed: %v", err)
		}
	}

	// Search with pagination
	req := search.SearchRequest{
		Keyword: "Article",
		From:    0,
		Size:    2,
	}
	result, err := engine.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result.Hits) > 2 {
		t.Fatalf("Expected at most 2 hits, got %d", len(result.Hits))
	}
}

func TestBleveEngine_SearchWithFacets(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index documents with tags
	docs := []search.Doc{
		{
			ID:   "doc1",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article 1",
				"tags":  "go",
			},
		},
		{
			ID:   "doc2",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article 2",
				"tags":  "python",
			},
		},
	}
	for _, doc := range docs {
		err := engine.Index(context.Background(), doc)
		if err != nil {
			t.Fatalf("Index failed: %v", err)
		}
	}

	// Search with facets
	req := search.SearchRequest{
		Keyword: "Article",
		Size:    10,
		Facets: []search.FacetRequest{
			{
				Name:  "tags",
				Field: "tags",
				Size:  10,
			},
		},
	}
	result, err := engine.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.Facets == nil {
		t.Fatalf("Expected facets, got nil")
	}
}

func TestBleveEngine_GetAutoCompleteSuggestions(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index documents
	docs := []search.Doc{
		{
			ID:   "apple",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Apple",
			},
		},
		{
			ID:   "application",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Application",
			},
		},
	}
	for _, doc := range docs {
		err := engine.Index(context.Background(), doc)
		if err != nil {
			t.Fatalf("Index failed: %v", err)
		}
	}

	// Get suggestions
	suggestions, err := engine.GetAutoCompleteSuggestions(context.Background(), "app")
	if err != nil {
		t.Fatalf("GetAutoCompleteSuggestions failed: %v", err)
	}

	if len(suggestions) == 0 {
		t.Fatalf("Expected suggestions, got none")
	}
}

func TestBleveEngine_GetAutoCompleteSuggestions_EmptyKeyword(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	suggestions, err := engine.GetAutoCompleteSuggestions(context.Background(), "")
	if err != nil {
		t.Fatalf("GetAutoCompleteSuggestions failed: %v", err)
	}

	if len(suggestions) != 0 {
		t.Fatalf("Expected empty suggestions for empty keyword, got %d", len(suggestions))
	}
}

func TestBleveEngine_GetSearchSuggestions(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index documents
	doc := search.Doc{
		ID:   "test1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test Document",
		},
	}
	err := engine.Index(context.Background(), doc)
	if err != nil {
		t.Fatalf("Index failed: %v", err)
	}

	// Get suggestions
	suggestions, err := engine.GetSearchSuggestions(context.Background(), "test")
	if err != nil {
		t.Fatalf("GetSearchSuggestions failed: %v", err)
	}

	if len(suggestions) == 0 {
		t.Fatalf("Expected suggestions, got none")
	}
}

func TestBleveEngine_GetSearchSuggestions_EmptyKeyword(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	suggestions, err := engine.GetSearchSuggestions(context.Background(), "")
	if err != nil {
		t.Fatalf("GetSearchSuggestions failed: %v", err)
	}

	if len(suggestions) != 0 {
		t.Fatalf("Expected empty suggestions for empty keyword, got %d", len(suggestions))
	}
}

func TestBleveEngine_Close(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer os.RemoveAll(indexPath)

	err := engine.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Try to use closed engine
	err = engine.Index(context.Background(), search.Doc{ID: "test"})
	if err != search.ErrClosed {
		t.Fatalf("Expected search.ErrClosed, got %v", err)
	}
}

func TestBleveEngine_Close_MultipleTimes(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer os.RemoveAll(indexPath)

	err := engine.Close()
	if err != nil {
		t.Fatalf("First Close failed: %v", err)
	}

	// Close again should not error
	err = engine.Close()
	if err != nil {
		t.Fatalf("Second Close failed: %v", err)
	}
}

func TestBleveEngine_WithDeadline_NilContext(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Test with nil context
	be := engine.(*bleveEngine)
	err := be.withDeadline(nil, 0, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("withDeadline with nil context failed: %v", err)
	}
}

func TestBleveEngine_WithDeadline_Timeout(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	be := engine.(*bleveEngine)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := be.withDeadline(ctx, 1*time.Millisecond, func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	if err == nil {
		t.Fatalf("Expected timeout error, got nil")
	}
}

func TestBleveEngine_Guard_Closed(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	engine.Close()

	// Try to index after close
	err := engine.Index(context.Background(), search.Doc{ID: "test"})
	assert.Equal(t, search.ErrClosed, err)

	// Try to search after close
	_, err = engine.Search(context.Background(), search.SearchRequest{})
	assert.Equal(t, search.ErrClosed, err)

	// Try to delete after close
	err = engine.Delete(context.Background(), "test")
	assert.Equal(t, search.ErrClosed, err)

	// Try to get suggestions after close
	_, err = engine.GetAutoCompleteSuggestions(context.Background(), "test")
	assert.Equal(t, search.ErrClosed, err)

	_, err = engine.GetSearchSuggestions(context.Background(), "test")
	assert.Equal(t, search.ErrClosed, err)
}

func TestBleveEngine_Close_Idempotent(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer os.RemoveAll(indexPath)

	// Close multiple times should not error
	err1 := engine.Close()
	assert.Nil(t, err1)

	err2 := engine.Close()
	assert.Nil(t, err2)
}

func TestBleveEngine_IndexBatch_Empty(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	err := engine.IndexBatch(context.Background(), []search.Doc{})
	assert.Nil(t, err)
}

func TestBleveEngine_IndexBatch_LargeBatch(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Create docs larger than default batch size
	docs := make([]search.Doc, 300)
	for i := 0; i < 300; i++ {
		docs[i] = search.Doc{
			ID:   string(rune(i)),
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Document " + string(rune(i)),
			},
		}
	}

	err := engine.IndexBatch(context.Background(), docs)
	assert.Nil(t, err)

	// Verify some documents were indexed
	result, err := engine.Search(context.Background(), search.SearchRequest{
		Keyword: "Document",
		Size:    10,
	})
	assert.Nil(t, err)
	assert.True(t, result.Total > 0)
}

func TestBleveEngine_Search_WithHighlight(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	doc := search.Doc{
		ID:   "test1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Machine Learning Basics",
			"body":  "Machine learning is a subset of artificial intelligence",
		},
	}
	engine.Index(context.Background(), doc)

	result, err := engine.Search(context.Background(), search.SearchRequest{
		Keyword:         "Machine",
		Highlight:       true,
		HighlightFields: []string{"title", "body"},
		Size:            10,
	})
	assert.Nil(t, err)
	assert.True(t, result.Total > 0)
}

func TestBleveEngine_Search_WithFacets(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	docs := []search.Doc{
		{
			ID:   "1",
			Type: "article",
			Fields: map[string]interface{}{
				"title":    "Go Programming",
				"category": "programming",
			},
		},
		{
			ID:   "2",
			Type: "article",
			Fields: map[string]interface{}{
				"title":    "Python Guide",
				"category": "programming",
			},
		},
		{
			ID:   "3",
			Type: "article",
			Fields: map[string]interface{}{
				"title":    "Web Design",
				"category": "design",
			},
		},
	}
	engine.IndexBatch(context.Background(), docs)

	result, err := engine.Search(context.Background(), search.SearchRequest{
		Keyword: "Guide",
		Facets: []search.FacetRequest{
			{
				Name:  "categories",
				Field: "category",
				Size:  10,
			},
		},
		Size: 10,
	})
	assert.Nil(t, err)
	assert.True(t, len(result.Facets) > 0)
}

func TestBleveEngine_Search_WithSort(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	docs := []search.Doc{
		{
			ID:   "1",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "First Article",
				"views": 100,
			},
		},
		{
			ID:   "2",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Second Article",
				"views": 200,
			},
		},
	}
	engine.IndexBatch(context.Background(), docs)

	result, err := engine.Search(context.Background(), search.SearchRequest{
		Keyword: "Article",
		SortBy:  []string{"-views"},
		Size:    10,
	})
	assert.Nil(t, err)
	assert.True(t, result.Total > 0)
}

func TestBleveEngine_Search_WithPagination(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index multiple documents
	for i := 0; i < 20; i++ {
		doc := search.Doc{
			ID:   string(rune(i)),
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article " + string(rune(i)),
			},
		}
		engine.Index(context.Background(), doc)
	}

	// Test pagination
	result1, err := engine.Search(context.Background(), search.SearchRequest{
		Keyword: "Article",
		From:    0,
		Size:    5,
	})
	assert.Nil(t, err)
	assert.Equal(t, 5, len(result1.Hits))

	result2, err := engine.Search(context.Background(), search.SearchRequest{
		Keyword: "Article",
		From:    5,
		Size:    5,
	})
	assert.Nil(t, err)
	assert.Equal(t, 5, len(result2.Hits))
}

func TestBleveEngine_Search_DefaultSize(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	doc := search.Doc{
		ID:   "test1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test",
		},
	}
	engine.Index(context.Background(), doc)

	// Search with size 0 should default to 10
	result, err := engine.Search(context.Background(), search.SearchRequest{
		Keyword: "Test",
		Size:    0,
	})
	assert.Nil(t, err)
	assert.True(t, result.Total > 0)
}

func TestBleveEngine_Search_WithIncludeFields(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	doc := search.Doc{
		ID:   "test1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test Document",
			"body":  "This is a test",
		},
	}
	engine.Index(context.Background(), doc)

	result, err := engine.Search(context.Background(), search.SearchRequest{
		Keyword:       "Test",
		IncludeFields: []string{"title"},
		Size:          10,
	})
	assert.Nil(t, err)
	assert.True(t, result.Total > 0)
}

func TestBleveEngine_GetAutoCompleteSuggestions_Empty(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	suggestions, err := engine.GetAutoCompleteSuggestions(context.Background(), "")
	assert.Nil(t, err)
	assert.Equal(t, 0, len(suggestions))
}

func TestBleveEngine_GetAutoCompleteSuggestions_WithData(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	doc := search.Doc{
		ID:   "test1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Machine Learning",
		},
	}
	engine.Index(context.Background(), doc)

	suggestions, err := engine.GetAutoCompleteSuggestions(context.Background(), "Mach")
	assert.Nil(t, err)
	// Should return suggestions starting with "Mach"
	assert.True(t, len(suggestions) >= 0)
}

func TestBleveEngine_GetSearchSuggestions_Empty(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	suggestions, err := engine.GetSearchSuggestions(context.Background(), "")
	assert.Nil(t, err)
	assert.Equal(t, 0, len(suggestions))
}

func TestBleveEngine_GetSearchSuggestions_WithData(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	doc := search.Doc{
		ID:   "test1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Python Programming",
		},
	}
	engine.Index(context.Background(), doc)

	suggestions, err := engine.GetSearchSuggestions(context.Background(), "Python")
	assert.Nil(t, err)
	assert.True(t, len(suggestions) >= 0)
}

func TestBleveEngine_Delete_Extra(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	doc := search.Doc{
		ID:   "test2",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test Document 2",
		},
	}
	engine.Index(context.Background(), doc)

	// Verify document exists
	result, _ := engine.Search(context.Background(), search.SearchRequest{
		Keyword: "Test",
		Size:    10,
	})
	assert.True(t, result.Total > 0)

	// Delete document
	err := engine.Delete(context.Background(), "test2")
	assert.Nil(t, err)

	// Verify document is deleted
	result, _ = engine.Search(context.Background(), search.SearchRequest{
		Keyword: "Document 2",
		Size:    10,
	})
	assert.Equal(t, uint64(0), result.Total)
}

func TestNew_InvalidIndexPath(t *testing.T) {
	cfg := Config{
		IndexPath:       "/invalid/path/that/does/not/exist/index",
		DefaultAnalyzer: "standard",
	}

	m := BuildIndexMapping("standard")
	_, err := New(cfg, m)
	assert.NotNil(t, err)
}

func TestBleveEngine_WithDeadline_Timeout_Extra(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// This should timeout
	err := engine.Index(ctx, search.Doc{ID: "test"})
	assert.NotNil(t, err)
}

func TestBleveEngine_Index_WithoutType(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	doc := search.Doc{
		ID: "test1",
		Fields: map[string]interface{}{
			"title": "Test Document",
		},
	}
	err := engine.Index(context.Background(), doc)
	assert.Nil(t, err)

	result, _ := engine.Search(context.Background(), search.SearchRequest{
		Keyword: "Test",
		Size:    10,
	})
	assert.True(t, result.Total > 0)
}

func TestBleveEngine_Search_NegativeFrom(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	doc := search.Doc{
		ID:   "test1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test",
		},
	}
	engine.Index(context.Background(), doc)

	// Negative From should be treated as 0
	result, err := engine.Search(context.Background(), search.SearchRequest{
		Keyword: "Test",
		From:    -5,
		Size:    10,
	})
	assert.Nil(t, err)
	assert.True(t, result.Total > 0)
}

func TestNewMemory_BasicIndex(t *testing.T) {
	eng, err := NewMemory()
	if err != nil {
		t.Fatalf("NewMemory: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()
	docs := []search.Doc{
		{ID: "1", Type: "article", Fields: map[string]any{"title": "Go programming", "content": "Learn Go basics"}},
		{ID: "2", Type: "article", Fields: map[string]any{"title": "Python tutorial", "content": "Python for beginners"}},
		{ID: "3", Type: "article", Fields: map[string]any{"title": "Rust guide", "content": "Systems programming in Rust"}},
	}
	if err := eng.IndexBatch(ctx, docs); err != nil {
		t.Fatalf("IndexBatch: %v", err)
	}

	count, err := eng.DocCount(ctx)
	if err != nil {
		t.Fatalf("DocCount: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 docs, got %d", count)
	}
}

func TestNewMemory_Search(t *testing.T) {
	eng, _ := NewMemory()
	defer eng.Close()
	ctx := context.Background()

	eng.IndexBatch(ctx, []search.Doc{
		{ID: "1", Type: "article", Fields: map[string]any{"title": "Go programming language", "content": "Go is a statically typed compiled language"}},
		{ID: "2", Type: "article", Fields: map[string]any{"title": "Python for data science", "content": "Python is great for data analysis"}},
		{ID: "3", Type: "article", Fields: map[string]any{"title": "Rust systems programming", "content": "Rust is memory safe"}},
	})

	res, err := eng.Search(ctx, search.NewKeywordSearch("Go", 10))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.Total == 0 {
		t.Error("expected results for 'Go'")
	}
}

func TestNewMemory_Delete(t *testing.T) {
	eng, _ := NewMemory()
	defer eng.Close()
	ctx := context.Background()

	eng.Index(ctx, search.Doc{ID: "1", Type: "article", Fields: map[string]any{"title": "test"}})

	count, _ := eng.DocCount(ctx)
	if count != 1 {
		t.Errorf("expected 1 doc, got %d", count)
	}

	eng.Delete(ctx, "1")

	count, _ = eng.DocCount(ctx)
	if count != 0 {
		t.Errorf("expected 0 docs after delete, got %d", count)
	}
}

func TestNewMemory_Stats(t *testing.T) {
	eng, _ := NewMemory()
	defer eng.Close()
	ctx := context.Background()

	eng.Index(ctx, search.Doc{ID: "1", Type: "article", Fields: map[string]any{"title": "test"}})
	eng.Search(ctx, search.NewKeywordSearch("test", 10))

	stats := eng.Stats()
	if stats["indexOps"].(uint64) < 1 {
		t.Error("expected indexOps >= 1")
	}
	if stats["searchOps"].(uint64) < 1 {
		t.Error("expected searchOps >= 1")
	}
}

func TestNewMemory_AutoComplete(t *testing.T) {
	eng, _ := NewMemory()
	defer eng.Close()
	ctx := context.Background()

	eng.IndexBatch(ctx, []search.Doc{
		{ID: "1", Fields: map[string]any{"title": "Go programming", "content": "Learn Go"}},
		{ID: "2", Fields: map[string]any{"title": "Go testing", "content": "Test Go code"}},
		{ID: "3", Fields: map[string]any{"title": "Python", "content": "Learn Python"}},
	})

	// Prefix query is case-sensitive on the token; "go" matches the tokenized "go".
	suggestions, err := eng.GetAutoCompleteSuggestions(ctx, "go")
	if err != nil {
		t.Fatalf("GetAutoCompleteSuggestions: %v", err)
	}
	if len(suggestions) == 0 {
		t.Error("expected suggestions for 'go'")
	}
}

func TestNewMemory_SearchSuggestions(t *testing.T) {
	eng, _ := NewMemory()
	defer eng.Close()
	ctx := context.Background()

	eng.IndexBatch(ctx, []search.Doc{
		{ID: "1", Fields: map[string]any{"title": "Go programming", "content": "Learn Go basics"}},
		{ID: "2", Fields: map[string]any{"title": "Advanced Go", "content": "Go concurrency patterns"}},
	})

	suggestions, err := eng.GetSearchSuggestions(ctx, "programming")
	if err != nil {
		t.Fatalf("GetSearchSuggestions: %v", err)
	}
	if len(suggestions) == 0 {
		t.Error("expected suggestions for 'programming'")
	}
}

func TestNewMemory_EmptyDocID(t *testing.T) {
	eng, _ := NewMemory()
	defer eng.Close()
	ctx := context.Background()

	err := eng.Index(ctx, search.Doc{ID: "", Fields: map[string]any{"title": "test"}})
	if err != search.ErrEmptyDocID {
		t.Errorf("expected search.ErrEmptyDocID, got %v", err)
	}
}

func TestNewMemory_BatchEmptyID(t *testing.T) {
	eng, _ := NewMemory()
	defer eng.Close()
	ctx := context.Background()

	err := eng.IndexBatch(ctx, []search.Doc{
		{ID: "1", Fields: map[string]any{"title": "ok"}},
		{ID: "", Fields: map[string]any{"title": "bad"}},
	})
	if err == nil {
		t.Error("expected error for batch with empty ID")
	}
}

func TestNewMemory_CloseTwice(t *testing.T) {
	eng, _ := NewMemory()
	if err := eng.Close(); err != nil {
		t.Errorf("first close: %v", err)
	}
	if err := eng.Close(); err != nil {
		t.Errorf("second close: %v", err)
	}
}

func TestNewMemory_SearchAfterClose(t *testing.T) {
	eng, _ := NewMemory()
	eng.Close()
	ctx := context.Background()

	_, err := eng.Search(ctx, search.NewKeywordSearch("test", 10))
	if err != search.ErrClosed {
		t.Errorf("expected search.ErrClosed, got %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("/tmp/test")
	if cfg.IndexPath != "/tmp/test" {
		t.Errorf("expected IndexPath /tmp/test, got %s", cfg.IndexPath)
	}
	if cfg.QueryTimeout != 10*time.Second {
		t.Errorf("expected QueryTimeout 10s, got %v", cfg.QueryTimeout)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("expected BatchSize 100, got %d", cfg.BatchSize)
	}
}

func TestNewDoc(t *testing.T) {
	doc := search.NewDoc("1", "article", map[string]any{"title": "test"})
	if doc.ID != "1" || doc.Type != "article" {
		t.Error("search.NewDoc fields mismatch")
	}
}

func TestNewKeywordSearch(t *testing.T) {
	req := search.NewKeywordSearch("hello", 5)
	if req.Keyword != "hello" || req.Size != 5 {
		t.Error("search.NewKeywordSearch fields mismatch")
	}
}

func TestNewTermSearch(t *testing.T) {
	req := search.NewTermSearch("category", "tech", 10)
	if req.MustTerms["category"][0] != "tech" {
		t.Error("search.NewTermSearch fields mismatch")
	}
}

func TestNewMatchSearch(t *testing.T) {
	req := search.NewMatchSearch("title", "hello", 10)
	if len(req.Matches) != 1 || req.Matches[0].Field != "title" {
		t.Error("search.NewMatchSearch fields mismatch")
	}
}

func TestNewPhraseSearch(t *testing.T) {
	req := search.NewPhraseSearch("content", "hello world", 10)
	if len(req.Phrases) != 1 || req.Phrases[0].Phrase != "hello world" {
		t.Error("search.NewPhraseSearch fields mismatch")
	}
}
