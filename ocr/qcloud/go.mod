module github.com/LingByte/ling-base/ocr/qcloud

go 1.26.2

replace github.com/LingByte/ling-base/ocr => ../

require (
	github.com/LingByte/ling-base/ocr v0.0.0
	github.com/stretchr/testify v1.12.0
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.3.164
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ocr v1.3.159
)

require (
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
