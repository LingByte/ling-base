module github.com/LingByte/ling-base/common/chunk

go 1.26.2

require github.com/LingByte/ling-base/relay v0.1.1

require (
	github.com/LingByte/ling-base/common/circuitbreaker v0.1.0 // indirect
	github.com/LingByte/ling-base/common/constants v0.1.1 // indirect
	github.com/LingByte/ling-base/common/logger v0.1.0 // indirect
	github.com/LingByte/ling-base/common/netutil v0.1.0 // indirect
	github.com/LingByte/ling-base/common/retry v0.1.0 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
)

replace github.com/LingByte/ling-base/relay => ../../relay
