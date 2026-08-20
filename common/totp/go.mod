module github.com/LingByte/ling-base/common/totp

go 1.26.2

require (
	github.com/LingByte/ling-base/common/hash v0.0.0
	github.com/pquerna/otp v1.5.0
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/stretchr/testify v1.12.0
)

require (
	github.com/boombuler/barcode v1.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/LingByte/ling-base/common/hash => ../hash
