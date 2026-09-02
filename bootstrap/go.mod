module github.com/LingByte/ling-base/bootstrap

go 1.26.2

require (
	github.com/LingByte/ling-base/common/constants v0.1.1
	github.com/LingByte/ling-base/common/eventbus v0.1.1
	github.com/LingByte/ling-base/version v0.1.0
	github.com/stretchr/testify v1.12.0
)

require (
	github.com/kr/text v0.2.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/LingByte/ling-base => ../

replace github.com/LingByte/ling-base/common/eventbus => ../common/eventbus

replace github.com/LingByte/ling-base/common/constants => ../common/constants

replace github.com/LingByte/ling-base/version => ../version
