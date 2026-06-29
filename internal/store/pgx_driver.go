//go:build pgx

// Build with: go get github.com/jackc/pgx/v5 && go build -tags pgx ./...
// Registers the pgx stdlib driver so store.NewPostgres("pgx", DATABASE_URL)
// connects. The default (untagged) build needs no Postgres dependency.
package store

import _ "github.com/jackc/pgx/v5/stdlib"
