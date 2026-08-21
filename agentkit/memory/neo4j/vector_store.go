package neo4jstore

import (
	"context"
	"time"

	"github.com/LingByte/ling-base/agentkit/memory/gomodel"
)

// VectorStore defines the contract for long-term memory backends.
type VectorStore interface {
	StoreMemory(ctx context.Context, sessionID, content string, metadata map[string]any, embedding []float32) error
	SearchMemory(ctx context.Context, sessionID string, queryEmbedding []float32, limit int) ([]gomodel.MemoryRecord, error)
	UpdateEmbedding(ctx context.Context, id int64, embedding []float32, lastEmbedded time.Time) error
	DeleteMemory(ctx context.Context, ids []int64) error
	Iterate(ctx context.Context, fn func(gomodel.MemoryRecord) bool) error
	Count(ctx context.Context) (int, error)
}

// SchemaInitializer allows stores to expose optional schema/bootstrap routines.
type SchemaInitializer interface {
	CreateSchema(ctx context.Context, schemaPath string) error
}

// GraphStore is implemented by vector stores that maintain graph neighborhoods for memories.
type GraphStore interface {
	UpsertGraph(ctx context.Context, record gomodel.MemoryRecord, edges []gomodel.GraphEdge) error
	Neighborhood(ctx context.Context, sessionID string, seedIDs []int64, hops, limit int) ([]gomodel.MemoryRecord, error)
}
