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
		&migrations.M20260927000003CreateBoardsTables{},
		&migrations.M20260927000004ArchiveGuilds{},
		&migrations.M20260927000005CreateWebHandoffsTable{},
		&migrations.M20260927000006CreateGuildInvitesTable{},
		&migrations.M20260927000007CreatePersonalTokensTable{},
		&migrations.M20260927000008AddSubtasksToTasks{},
		&migrations.M20260927000009CreateTaskCommentsTable{},
		&migrations.M20260927000010CreateTaskActivityTable{},
		&migrations.M20260927000011CreateBoardColumnsTable{},
		&migrations.M20260927000012CreateBoardPresenceTable{},
	}
}
