package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

// M20260927000002CreateGuildsTables stores the guilds context's guilds and
// memberships. member_id is identity's id; there is deliberately no foreign
// key across contexts.
type M20260927000002CreateGuildsTables struct{}

func (r *M20260927000002CreateGuildsTables) Signature() string {
	return "20260927000002_create_guilds_tables"
}

func (r *M20260927000002CreateGuildsTables) Up() error {
	if err := facades.Schema().Create("guilds", func(table schema.Blueprint) {
		table.ID()
		table.String("name", 60)
		table.TimestampsTz()
	}); err != nil {
		return err
	}
	return facades.Schema().Create("guild_memberships", func(table schema.Blueprint) {
		table.UnsignedBigInteger("guild_id")
		table.UnsignedBigInteger("member_id")
		table.TimestampTz("joined_at")
		table.Primary("guild_id", "member_id")
		table.Index("member_id")
		table.Foreign("guild_id").References("id").On("guilds")
	})
}

func (r *M20260927000002CreateGuildsTables) Down() error {
	if err := facades.Schema().DropIfExists("guild_memberships"); err != nil {
		return err
	}
	return facades.Schema().DropIfExists("guilds")
}
