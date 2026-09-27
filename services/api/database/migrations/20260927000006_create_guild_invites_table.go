package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

// M20260927000006CreateGuildInvitesTable stores the guilds context's invites.
type M20260927000006CreateGuildInvitesTable struct{}

func (r *M20260927000006CreateGuildInvitesTable) Signature() string {
	return "20260927000006_create_guild_invites_table"
}

func (r *M20260927000006CreateGuildInvitesTable) Up() error {
	return facades.Schema().Create("guild_invites", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("guild_id")
		table.String("code", 32)
		table.UnsignedBigInteger("created_by")
		table.TimestampTz("expires_at").Nullable()
		table.Integer("max_uses").Nullable()
		table.Integer("uses").Default(0)
		table.TimestampTz("revoked_at").Nullable()
		table.TimestampsTz()
		table.Unique("code")
		table.Index("guild_id")
		table.Foreign("guild_id").References("id").On("guilds").CascadeOnDelete()
	})
}

func (r *M20260927000006CreateGuildInvitesTable) Down() error {
	return facades.Schema().DropIfExists("guild_invites")
}
