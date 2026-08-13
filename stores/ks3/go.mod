module github.com/LingByte/ling-base/stores/ks3

go 1.26.2

require github.com/LingByte/ling-base/stores v0.0.0

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/ks3sdklib/aws-sdk-go v1.12.0
	golang.org/x/crypto v0.54.0 // indirect
)

replace github.com/LingByte/ling-base/stores => ../
