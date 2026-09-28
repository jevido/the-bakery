package migrations

// M20260927000022CreateReportsTable holds members' reports about members
// and guilds, and how operators handled them.
type M20260927000022CreateReportsTable struct{}

func (r *M20260927000022CreateReportsTable) Signature() string {
	return "20260927000022_create_reports_table"
}

func (r *M20260927000022CreateReportsTable) Up() error {
	return exec(
		`CREATE TABLE reports (
			id bigserial PRIMARY KEY,
			by_member_id bigint NOT NULL,
			target_kind varchar(10) NOT NULL CHECK (target_kind IN ('member', 'guild')),
			target_id bigint NOT NULL,
			reason text NOT NULL,
			status varchar(10) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'dismissed', 'actioned')),
			handled_by bigint REFERENCES operators (id),
			handled_at timestamptz,
			sanction_id bigint REFERENCES sanctions (id),
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX reports_status_index ON reports (status, id)`,
		`CREATE INDEX reports_member_index ON reports (by_member_id, created_at)`,
	)
}

func (r *M20260927000022CreateReportsTable) Down() error {
	return exec(`DROP TABLE reports`)
}
