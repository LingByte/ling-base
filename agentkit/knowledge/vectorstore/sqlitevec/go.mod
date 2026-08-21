module github.com/LingByte/ling-base/agentkit/knowledge/vectorstore/sqlitevec

go 1.26.0

replace github.com/LingByte/ling-base/agentkit => ../../..

require (
	github.com/LingByte/ling-base/agentkit v0.2.0
	github.com/asg017/sqlite-vec-go-bindings v0.1.6
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/ncruces/go-sqlite3 v0.17.1 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/tetratelabs/wazero v1.7.3 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.37.0 // indirect
)
