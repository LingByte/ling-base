module github.com/LingByte/ling-base/relay

go 1.26.2

require (
	github.com/LingByte/ling-base/common/circuitbreaker v0.1.0
	github.com/LingByte/ling-base/common/logger v0.1.0
	github.com/LingByte/ling-base/common/retry v0.1.0
	github.com/LingByte/ling-base/relay/relaykit v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.43.6
	github.com/aws/aws-sdk-go-v2/credentials v1.19.35
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.57.3
	github.com/aws/smithy-go v1.27.8
	github.com/bytedance/gopkg v0.1.4
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/pkg/errors v0.9.1
	github.com/samber/lo v1.47.0
	github.com/shopspring/decimal v1.4.0
	github.com/stretchr/testify v1.12.0
	github.com/tidwall/sjson v1.2.5
	go.uber.org/zap v1.28.0
)

require (
	github.com/LingByte/ling-base/common/constants v0.1.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.37 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.37 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/LingByte/ling-base/relay/relaykit => ./relaykit
