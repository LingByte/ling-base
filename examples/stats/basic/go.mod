module github.com/LingByte/ling-base/examples/stats/basic

go 1.26.2

require (
	github.com/LingByte/ling-base/common/stats v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/stats/memory v0.0.0-00010101000000-000000000000
)

require (
	github.com/LingByte/ling-base/common/stats/file v0.0.0-00010101000000-000000000000 // indirect
	github.com/axiomhq/hyperloglog v0.2.3 // indirect
	github.com/dgryski/go-metro v0.0.0-20180109044635-280f6062b5bc // indirect
	github.com/kamstrup/intmap v0.5.1 // indirect
)

replace (
	github.com/LingByte/ling-base/common/stats => ../../../common/stats
	github.com/LingByte/ling-base/common/stats/file => ../../../common/stats/file
	github.com/LingByte/ling-base/common/stats/memory => ../../../common/stats/memory
)
