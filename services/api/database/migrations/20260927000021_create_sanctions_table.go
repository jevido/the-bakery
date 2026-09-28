package migrations

// M20260927000021CreateSanctionsTable holds suspensions and bans of members
// and guilds. target_id is identity's or guilds' id, without a foreign key.
type M20260927000021CreateSanctionsTable struct{}

func (r *M20260927000021CreateSanctionsTable) Signature() string {
	return "20260927000021_create_sanctions_table"
}

func (r *M20260927000021CreateSanctionsTable) Up() error {
	return exec(
		`CREATE TABLE sanctions (
			id bigserial PRIMARY KEY,
			target_kind varchar(10) NOT NULL CHECK (target_kind IN ('member', 'guild')),
			target_id bigint NOT NULL,
			kind varchar(12) NOT NULL CHECK (kind IN ('suspension', 'ban')),
			reason text NOT NULL,
			until timestamptz,
			by_operator_id bigint NOT NULL REFERENCES operators (id),
			lifted_at timestamptz,
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now(),
			CHECK ((kind = 'suspension') = (until IS NOT NULL))
		)`,
		`CREATE UNIQUE INDEX sanctions_one_per_target ON sanctions (target_kind, target_id) WHERE lifted_at IS NULL`,
	)
}

func (r *M20260927000021CreateSanctionsTable) Down() error {
	return exec(`DROP TABLE sanctions`)
}
