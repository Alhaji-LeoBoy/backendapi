package migrations

import "embed"

//go:embed *.sql
var FileSystem embed.FS
