module github.com/LingByte/ling-base/common/notification/sms

go 1.26.2

require (
	github.com/LingByte/ling-base/common/hash v0.0.0
	github.com/LingByte/ling-base/common/notification v0.1.1
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/LingByte/ling-base/common/notification/httpclient v0.1.0
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

replace github.com/LingByte/ling-base/common/hash => ../../hash

replace github.com/LingByte/ling-base/common/notification/httpclient => ../httpclient
