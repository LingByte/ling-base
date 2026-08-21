module github.com/LingByte/ling-base/agentkit/memory/mysql

go 1.26.0

replace (
	github.com/LingByte/ling-base/agentkit => ../..
	github.com/LingByte/ling-base/agentkit/storage/mysql => ../../storage/mysql
)

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/LingByte/ling-base/agentkit v0.0.0-20251126064502-c8c2594d2519
	github.com/LingByte/ling-base/agentkit/storage/mysql v0.0.0-20251126064502-c8c2594d2519
	github.com/go-sql-driver/mysql v1.9.3
	github.com/stretchr/testify v1.12.1
)

require (
	filippo.io/edwards25519 v1.1.1 // indirect
	github.com/go-ego/gse v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/vcaesar/cedar v0.20.2 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
