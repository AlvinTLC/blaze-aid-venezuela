// Package migrations embeds the SQL schema files so the binary can self-migrate
// on boot without shipping the .sql files or relying on docker initdb.d.
package migrations

import "embed"

// FS holds every migrations/*.sql, applied in lexical order by internal/migrate.
//
//go:embed *.sql
var FS embed.FS
