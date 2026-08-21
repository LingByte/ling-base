module github.com/LingByte/ling-base/agentkit/agent/n8n

go 1.26.0

replace github.com/LingByte/ling-base/agentkit => ../..

require github.com/LingByte/ling-base/agentkit v0.5.0

require (
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
