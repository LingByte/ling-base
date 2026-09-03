module github.com/LingByte/ling-base/voice/recognizer

go 1.26.2

require (
	github.com/LingByte/ling-base/common/logger v0.1.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
)

require (
	github.com/LingByte/ling-base/common/constants v0.1.0 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

replace github.com/LingByte/ling-base/common/constants => ../../common/constants
