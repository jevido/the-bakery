package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

// M20260927000005CreateWebHandoffsTable stores identity's handoff codes, by
// hash only.
type M20260927000005CreateWebHandoffsTable struct{}

func (r *M20260927000005CreateWebHandoffsTable) Signature() string {
	return "20260927000005_create_web_handoffs_table"
}

func (r *M20260927000005CreateWebHandoffsTable) Up() error {
	return facades.Schema().Create("web_handoffs", func(table schema.Blueprint) {
		table.ID()
		table.String("code_hash", 64)
		table.UnsignedBigInteger("member_id")
		table.TimestampTz("expires_at")
		table.TimestampTz("used_at").Nullable()
		table.TimestampTz("created_at")
		table.Unique("code_hash")
		table.Index("expires_at")
	})
}

func (r *M20260927000005CreateWebHandoffsTable) Down() error {
	return facades.Schema().DropIfExists("web_handoffs")
}
