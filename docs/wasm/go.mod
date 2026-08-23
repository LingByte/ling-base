module github.com/LingByte/ling-base/docs/wasm

go 1.26.2

require (
	github.com/LingByte/ling-base/common/barcode v0.1.0
	github.com/LingByte/ling-base/common/bloom v0.1.0
	github.com/LingByte/ling-base/common/bloom/memory v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/captcha v0.1.0
	github.com/LingByte/ling-base/common/compress v0.1.0
	github.com/LingByte/ling-base/common/convert v0.1.0
	github.com/LingByte/ling-base/common/crypto v0.2.0
	github.com/LingByte/ling-base/common/hash v0.1.0
	github.com/LingByte/ling-base/common/i18n v0.1.0
	github.com/LingByte/ling-base/common/idgen v0.1.0
	github.com/LingByte/ling-base/common/jwtutil v0.1.0
	github.com/LingByte/ling-base/common/mathutil v0.1.0
	github.com/LingByte/ling-base/common/netutil v0.1.0
	github.com/LingByte/ling-base/common/nltime v0.1.0
	github.com/LingByte/ling-base/common/password v0.1.0
	github.com/LingByte/ling-base/common/phone v0.1.0
	github.com/LingByte/ling-base/common/pinyin v0.1.0
	github.com/LingByte/ling-base/common/qrcode v0.1.0
	github.com/LingByte/ling-base/common/random v0.1.0
	github.com/LingByte/ling-base/common/timeutil v0.1.0
	github.com/LingByte/ling-base/common/totp v0.1.0
	github.com/LingByte/ling-base/common/validate v0.1.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/LingByte/ling-base/common/imageutil v0.1.0 // indirect
	github.com/boombuler/barcode v1.1.0 // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/makiuchi-d/gozxing v0.1.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.29 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pquerna/otp v1.5.0 // indirect
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e // indirect
	github.com/yeqown/go-qrcode/v2 v2.3.0 // indirect
	github.com/yeqown/go-qrcode/writer/standard v1.4.0 // indirect
	github.com/yeqown/reedsolomon v1.0.0 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/LingByte/ling-base/common/barcode => ../../common/barcode
	github.com/LingByte/ling-base/common/bloom => ../../common/bloom
	github.com/LingByte/ling-base/common/bloom/memory => ../../common/bloom/memory
	github.com/LingByte/ling-base/common/captcha => ../../common/captcha
	github.com/LingByte/ling-base/common/compress => ../../common/compress
	github.com/LingByte/ling-base/common/convert => ../../common/convert
	github.com/LingByte/ling-base/common/crypto => ../../common/crypto
	github.com/LingByte/ling-base/common/hash => ../../common/hash
	github.com/LingByte/ling-base/common/i18n => ../../common/i18n
	github.com/LingByte/ling-base/common/idgen => ../../common/idgen
	github.com/LingByte/ling-base/common/imageutil => ../../common/imageutil
	github.com/LingByte/ling-base/common/jwtutil => ../../common/jwtutil
	github.com/LingByte/ling-base/common/mathutil => ../../common/mathutil
	github.com/LingByte/ling-base/common/netutil => ../../common/netutil
	github.com/LingByte/ling-base/common/nltime => ../../common/nltime
	github.com/LingByte/ling-base/common/password => ../../common/password
	github.com/LingByte/ling-base/common/phone => ../../common/phone
	github.com/LingByte/ling-base/common/pinyin => ../../common/pinyin
	github.com/LingByte/ling-base/common/qrcode => ../../common/qrcode
	github.com/LingByte/ling-base/common/random => ../../common/random
	github.com/LingByte/ling-base/common/timeutil => ../../common/timeutil
	github.com/LingByte/ling-base/common/totp => ../../common/totp
	github.com/LingByte/ling-base/common/validate => ../../common/validate
)
