package gomemory

import (
	"strings"

	embedpkg "github.com/LingByte/ling-base/agentkit/memory/goembed"
	memengine "github.com/LingByte/ling-base/agentkit/memory/goengine"
	"github.com/LingByte/ling-base/agentkit/memory/gomodel"
	sessionpkg "github.com/LingByte/ling-base/agentkit/memory/gosession"
	markdownpkg "github.com/LingByte/ling-base/agentkit/memory/markdown"
	storepkg "github.com/LingByte/ling-base/agentkit/memory/neo4j"
)

// Type aliases preserving the original public API.
type (
	Engine              = memengine.Engine
	Options             = memengine.Options
	ScoreWeights        = memengine.ScoreWeights
	Metrics             = memengine.Metrics
	MetricsSnapshot     = memengine.MetricsSnapshot
	Summarizer          = memengine.Summarizer
	HeuristicSummarizer = memengine.HeuristicSummarizer

	MemoryRecord = gomodel.MemoryRecord
	GraphEdge    = gomodel.GraphEdge
	EdgeType     = gomodel.EdgeType

	MemoryBank    = sessionpkg.MemoryBank
	SessionMemory = sessionpkg.SessionMemory
	SpaceRegistry = sessionpkg.SpaceRegistry
	SpaceRole     = sessionpkg.SpaceRole
	Space         = sessionpkg.Space
	SharedSession = sessionpkg.SharedSession

	VectorStore       = storepkg.VectorStore
	SchemaInitializer = storepkg.SchemaInitializer
	GraphStore        = storepkg.GraphStore

	InMemoryStore           = storepkg.InMemoryStore
	PostgresStore           = storepkg.PostgresStore
	QdrantStore             = storepkg.QdrantStore
	Neo4jStore              = storepkg.Neo4jStore
	MongoStore              = storepkg.MongoStore
	Distance                = storepkg.Distance
	CreateCollectionRequest = storepkg.CreateCollectionRequest

	Embedder      = embedpkg.Embedder
	DummyEmbedder = embedpkg.DummyEmbedder
)
type MarkdownStore = markdownpkg.Store
type MarkdownRecord = markdownpkg.Record

var NewMarkdownStore = markdownpkg.NewStore

const (
	EdgeFollows     = gomodel.EdgeFollows
	EdgeExplains    = gomodel.EdgeExplains
	EdgeContradicts = gomodel.EdgeContradicts
	EdgeDerivedFrom = gomodel.EdgeDerivedFrom

	SpaceRoleReader = sessionpkg.SpaceRoleReader
	SpaceRoleWriter = sessionpkg.SpaceRoleWriter
	SpaceRoleAdmin  = sessionpkg.SpaceRoleAdmin
)

var (
	ErrNotSupported = embedpkg.ErrNotSupported

	NewEngine              = memengine.NewEngine
	DefaultOptions         = memengine.DefaultOptions
	NewMemoryBank          = sessionpkg.NewMemoryBank
	NewMemoryBankWithStore = sessionpkg.NewMemoryBankWithStore
	NewSessionMemory       = sessionpkg.NewSessionMemory
	NewSharedSession       = sessionpkg.NewSharedSession
	NewSpaceRegistry       = sessionpkg.NewSpaceRegistry

	AutoEmbedder        = embedpkg.AutoEmbedder
	DummyEmbedding      = embedpkg.DummyEmbedding
	NewOpenAIEmbedder   = embedpkg.NewOpenAIEmbedder
	NewVertexAIEmbedder = embedpkg.NewVertexAIEmbedder
	NewOllamaEmbedder   = embedpkg.NewOllamaEmbedder
	NewFastEmbeed       = embedpkg.NewFastEmbeed
	NewClaudeEmbedder   = embedpkg.NewClaudeEmbedder

	NewInMemoryStore = storepkg.NewInMemoryStore
	NewPostgresStore = storepkg.NewPostgresStore
	NewQdrantStore   = storepkg.NewQdrantStore
	NewNeo4jStore    = storepkg.NewNeo4jStore
	NewMongoStore    = storepkg.NewMongoStore
)

// ChunkText splits long text into roughly `chunkSize` rune segments
// along line or paragraph boundaries for better embedding quality.
func ChunkText(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		chunkSize = 2000
	}

	lines := strings.Split(text, "\n")
	var (
		chunks []string
		buf    strings.Builder
		size   int
	)
	for _, line := range lines {
		if size+len(line) > chunkSize {
			chunks = append(chunks, buf.String())
			buf.Reset()
			size = 0
		}
		buf.WriteString(line)
		buf.WriteString("\n")
		size += len(line)
	}
	if buf.Len() > 0 {
		chunks = append(chunks, buf.String())
	}
	if len(chunks) == 0 {
		chunks = []string{text}
	}
	return chunks
}
