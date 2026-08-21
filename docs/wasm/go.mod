module github.com/LingByte/ling-base/docs/wasm

go 1.26.2

require (
	github.com/LingByte/ling-base/common/compress v0.1.0
	github.com/LingByte/ling-base/common/password v0.1.0
	github.com/LingByte/ling-base/common/totp v0.1.0
)

require (
	github.com/LingByte/ling-base/common/hash v0.0.0 // indirect
	github.com/boombuler/barcode v1.1.0 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.29 // indirect
	github.com/pquerna/otp v1.5.0 // indirect
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace (
	github.com/LingByte/ling-base/common/compress => ../../common/compress
	github.com/LingByte/ling-base/common/hash => ../../common/hash
	github.com/LingByte/ling-base/common/password => ../../common/password
	github.com/LingByte/ling-base/common/totp => ../../common/totp
)
