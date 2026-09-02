module github.com/LingByte/ling-base/common/totp

go 1.26.2

require (
	github.com/LingByte/ling-base/common/hash v0.0.0
	github.com/pquerna/otp v1.5.0
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/boombuler/barcode v1.1.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

replace github.com/LingByte/ling-base/common/hash => ../hash
