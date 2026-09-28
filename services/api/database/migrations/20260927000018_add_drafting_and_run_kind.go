package migrations

// M20260927000018AddDraftingAndRunKind lets a task be prioritized for one
// agent or forbidden for all, and tells work runs from plan runs.
type M20260927000018AddDraftingAndRunKind struct{}

func (r *M20260927000018AddDraftingAndRunKind) Signature() string {
	return "20260927000018_add_drafting_and_run_kind"
}

func (r *M20260927000018AddDraftingAndRunKind) Up() error {
	return exec(
		`ALTER TABLE tasks ADD COLUMN prioritized_agent_id bigint, ADD COLUMN forbidden boolean NOT NULL DEFAULT false`,
		`ALTER TABLE task_runs ADD COLUMN kind varchar(10) NOT NULL DEFAULT 'work' CHECK (kind IN ('work', 'plan'))`,
	)
}

func (r *M20260927000018AddDraftingAndRunKind) Down() error {
	return exec(
		`ALTER TABLE task_runs DROP COLUMN kind`,
		`ALTER TABLE tasks DROP COLUMN prioritized_agent_id, DROP COLUMN forbidden`,
	)
}
