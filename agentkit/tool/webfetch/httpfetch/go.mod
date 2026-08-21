module github.com/LingByte/ling-base/agentkit/tool/webfetch/httpfetch

go 1.26.0

replace github.com/LingByte/ling-base/agentkit => ../../../

require (
	github.com/JohannesKaufmann/html-to-markdown/v2 v2.4.0
	github.com/LingByte/ling-base/agentkit v0.8.1-0.20251222024650-ea147adf3d21
	github.com/dslipak/pdf v0.0.2
	github.com/go-pdf/fpdf v0.9.0
	github.com/stretchr/testify v1.12.1
	golang.org/x/net v0.46.0
)

require (
	github.com/JohannesKaufmann/dom v0.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
