module github.com/LingByte/ling-base/common/archive

go 1.26.2

replace github.com/LingByte/ling-base/common/compress => ../compress

require (
	github.com/LingByte/ling-base/common/compress v0.1.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.29 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)
