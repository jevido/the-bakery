package migrations

// M20260927000017CreateTaskClaimsTable records which agent holds which task.
// One unreleased claim per task; an expired one is released by the next
// claim in the same transaction. agent_id is the agents context's id.
type M20260927000017CreateTaskClaimsTable struct{}

func (r *M20260927000017CreateTaskClaimsTable) Signature() string {
	return "20260927000017_create_task_claims_table"
}

func (r *M20260927000017CreateTaskClaimsTable) Up() error {
	return exec(
		`CREATE TABLE task_claims (
			id bigserial PRIMARY KEY,
			task_id bigint NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
			agent_id bigint NOT NULL,
			member_id bigint NOT NULL,
			machine_id varchar(100) NOT NULL,
			claimed_at timestamptz NOT NULL,
			expires_at timestamptz NOT NULL,
			released_at timestamptz
		)`,
		`CREATE UNIQUE INDEX task_claims_one_per_task ON task_claims (task_id) WHERE released_at IS NULL`,
	)
}

func (r *M20260927000017CreateTaskClaimsTable) Down() error {
	return exec(`DROP TABLE task_claims`)
}
