module github.com/LingByte/ling-base/agentkit/knowledge/vectorstore/elasticsearch

go 1.26.0

replace (
	github.com/LingByte/ling-base/agentkit => ../../../
	github.com/LingByte/ling-base/agentkit/storage/elasticsearch => ../../../storage/elasticsearch
)

require (
	github.com/LingByte/ling-base/agentkit v0.2.0
	github.com/LingByte/ling-base/agentkit/storage/elasticsearch v0.2.0
	github.com/elastic/go-elasticsearch/v9 v9.1.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/elastic/elastic-transport-go/v8 v8.7.0 // indirect
	github.com/elastic/go-elasticsearch/v7 v7.17.10 // indirect
	github.com/elastic/go-elasticsearch/v8 v8.19.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)
