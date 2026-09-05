module github.com/LingByte/ling-base/common/registry/consul

go 1.26.2

require (
	github.com/LingByte/ling-base/common/registry v0.0.0
	github.com/hashicorp/consul/api v1.31.2
)

replace github.com/LingByte/ling-base/common/registry => ../
