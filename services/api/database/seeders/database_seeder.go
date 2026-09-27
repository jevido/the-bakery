// Package seeders fills the dev database with a small colony to try things
// on. Running it twice changes nothing.
package seeders

import (
	"context"
	"fmt"

	"github.com/jevido/the-bakery/services/api/contexts/guilds"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

type DatabaseSeeder struct{}

func (DatabaseSeeder) Signature() string { return "DatabaseSeeder" }

func (DatabaseSeeder) Run() error {
	ctx := context.Background()
	var ids []uint64
	for _, m := range []struct{ email, name string }{
		{"ada@bakery.test", "Ada"},
		{"bram@bakery.test", "Bram"},
		{"cas@bakery.test", "Cas"},
	} {
		id, err := identity.SeedMember(ctx, m.email, m.name, "password")
		if err != nil {
			return fmt.Errorf("seeding member %s: %w", m.email, err)
		}
		ids = append(ids, id)
	}
	if _, err := guilds.SeedGuild(ctx, "First Colony", ids...); err != nil {
		return fmt.Errorf("seeding First Colony: %w", err)
	}
	return nil
}
