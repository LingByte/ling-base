module github.com/LingByte/ling-base/common/knowledge

go 1.26.2

require (
	github.com/LingByte/ling-base/common/retrieve v0.0.0
	github.com/LingByte/ling-base/common/search v0.1.0
	github.com/LingByte/ling-base/common/utils v0.0.0
	github.com/LingByte/ling-base/providers/embedder v0.0.0
	github.com/alibabacloud-go/bailian-20231229/v2 v2.11.1
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.1.14
	github.com/alibabacloud-go/tea v1.3.13
	github.com/alibabacloud-go/tea-utils/v2 v2.0.7
	github.com/jackc/pgx/v5 v5.10.0
	github.com/milvus-io/milvus-sdk-go/v2 v2.4.2
	github.com/pgvector/pgvector-go v0.4.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/aliyun/credentials-go v1.4.5 // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/cockroachdb/errors v1.9.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20211118104740-dabe8e521a4f // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/getsentry/sentry-go v0.12.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/milvus-io/milvus-proto/go-api/v2 v2.4.10-0.20240819025435-512e3b98866a // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto v0.0.0-20220503193339-ba3ae3f07e29 // indirect
	google.golang.org/grpc v1.48.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)

replace (
	github.com/LingByte/ling-base/common/retrieve => ../retrieve
	github.com/LingByte/ling-base/common/search => ../search
	github.com/LingByte/ling-base/common/utils => ../utils
	github.com/LingByte/ling-base/providers/embedder => ../../providers/embedder
)
