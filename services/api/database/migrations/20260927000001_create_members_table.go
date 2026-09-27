package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

// M20260927000001CreateMembersTable stores identity's members.
type M20260927000001CreateMembersTable struct{}

func (r *M20260927000001CreateMembersTable) Signature() string {
	return "20260927000001_create_members_table"
}

func (r *M20260927000001CreateMembersTable) Up() error {
	return facades.Schema().Create("members", func(table schema.Blueprint) {
		table.ID()
		table.String("email")
		table.String("display_name", 60)
		table.String("password_hash")
		table.TimestampsTz()
		table.Unique("email")
	})
}

func (r *M20260927000001CreateMembersTable) Down() error {
	return facades.Schema().DropIfExists("members")
}
