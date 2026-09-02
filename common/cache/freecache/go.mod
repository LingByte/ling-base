module github.com/LingByte/ling-base/common/cache/freecache

go 1.26.2

require (
	github.com/LingByte/ling-base/common/cache v0.0.0
	github.com/coocood/freecache v1.2.7
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
)

replace github.com/LingByte/ling-base/common/cache => ../
