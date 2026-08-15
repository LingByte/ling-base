module github.com/LingByte/ling-base/cache/ristretto

go 1.26.2

require (
	github.com/LingByte/ling-base/cache v0.0.0
	github.com/dgraph-io/ristretto/v2 v2.4.2
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/LingByte/ling-base/cache => ../
