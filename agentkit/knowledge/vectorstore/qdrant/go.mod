module github.com/LingByte/ling-base/agentkit/knowledge/vectorstore/qdrant

go 1.26.0

replace (
	github.com/LingByte/ling-base/agentkit => ../../../
	github.com/LingByte/ling-base/agentkit/storage/qdrant => ../../../storage/qdrant
)

require (
	github.com/LingByte/ling-base/agentkit v0.8.0
	github.com/LingByte/ling-base/agentkit/storage/qdrant v1.1.2-0.20260108033914-7a20241f1ad5
	github.com/google/uuid v1.6.0
	github.com/qdrant/go-client v1.16.0
	github.com/stretchr/testify v1.12.1
	google.golang.org/grpc v1.79.3
)

require (
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
