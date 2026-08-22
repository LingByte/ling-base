module github.com/LingByte/ling-base/agentkit/relaymodel

go 1.26.2

replace (
	github.com/LingByte/ling-base/agentkit => ../
	github.com/LingByte/ling-base/relay => ../../relay
	github.com/LingByte/ling-base/relay/compat => ../../relay/compat
	github.com/LingByte/ling-base/relay/relaykit => ../../relay/relaykit
)

require (
	github.com/LingByte/ling-base/agentkit v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/relay v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/relay/compat v0.0.0-00010101000000-000000000000
)

require (
	github.com/LingByte/ling-base/common/circuitbreaker v0.1.0 // indirect
	github.com/LingByte/ling-base/common/constants v0.1.1 // indirect
	github.com/LingByte/ling-base/common/logger v0.1.0 // indirect
	github.com/LingByte/ling-base/common/retry v0.1.0 // indirect
	github.com/LingByte/ling-base/relay/relaykit v0.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/samber/lo v1.47.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)
