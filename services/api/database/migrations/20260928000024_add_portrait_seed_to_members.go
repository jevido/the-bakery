package migrations

// M20260928000024AddPortraitSeedToMembers gives every member a portrait
// seed; existing members get a random one.
type M20260928000024AddPortraitSeedToMembers struct{}

func (r *M20260928000024AddPortraitSeedToMembers) Signature() string {
	return "20260928000024_add_portrait_seed_to_members"
}

func (r *M20260928000024AddPortraitSeedToMembers) Up() error {
	return exec(
		`ALTER TABLE members ADD COLUMN portrait_seed varchar(40) NOT NULL DEFAULT ''`,
		`UPDATE members SET portrait_seed = substr(md5(random()::text || id::text), 1, 16) WHERE portrait_seed = ''`,
	)
}

func (r *M20260928000024AddPortraitSeedToMembers) Down() error {
	return exec(`ALTER TABLE members DROP COLUMN portrait_seed`)
}
