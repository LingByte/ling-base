module github.com/LingByte/ling-base/providers/voice/recognizer/std

go 1.26.2

require (
	github.com/LingByte/ling-base/providers/voice/recognizer v0.1.0
	github.com/LingByte/ling-base/providers/voice/recognizer/aws v0.0.0
	github.com/LingByte/ling-base/providers/voice/recognizer/baidu v0.0.0
	github.com/LingByte/ling-base/providers/voice/recognizer/deepgram v0.0.0
	github.com/LingByte/ling-base/providers/voice/recognizer/funasr v0.0.0
	github.com/LingByte/ling-base/providers/voice/recognizer/gladia v0.0.0
	github.com/LingByte/ling-base/providers/voice/recognizer/google v0.0.0
	github.com/LingByte/ling-base/providers/voice/recognizer/local v0.0.0
	github.com/LingByte/ling-base/providers/voice/recognizer/qcloud v0.0.0
	github.com/LingByte/ling-base/providers/voice/recognizer/voiceapi v0.0.0
	github.com/LingByte/ling-base/providers/voice/recognizer/volcengine v0.0.0
	github.com/LingByte/ling-base/providers/voice/recognizer/volcengine_llm v0.0.0
	github.com/LingByte/ling-base/providers/voice/recognizer/whisper v0.0.0
	github.com/sirupsen/logrus v1.9.4
)

require (
	github.com/LingByte/ling-base/common/constants v0.1.1 // indirect
	github.com/LingByte/ling-base/common/logger v0.1.0 // indirect
	github.com/LingByte/ling-base/common/netutil v0.1.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.43.6 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.18 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.36 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.35 // indirect
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
	github.com/aws/aws-sdk-go-v2/service/transcribestreaming v1.38.3 // indirect
	github.com/aws/smithy-go v1.27.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/matoous/go-nanoid v1.5.0 // indirect
	github.com/miekg/dns v1.1.72 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/tencentcloud/tencentcloud-speech-sdk-go v1.0.25 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

replace (
	github.com/LingByte/ling-base/common/constants => ../../../../common/constants
	github.com/LingByte/ling-base/common/logger => ../../../../common/logger
	github.com/LingByte/ling-base/common/netutil => ../../../../common/netutil
	github.com/LingByte/ling-base/providers/voice/recognizer => ../
	github.com/LingByte/ling-base/providers/voice/recognizer/aws => ../aws
	github.com/LingByte/ling-base/providers/voice/recognizer/baidu => ../baidu
	github.com/LingByte/ling-base/providers/voice/recognizer/deepgram => ../deepgram
	github.com/LingByte/ling-base/providers/voice/recognizer/funasr => ../funasr
	github.com/LingByte/ling-base/providers/voice/recognizer/gladia => ../gladia
	github.com/LingByte/ling-base/providers/voice/recognizer/google => ../google
	github.com/LingByte/ling-base/providers/voice/recognizer/local => ../local
	github.com/LingByte/ling-base/providers/voice/recognizer/qcloud => ../qcloud
	github.com/LingByte/ling-base/providers/voice/recognizer/voiceapi => ../voiceapi
	github.com/LingByte/ling-base/providers/voice/recognizer/volcengine => ../volcengine
	github.com/LingByte/ling-base/providers/voice/recognizer/volcengine_llm => ../volcengine_llm
	github.com/LingByte/ling-base/providers/voice/recognizer/whisper => ../whisper
)
