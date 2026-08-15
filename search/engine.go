package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

// Sentinel errors for the search engine.
var (
	ErrClosed       = errors.New("search engine closed")
	ErrEmptyDocID   = errors.New("document id cannot be empty")
	ErrNilDoc       = errors.New("document cannot be nil")
	ErrIndexNotOpen = errors.New("index is not open")
)

// Engine defines the full-text search interface.
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

// bleveEngine implements Engine using Bleve.
type bleveEngine struct {
	cfg           Config
	index         bleve.Index
	defaultFields []string
	mu            sync.RWMutex
	closed        bool
	stats         engineStats
}

// engineStats tracks runtime statistics.
type engineStats struct {
	IndexOps  uint64
	SearchOps uint64
	DeleteOps uint64
	BatchOps  uint64
	mu        sync.Mutex
}

func (s *engineStats) incIndex()  { s.mu.Lock(); s.IndexOps++; s.mu.Unlock() }
func (s *engineStats) incSearch() { s.mu.Lock(); s.SearchOps++; s.mu.Unlock() }
func (s *engineStats) incDelete() { s.mu.Lock(); s.DeleteOps++; s.mu.Unlock() }
func (s *engineStats) incBatch()  { s.mu.Lock(); s.BatchOps++; s.mu.Unlock() }

// New creates a new Engine backed by a Bleve index at cfg.IndexPath.
// If the path doesn't exist a new index is created; if it exists the index is opened.
func New(cfg Config, m mapping.IndexMapping) (Engine, error) {
	be := &bleveEngine{cfg: cfg, defaultFields: cfg.DefaultSearchFields}

	var idx bleve.Index
	if _, err := os.Stat(cfg.IndexPath); err == nil {
		i, e := bleve.Open(cfg.IndexPath)
		if e != nil {
			return nil, fmt.Errorf("open index at %s: %w", cfg.IndexPath, e)
		}
		idx = i
	} else if os.IsNotExist(err) {
		if m == nil {
			m = BuildIndexMapping(cfg.DefaultAnalyzer)
		}
		i, e := bleve.New(cfg.IndexPath, m)
		if e != nil {
			return nil, fmt.Errorf("create index at %s: %w", cfg.IndexPath, e)
		}
		idx = i
	} else {
		return nil, fmt.Errorf("stat index path %s: %w", cfg.IndexPath, err)
	}
	be.index = idx
	return be, nil
}

// NewDefault creates a new Engine with a default configuration and mapping.
// The index is stored on disk at indexPath.
func NewDefault(indexPath string) (Engine, error) {
	cfg := DefaultConfig(indexPath)
	return New(cfg, nil)
}

// NewMemory creates a new Engine backed by an in-memory index (no disk persistence).
// This is useful for testing and ephemeral search scenarios.
func NewMemory() (Engine, error) {
	cfg := Config{
		IndexPath:    "",
		QueryTimeout: 5 * time.Second,
		BatchSize:    100,
	}
	m := BuildIndexMapping(cfg.DefaultAnalyzer)
	idx, err := bleve.NewMemOnly(m)
	if err != nil {
		return nil, fmt.Errorf("create in-memory index: %w", err)
	}
	return &bleveEngine{
		cfg:   cfg,
		index: idx,
	}, nil
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(indexPath string) Config {
	return Config{
		IndexPath:       indexPath,
		DefaultAnalyzer: "",
		QueryTimeout:    10 * time.Second,
		BatchSize:       100,
	}
}

func (e *bleveEngine) guard() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.closed {
		return ErrClosed
	}
	return nil
}

func (e *bleveEngine) withDeadline(ctx context.Context, d time.Duration, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if d <= 0 {
		return fn(ctx)
	}
	c, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	ch := make(chan error, 1)
	go func() {
		ch <- fn(c)
	}()
	select {
	case <-c.Done():
		return c.Err()
	case err := <-ch:
		return err
	}
}

