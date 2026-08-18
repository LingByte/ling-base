module github.com/LingByte/ling-base/common/stats/mysql

go 1.26.2

require (
	github.com/LingByte/ling-base/common/stats v0.0.0-00010101000000-000000000000
	github.com/go-sql-driver/mysql v1.10.0
)

require filippo.io/edwards25519 v1.2.0 // indirect

replace github.com/LingByte/ling-base/common/stats => ../
