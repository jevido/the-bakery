package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

// M20260927000007CreatePersonalTokensTable stores identity's personal
// tokens, by hash only.
type M20260927000007CreatePersonalTokensTable struct{}

func (r *M20260927000007CreatePersonalTokensTable) Signature() string {
	return "20260927000007_create_personal_tokens_table"
}

func (r *M20260927000007CreatePersonalTokensTable) Up() error {
	return facades.Schema().Create("personal_tokens", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("member_id")
		table.String("name", 60)
		table.String("token_hash", 64)
		table.TimestampTz("last_used_at").Nullable()
		table.TimestampTz("revoked_at").Nullable()
		table.TimestampsTz()
		table.Unique("token_hash")
		table.Index("member_id")
	})
}

func (r *M20260927000007CreatePersonalTokensTable) Down() error {
	return facades.Schema().DropIfExists("personal_tokens")
}