func (e *bleveEngine) Index(ctx context.Context, doc Doc) error {
	if err := e.guard(); err != nil {
		return err
	}
	if doc.ID == "" {
		return ErrEmptyDocID
	}
	e.stats.incIndex()
	return e.withDeadline(ctx, e.cfg.QueryTimeout, func(ctx context.Context) error {
		data := make(map[string]any, len(doc.Fields)+1)
		for k, v := range doc.Fields {
			data[k] = v
		}
		if doc.Type != "" {
			data["type"] = doc.Type
		}
		return e.index.Index(doc.ID, data)
	})
}

func (e *bleveEngine) IndexBatch(ctx context.Context, docs []Doc) error {
	if err := e.guard(); err != nil {
		return err
	}
	for _, d := range docs {
		if d.ID == "" {
			return fmt.Errorf("batch contains document with empty id")
		}
	}
	e.stats.incBatch()
	bs := e.cfg.BatchSize
	if bs <= 0 {
		bs = 100
	}
	return e.withDeadline(ctx, 0, func(ctx context.Context) error {
		for i := 0; i < len(docs); i += bs {
			end := i + bs
			if end > len(docs) {
				end = len(docs)
			}
			b := e.index.NewBatch()
			for _, d := range docs[i:end] {
				data := make(map[string]any, len(d.Fields)+1)
				for k, v := range d.Fields {
					data[k] = v
				}
				if d.Type != "" {
					data["type"] = d.Type
				}
				if err := b.Index(d.ID, data); err != nil {
					return fmt.Errorf("batch index doc %s: %w", d.ID, err)
				}
			}
			if err := e.index.Batch(b); err != nil {
				return fmt.Errorf("batch commit: %w", err)
			}
		}
		return nil
	})
}

func (e *bleveEngine) Delete(ctx context.Context, id string) error {
	if err := e.guard(); err != nil {
		return err
	}
	if id == "" {
		return ErrEmptyDocID
	}
	e.stats.incDelete()
	return e.withDeadline(ctx, e.cfg.QueryTimeout, func(ctx context.Context) error {
		return e.index.Delete(id)
	})
}

func (e *bleveEngine) Search(ctx context.Context, req SearchRequest) (SearchResult, error) {
	if err := e.guard(); err != nil {
		return SearchResult{}, err
	}
	e.stats.incSearch()

	q := buildQuery(req, e.defaultFields)
	sr := bleve.NewSearchRequest(q)

	// Pagination
	if req.Size <= 0 {
		req.Size = 10
	}
	if req.From < 0 {
		req.From = 0
	}
	sr.Size = req.Size
	sr.From = req.From

	// Sorting
	if len(req.SortBy) > 0 {
		sr.SortBy(req.SortBy)
	}

	// Fields
	if len(req.IncludeFields) == 0 {
		sr.Fields = []string{"*"}
	} else {
		sr.Fields = req.IncludeFields
	}

	// Highlight
	if req.Highlight {
		hl := bleve.NewHighlightWithStyle("html")
		sr.Highlight = hl
	}

	// Facets
	if len(req.Facets) > 0 {
		sr.Facets = make(map[string]*bleve.FacetRequest, len(req.Facets))
		for _, f := range req.Facets {
			size := f.Size
			if size <= 0 {
				size = 10
			}
			sr.Facets[f.Name] = bleve.NewFacetRequest(f.Field, size)
		}
	}

	var res *bleve.SearchResult
	err := e.withDeadline(ctx, e.cfg.QueryTimeout, func(ctx context.Context) error {
		r, e2 := e.index.Search(sr)
		if e2 != nil {
			return e2
		}
		res = r
		return nil
	})
	if err != nil {
		return SearchResult{}, err
	}

	out := SearchResult{
		Total:  res.Total,
		Took:   res.Took,
		Hits:   make([]Hit, 0, len(res.Hits)),
		Facets: map[string]FacetResult{},
	}
	for _, h := range res.Hits {
		out.Hits = append(out.Hits, Hit{
			ID:        h.ID,
			Score:     h.Score,
			Fields:    h.Fields,
			Fragments: h.Fragments,
		})
	}
	if res.Facets != nil {
		for name, fr := range res.Facets {
			ft := FacetResult{Total: fr.Total}
			if fr.Terms != nil {
				for _, t := range fr.Terms.Terms() {
					ft.Terms = append(ft.Terms, FacetTerm{Term: t.Term, Count: t.Count})
				}
			}
			out.Facets[name] = ft
		}
	}
	return out, nil
}

