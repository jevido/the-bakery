package bootstrap

import (
	"github.com/goravel/framework/contracts/database/schema"

	"github.com/jevido/the-bakery/services/api/database/migrations"
)

func Migrations() []schema.Migration {
	return []schema.Migration{
		&migrations.M20210101000001CreateJobsTable{},
		&migrations.M20260927000001CreateMembersTable{},
		&migrations.M20260927000002CreateGuildsTables{},
	}
}
