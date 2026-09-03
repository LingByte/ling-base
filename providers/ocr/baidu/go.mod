module github.com/LingByte/ling-base/providers/ocr/baidu

go 1.26.2

replace github.com/LingByte/ling-base/providers/ocr => ../

require (
	github.com/LingByte/ling-base/common/netutil v0.1.0
	github.com/LingByte/ling-base/providers/ocr v0.0.0
	github.com/stretchr/testify v1.12.1
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect
