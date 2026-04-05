package migrations

import "embed"

//go:embed psql/*.sql
var EmbedPSQLMigrations embed.FS
