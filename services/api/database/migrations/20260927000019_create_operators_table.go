package migrations

// M20260927000019CreateOperatorsTable holds the console's operators, apart
// from members. totp_secret is encrypted with the app key.
type M20260927000019CreateOperatorsTable struct{}

func (r *M20260927000019CreateOperatorsTable) Signature() string {
	return "20260927000019_create_operators_table"
}

func (r *M20260927000019CreateOperatorsTable) Up() error {
	return exec(
		`CREATE TABLE operators (
			id bigserial PRIMARY KEY,
			email varchar(254) NOT NULL UNIQUE,
			password_hash varchar(255) NOT NULL,
			totp_secret text NOT NULL,
			totp_confirmed_at timestamptz,
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now()
		)`,
	)
}

func (r *M20260927000019CreateOperatorsTable) Down() error {
	return exec(`DROP TABLE operators`)
}
