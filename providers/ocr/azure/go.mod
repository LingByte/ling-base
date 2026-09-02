module github.com/LingByte/ling-base/providers/ocr/azure

go 1.26.2

replace github.com/LingByte/ling-base/providers/ocr => ../

require (
	github.com/LingByte/ling-base/providers/ocr v0.0.0
	github.com/stretchr/testify v1.12.1
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect
