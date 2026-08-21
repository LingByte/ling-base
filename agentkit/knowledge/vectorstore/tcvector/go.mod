module github.com/LingByte/ling-base/agentkit/knowledge/vectorstore/tcvector

go 1.26.0

replace (
	github.com/LingByte/ling-base/agentkit => ../../../
	github.com/LingByte/ling-base/agentkit/storage/tcvector => ../../../storage/tcvector
)

require (
	github.com/LingByte/ling-base/agentkit v0.2.0
	github.com/LingByte/ling-base/agentkit/storage/tcvector v0.0.4
	github.com/stretchr/testify v1.12.1
	github.com/tencent/vectordatabase-sdk-go v1.8.0
)

require (
	github.com/clbanning/mxj v1.8.4 // indirect
	github.com/go-ego/gse v1.0.0 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mozillazg/go-httpheader v0.2.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/tencentyun/cos-go-sdk-v5 v0.7.69 // indirect
	github.com/vcaesar/cedar v0.20.2 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251002232023-7c0ddcbb5797 // indirect
	google.golang.org/grpc v1.75.1 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
