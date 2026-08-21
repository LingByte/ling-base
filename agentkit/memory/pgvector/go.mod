module github.com/LingByte/ling-base/agentkit/memory/pgvector

go 1.26.0

replace (
	github.com/LingByte/ling-base/agentkit => ../..
	github.com/LingByte/ling-base/agentkit/storage/postgres => ../../storage/postgres
)

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/LingByte/ling-base/agentkit v1.1.1
	github.com/LingByte/ling-base/agentkit/storage/postgres v1.1.1
	github.com/lib/pq v1.10.9
	github.com/pgvector/pgvector-go v0.2.3
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/go-ego/gse v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/vcaesar/cedar v0.20.2 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
