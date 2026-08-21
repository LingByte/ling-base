module github.com/LingByte/ling-base/agentkit/model/tiktoken

go 1.26.0

replace github.com/LingByte/ling-base/agentkit => ../../

require (
	github.com/LingByte/ling-base/agentkit v0.2.1
	github.com/stretchr/testify v1.12.1
	github.com/tiktoken-go/tokenizer v0.7.0
)

require (
	github.com/dlclark/regexp2 v1.11.5 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)