func (e *bleveEngine) DocCount(ctx context.Context) (uint64, error) {
	if err := e.guard(); err != nil {
		return 0, err
	}
	var count uint64
	err := e.withDeadline(ctx, e.cfg.QueryTimeout, func(ctx context.Context) error {
		c, err := e.index.DocCount()
		if err != nil {
			return err
		}
		count = c
		return nil
	})
	return count, err
}

func (e *bleveEngine) Stats() map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := map[string]any{
		"closed": e.closed,
	}
	e.stats.mu.Lock()
	stats["indexOps"] = e.stats.IndexOps
	stats["searchOps"] = e.stats.SearchOps
	stats["deleteOps"] = e.stats.DeleteOps
	stats["batchOps"] = e.stats.BatchOps
	e.stats.mu.Unlock()

	if e.index != nil && !e.closed {
		if bleveStats := e.index.Stats(); bleveStats != nil {
			if data, err := json.Marshal(bleveStats); err == nil {
				var bleveMap map[string]any
				if json.Unmarshal(data, &bleveMap) == nil {
					stats["bleve"] = bleveMap
				}
			}
		}
	}
	return stats
}

func (e *bleveEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil
	}
	e.closed = true
	if e.index != nil {
		return e.index.Close()
	}
	return nil
}

func (e *bleveEngine) GetAutoCompleteSuggestions(ctx context.Context, keyword string) ([]string, error) {
	if err := e.guard(); err != nil {
		return nil, err
	}
	if keyword == "" {
		return []string{}, nil
	}

	var suggestions []string
	err := e.withDeadline(ctx, e.cfg.QueryTimeout, func(ctx context.Context) error {
		query := bleve.NewPrefixQuery(keyword)
		sr := bleve.NewSearchRequest(query)
		sr.Size = 10
		sr.Fields = []string{"title", "name", "content"}

		searchResult, err := e.index.Search(sr)
		if err != nil {
			return err
		}

		suggestions = make([]string, 0, len(searchResult.Hits))
		for _, hit := range searchResult.Hits {
			// Try to extract a meaningful field for the suggestion.
			if hit.Fields != nil {
				for _, field := range []string{"title", "name", "content"} {
					if v, ok := hit.Fields[field].(string); ok && v != "" {
						suggestions = append(suggestions, v)
						break
					}
				}
			}
			if len(suggestions) == 0 || suggestions[len(suggestions)-1] == "" {
				// Fallback to document ID.
				if len(suggestions) > 0 && suggestions[len(suggestions)-1] == "" {
					suggestions = suggestions[:len(suggestions)-1]
				}
				suggestions = append(suggestions, hit.ID)
			}
		}
		return nil
	})
	return suggestions, err
}

func (e *bleveEngine) GetSearchSuggestions(ctx context.Context, keyword string) ([]string, error) {
	if err := e.guard(); err != nil {
		return nil, err
	}
	if keyword == "" {
		return []string{}, nil
	}

	var suggestions []string
	err := e.withDeadline(ctx, e.cfg.QueryTimeout, func(ctx context.Context) error {
		query := bleve.NewMatchQuery(keyword)
		sr := bleve.NewSearchRequest(query)
		sr.Size = 10
		sr.Fields = []string{"title", "name", "content"}

		searchResult, err := e.index.Search(sr)
		if err != nil {
			return err
		}

		suggestions = make([]string, 0, len(searchResult.Hits))
		for _, hit := range searchResult.Hits {
			if hit.Fields != nil {
				for _, field := range []string{"title", "name", "content"} {
					if v, ok := hit.Fields[field].(string); ok && v != "" {
						suggestions = append(suggestions, v)
						break
					}
				}
			}
			if len(suggestions) == 0 || suggestions[len(suggestions)-1] == "" {
				if len(suggestions) > 0 && suggestions[len(suggestions)-1] == "" {
					suggestions = suggestions[:len(suggestions)-1]
				}
				suggestions = append(suggestions, hit.ID)
			}
		}
		return nil
	})
	return suggestions, err
}
