module github.com/LingByte/ling-base/common/tree/sqlite

go 1.26.2

require (
	github.com/LingByte/ling-base/common/tree/gormstore v0.0.0
	gorm.io/driver/sqlite v1.5.7
	gorm.io/gorm v1.31.2
)

replace github.com/LingByte/ling-base/common/tree/gormstore => ../gormstore
