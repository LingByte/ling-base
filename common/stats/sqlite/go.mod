module github.com/LingByte/ling-base/common/stats/sqlite

go 1.26.2

require (
	github.com/LingByte/ling-base/common/stats v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/stats/memory v0.1.0
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/stretchr/testify v1.12.0
)

require (
	github.com/axiomhq/hyperloglog v0.2.3 // indirect
	github.com/dgryski/go-metro v0.0.0-20180109044635-280f6062b5bc // indirect
	github.com/kamstrup/intmap v0.5.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/LingByte/ling-base/common/stats => ../
