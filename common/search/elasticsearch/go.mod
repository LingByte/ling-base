module github.com/LingByte/ling-base/common/search/elasticsearch

go 1.26.2

require (
	github.com/LingByte/ling-base/common/search v0.1.0
	github.com/elastic/go-elasticsearch/v8 v8.15.0
	github.com/stretchr/testify v1.12.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.6.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/LingByte/ling-base/common/search => ../
