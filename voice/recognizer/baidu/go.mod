module github.com/LingByte/ling-base/voice/recognizer/baidu

go 1.26.2

require (
	github.com/LingByte/ling-base/common/logger v0.1.0
	github.com/LingByte/ling-base/common/netutil v0.1.0
	github.com/LingByte/ling-base/voice/recognizer v0.1.1
	github.com/google/uuid v1.6.0
)

require (
	github.com/LingByte/ling-base/common/constants v0.1.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
)

replace github.com/LingByte/ling-base/voice/recognizer => ../

replace github.com/LingByte/ling-base/common/constants => ../../../common/constants
