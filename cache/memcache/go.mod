module github.com/LingByte/ling-base/cache/memcache

go 1.26.2

require (
	github.com/LingByte/ling-base/cache v0.0.0
	github.com/bradfitz/gomemcache v0.0.0-20260422231931-4d751bb6e37c
)

require github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect

replace github.com/LingByte/ling-base/cache => ../
