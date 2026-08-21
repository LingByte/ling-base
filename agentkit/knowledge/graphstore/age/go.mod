module github.com/LingByte/ling-base/agentkit/knowledge/graphstore/age

go 1.21

replace (
	github.com/LingByte/ling-base/agentkit => ../../../
	github.com/LingByte/ling-base/agentkit/storage/postgres => ../../../storage/postgres
)

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/LingByte/ling-base/agentkit v1.9.2-0.20260602121024-664ebd0ab56d
	github.com/LingByte/ling-base/agentkit/storage/postgres v1.9.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.1 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)
