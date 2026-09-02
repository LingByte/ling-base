module github.com/LingByte/ling-base/stores/ks3

go 1.26.2

require github.com/LingByte/ling-base/stores v0.0.0

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/ks3sdklib/aws-sdk-go v1.12.0
	golang.org/x/crypto v0.55.0 // indirect
)

replace github.com/LingByte/ling-base/stores => ../
