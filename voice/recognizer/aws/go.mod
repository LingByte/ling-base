module github.com/LingByte/ling-base/voice/recognizer/aws

go 1.26.2

require (
	github.com/LingByte/ling-base/common/logger v0.1.0
	github.com/LingByte/ling-base/voice/recognizer v0.1.1
	github.com/aws/aws-sdk-go-v2/config v1.32.36
	github.com/aws/aws-sdk-go-v2/credentials v1.19.35
	github.com/aws/aws-sdk-go-v2/service/transcribestreaming v1.38.3
)

require (
	github.com/LingByte/ling-base/common/constants v0.1.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.43.6 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.18 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.37 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.37 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.37 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.5.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.33.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.38.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.45.5 // indirect
	github.com/aws/smithy-go v1.27.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
)

replace github.com/LingByte/ling-base/voice/recognizer => ../

replace github.com/LingByte/ling-base/common/constants => ../../../common/constants
