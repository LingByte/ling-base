module github.com/LingByte/ling-base/agentkit/knowledge/vectorstore/sqlitevec

go 1.24.0

toolchain go1.24.11

replace github.com/LingByte/ling-base/agentkit => ../../..

require (
	github.com/asg017/sqlite-vec-go-bindings v0.1.6
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/stretchr/testify v1.11.1
	github.com/LingByte/ling-base/agentkit v0.2.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ncruces/go-sqlite3 v0.17.1 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tetratelabs/wazero v1.7.3 // indirect
	golang.org/x/sys v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
