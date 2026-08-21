module github.com/LingByte/ling-base/agentkit/memory/redis

go 1.26.0

replace (
	github.com/LingByte/ling-base/agentkit => ../../
	github.com/LingByte/ling-base/agentkit/storage/redis => ../../storage/redis
)

require (
	github.com/LingByte/ling-base/agentkit v0.2.0
	github.com/LingByte/ling-base/agentkit/storage/redis v0.2.0
	github.com/alicebob/miniredis/v2 v2.35.0
	github.com/redis/go-redis/v9 v9.11.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-ego/gse v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/vcaesar/cedar v0.20.2 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
