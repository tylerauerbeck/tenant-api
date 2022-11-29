// Package dbschema provides an embedded filesystem containing all the database
// schema migrations
package dbschema

import (
	"embed"
)

// Migrations contain an embedded filesystem with all the sql migration files
//
//go:embed migrations/*.sql
var Migrations embed.FS
