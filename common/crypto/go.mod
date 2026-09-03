module github.com/LingByte/ling-base/common/crypto

go 1.26.2

require (
	github.com/LingByte/ling-base/common/hash v0.1.0
	github.com/stretchr/testify v1.12.1
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

replace github.com/LingByte/ling-base/common/hash => ../hash
