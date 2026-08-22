module github.com/LingByte/ling-base/agentkit/model/provider

go 1.26.2

replace (
	github.com/LingByte/ling-base/agentkit => ../../
	github.com/LingByte/ling-base/agentkit/model/anthropic => ../../model/anthropic
	github.com/LingByte/ling-base/agentkit/model/gemini => ../../model/gemini
	github.com/LingByte/ling-base/agentkit/model/hunyuan => ../../model/hunyuan
	github.com/LingByte/ling-base/agentkit/model/ollama => ../../model/ollama
	github.com/LingByte/ling-base/agentkit/model/relay => ../../model/relay
	github.com/LingByte/ling-base/relay => ../../../relay
	github.com/LingByte/ling-base/relay/relaykit => ../../../relay/relaykit
)

require (
	github.com/LingByte/ling-base/agentkit v0.6.0
	github.com/LingByte/ling-base/agentkit/model/anthropic v0.0.0-20251126064502-c8c2594d2519
	github.com/LingByte/ling-base/agentkit/model/gemini v0.8.1-0.20251222024650-ea147adf3d21
	github.com/LingByte/ling-base/agentkit/model/ollama v0.8.0
	github.com/LingByte/ling-base/agentkit/model/relay v0.0.0-00010101000000-000000000000
	github.com/anthropics/anthropic-sdk-go v1.66.0
	github.com/ollama/ollama v0.32.15
	github.com/openai/openai-go v1.12.0
	github.com/stretchr/testify v1.12.1
	google.golang.org/genai v1.69.0
)

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.17.0 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/LingByte/ling-base/common/circuitbreaker v0.1.0 // indirect
	github.com/LingByte/ling-base/common/constants v0.1.1 // indirect
	github.com/LingByte/ling-base/common/logger v0.1.0 // indirect
	github.com/LingByte/ling-base/common/retry v0.1.0 // indirect
	github.com/LingByte/ling-base/relay v0.0.0-00010101000000-000000000000 // indirect
	github.com/LingByte/ling-base/relay/relaykit v0.0.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.7 // indirect
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.22.0 // indirect
	github.com/invopop/jsonschema v0.14.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/pb33f/ordered-map/v2 v2.3.1 // indirect
	github.com/samber/lo v1.47.0 // indirect
	github.com/standard-webhooks/standard-webhooks/libraries v0.0.1 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/otel v1.41.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.29.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.29.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.29.0 // indirect
	go.opentelemetry.io/otel/metric v1.41.0 // indirect
	go.opentelemetry.io/otel/sdk v1.41.0 // indirect
	go.opentelemetry.io/otel/trace v1.41.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.2 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
