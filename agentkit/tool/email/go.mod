module github.com/LingByte/ling-base/agentkit/tool/email

go 1.26.0

replace github.com/LingByte/ling-base/agentkit => ../..

require (
	github.com/LingByte/ling-base/agentkit v0.5.0
	github.com/stretchr/testify v1.12.1
	github.com/wneessen/go-mail v0.7.2
)

require (
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/text v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
