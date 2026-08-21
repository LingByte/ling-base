module github.com/LingByte/ling-base/common/config

go 1.26.2

require (
	github.com/LingByte/ling-base/common/cache v0.1.0
	github.com/stretchr/testify v1.12.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.2
)

require (
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	golang.org/x/text v0.41.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/LingByte/ling-base/common/cache => ../cache
