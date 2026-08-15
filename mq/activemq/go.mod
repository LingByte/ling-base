module github.com/LingByte/ling-base/mq/activemq

go 1.26.2

require (
	github.com/LingByte/ling-base/mq v0.0.0-00010101000000-000000000000
	github.com/go-stomp/stomp/v3 v3.1.5
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/LingByte/ling-base/mq => ../
