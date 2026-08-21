# search

Full-text search engine abstraction built on top of Bleve, supporting indexing, batch indexing, deletion, complex queries, facets, highlighting, and auto-complete suggestions.

## Key types

- `Engine` — the full-text search interface (`Index`, `IndexBatch`, `Delete`, `Search`, suggestions, `DocCount`, `Stats`, `Close`)
- `ExtendedEngine` — additional operations (`GetByID`, `Update`, `BulkDelete`, `DeleteByQuery`, `Scroll`, `SearchAfter`, `Refresh`, `HealthCheck`)
- `Doc` — a document to index (ID, Type, Fields)
- `SearchRequest` — query with keyword, term filters, range filters, match/phrase/prefix/wildcard/regex/fuzzy clauses, facets, sorting, pagination, highlighting
- `SearchResult` / `Hit` / `FacetResult` — search responses
- `Config` — index path, analyzer, default search fields, timeouts, batch size

## Quick start

```go
import "github.com/LingByte/ling-base/common/search"

eng, err := search.NewDefault("/tmp/myindex")
if err != nil {
    log.Fatal(err)
}
defer eng.Close()

eng.Index(ctx, search.Doc{
    ID:   "doc1",
    Type: "article",
    Fields: map[string]any{
        "title":   "Hello World",
        "content": "This is a test document",
    },
})

res, _ := eng.Search(ctx, search.SearchRequest{Keyword: "hello", Size: 10})
for _, hit := range res.Hits {
    fmt.Println(hit.ID, hit.Score)
}
```
