// Package migrations embeds the *.up.sql files for cmd/migrate.
package migrations

import "embed"

// UpFS holds the ordered upgrade scripts (lexical filename order).
//
//go:embed *.up.sql
var UpFS embed.FS
