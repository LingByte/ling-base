module github.com/LingByte/ling-base/common/bloom/redisbloom

go 1.26.2

require (
	github.com/LingByte/ling-base/common/bloom v0.0.0
	github.com/redis/go-redis/v9 v9.22.0
	github.com/stretchr/testify v1.12.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/LingByte/ling-base/common/bloom => ../
