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
		&migrations.M20260927000013CreateWorkTypesTable{},
		&migrations.M20260927000014CreateAgentsTables{},
		&migrations.M20260927000015CreateAgentSharesTable{},
		&migrations.M20260927000016CreateTaskRunsTable{},
		&migrations.M20260927000017CreateTaskClaimsTable{},
		&migrations.M20260927000018AddDraftingAndRunKind{},
		&migrations.M20260927000019CreateOperatorsTable{},
		&migrations.M20260927000020CreateAuditEntriesTable{},
		&migrations.M20260927000021CreateSanctionsTable{},
		&migrations.M20260927000022CreateReportsTable{},
		&migrations.M20260927000023CreateSignalsTable{},
		&migrations.M20260928000024AddPortraitSeedToMembers{},
	}
}
