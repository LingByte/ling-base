module github.com/LingByte/ling-base/providers/censor/qiniu

go 1.26.2

require (
	github.com/LingByte/ling-base/providers/censor v0.0.0
	github.com/qiniu/go-sdk/v7 v7.27.0
)

require github.com/BurntSushi/toml v1.6.0 // indirect

replace github.com/LingByte/ling-base/providers/censor => ../
