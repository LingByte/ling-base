module github.com/LingByte/ling-base/common/validate

go 1.26.2

replace github.com/LingByte/ling-base/common/convert => ../convert

require (
	github.com/LingByte/ling-base/common/convert v0.1.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
