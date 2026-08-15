module github.com/LingByte/ling-base/bootstrap

go 1.26.2

require (
	github.com/LingByte/ling-base/eventbus v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/eventbus/memory v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/version v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/LingByte/ling-base => ../
	github.com/LingByte/ling-base/eventbus => ../eventbus
	github.com/LingByte/ling-base/eventbus/memory => ../eventbus/memory
	github.com/LingByte/ling-base/version => ../version
)
