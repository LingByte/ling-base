module github.com/LingByte/ling-base/relay/compat

go 1.26.2

replace (
	github.com/LingByte/ling-base/common/constants => ../../common/constants
	github.com/LingByte/ling-base/common/logger => ../../common/logger
)

require github.com/LingByte/ling-base/common/logger v0.1.0

require (
	github.com/LingByte/ling-base/common/constants v0.1.0 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
)
