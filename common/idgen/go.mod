module github.com/LingByte/ling-base/common/idgen

go 1.26.2

replace github.com/LingByte/ling-base/common/random => ../random

require (
	github.com/LingByte/ling-base/common/random v0.1.0
	github.com/stretchr/testify v1.12.1
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect
