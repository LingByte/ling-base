module github.com/LingByte/ling-base/providers/ocr/qcloud

go 1.26.2

replace github.com/LingByte/ling-base/providers/ocr => ../

require (
	github.com/LingByte/ling-base/providers/ocr v0.0.0
	github.com/stretchr/testify v1.12.1
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.3.164
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ocr v1.3.159
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect
