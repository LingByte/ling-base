module github.com/LingByte/ling-base/mq/kafka

go 1.26.2

replace github.com/LingByte/ling-base/mq => ../

require (
	github.com/LingByte/ling-base/mq v0.0.0-00010101000000-000000000000
	github.com/segmentio/kafka-go v0.4.50
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
