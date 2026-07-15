// Package migrations embeds the SQL migration files so the API binary can
// apply them at startup without needing the files on disk at runtime.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
