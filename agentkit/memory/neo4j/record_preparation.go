package neo4jstore

import (
	"sort"
	"time"

	"github.com/LingByte/ling-base/agentkit/memory/gomodel"
)

// rescoreMemoryRecords normalizes backend results using the same prepared
// cosine query and matrix-aware scoring used by the local vector stores.
func rescoreMemoryRecords(records []gomodel.MemoryRecord, queryEmbedding []float32, limit int) []gomodel.MemoryRecord {
	query := gomodel.NewCosineQuery(queryEmbedding)
	for i := range records {
		if len(records[i].Embedding) == 0 && len(records[i].EmbeddingMatrix) == 0 {
			continue
		}
		records[i].Score = query.MaxSimilarity(records[i])
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Score > records[j].Score
	})
	if limit > 0 && len(records) > limit {
		return records[:limit]
	}
	return records
}

// mergeBackendCosineScores retains the cosine score already calculated by a
// vector database and only evaluates auxiliary matrix embeddings that the
// backend did not index. Backend results are already ordered, so sorting is
// skipped unless a matrix actually changes a score.
func mergeBackendCosineScores(records []gomodel.MemoryRecord, queryEmbedding []float32, limit int) []gomodel.MemoryRecord {
	query := gomodel.NewCosineQuery(queryEmbedding)
	changed := false
	for i := range records {
		for _, vector := range records[i].EmbeddingMatrix {
			if len(vector) == 0 {
				continue
			}
			if score := query.Similarity(vector); score > records[i].Score {
				records[i].Score = score
				changed = true
			}
		}
	}
	if changed {
		sort.SliceStable(records, func(i, j int) bool {
			return records[i].Score > records[j].Score
		})
	}
	if limit > 0 && len(records) > limit {
		return records[:limit]
	}
	return records
}

// prepareMemoryRecord applies the backend-independent normalization needed
// before a memory is serialized. Persistent stores historically include a
// missing space in metadata; the in-memory store only defaults record.Space.
func prepareMemoryRecord(
	sessionID, content string,
	metadata map[string]any,
	embedding []float32,
	now time.Time,
	persistDefaultSpace bool,
) gomodel.MemoryRecord {
	if persistDefaultSpace {
		if metadata == nil {
			metadata = map[string]any{}
		}
		if _, ok := metadata["space"]; !ok {
			metadata["space"] = sessionID
		}
	}

	importance, source, summary, lastEmbedded, metadataJSON := gomodel.NormalizeMetadata(metadata, now)
	normalizedMetadata := gomodel.DecodeMetadata(metadataJSON)
	space := gomodel.StringFromAny(normalizedMetadata["space"])
	if space == "" {
		space = sessionID
	}

	matrix := gomodel.ValidEmbeddingMatrix(normalizedMetadata)
	storedEmbedding := append([]float32(nil), embedding...)
	if len(storedEmbedding) == 0 {
		for _, vector := range matrix {
			if len(vector) == 0 {
				continue
			}
			storedEmbedding = append([]float32(nil), vector...)
			break
		}
	}

	return gomodel.MemoryRecord{
		SessionID:       sessionID,
		Space:           space,
		Content:         content,
		Metadata:        metadataJSON,
		Embedding:       storedEmbedding,
		EmbeddingMatrix: matrix,
		Importance:      importance,
		Source:          source,
		Summary:         summary,
		CreatedAt:       now,
		LastEmbedded:    lastEmbedded,
		GraphEdges:      gomodel.ValidGraphEdges(normalizedMetadata),
	}
}
