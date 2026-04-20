package postgres

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrations embed.FS

var Migrations, _ = fs.Sub(migrations, "migrations")
