package migrations

import "embed"

// FS contains goose SQL migrations for goalrail-server.
//
//go:embed *.sql
var FS embed.FS
