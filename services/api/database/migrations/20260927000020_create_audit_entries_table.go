package migrations

// M20260927000020CreateAuditEntriesTable is the platform's audit log:
// append-only, nothing in the code updates or deletes a row.
type M20260927000020CreateAuditEntriesTable struct{}

func (r *M20260927000020CreateAuditEntriesTable) Signature() string {
	return "20260927000020_create_audit_entries_table"
}

func (r *M20260927000020CreateAuditEntriesTable) Up() error {
	return exec(
		`CREATE TABLE audit_entries (
			id bigserial PRIMARY KEY,
			actor_kind varchar(10) NOT NULL CHECK (actor_kind IN ('member', 'operator')),
			actor_id bigint NOT NULL,
			action varchar(60) NOT NULL,
			target_kind varchar(20) NOT NULL DEFAULT '',
			target_id bigint NOT NULL DEFAULT 0,
			reason text NOT NULL DEFAULT '',
			meta jsonb NOT NULL DEFAULT '{}',
			ip varchar(64) NOT NULL DEFAULT '',
			at timestamptz NOT NULL
		)`,
		`CREATE INDEX audit_entries_actor_index ON audit_entries (actor_kind, actor_id, id DESC)`,
		`CREATE INDEX audit_entries_target_index ON audit_entries (target_kind, target_id, id DESC)`,
		`CREATE INDEX audit_entries_action_index ON audit_entries (action, id DESC)`,
	)
}

func (r *M20260927000020CreateAuditEntriesTable) Down() error {
	return exec(`DROP TABLE audit_entries`)
}
