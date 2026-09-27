package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

// M20260927000004ArchiveGuilds lets a guild be archived.
type M20260927000004ArchiveGuilds struct{}

func (r *M20260927000004ArchiveGuilds) Signature() string {
	return "20260927000004_archive_guilds"
}

func (r *M20260927000004ArchiveGuilds) Up() error {
	return facades.Schema().Table("guilds", func(table schema.Blueprint) {
		table.TimestampTz("archived_at").Nullable()
	})
}

func (r *M20260927000004ArchiveGuilds) Down() error {
	return facades.Schema().Table("guilds", func(table schema.Blueprint) {
		table.DropColumn("archived_at")
	})
}
