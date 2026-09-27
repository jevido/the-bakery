// Package seeders fills the dev database with a small colony to try things
// on. Running it twice changes nothing.
package seeders

import (
	"context"
	"fmt"

	"github.com/jevido/the-bakery/services/api/contexts/boards"
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
	guildID, err := guilds.SeedGuild(ctx, "First Colony", ids...)
	if err != nil {
		return fmt.Errorf("seeding First Colony: %w", err)
	}
	err = boards.SeedBoard(ctx, guildID, ids[0], "Getting settled", []boards.SeedTask{
		{Title: "Build a research bench", Column: "backlog", WorkType: "coding"},
		{Title: "Hunt the boomalope before it explodes", Column: "backlog", WorkType: "research"},
		{Title: "Plant rice by the river", Column: "todo", WorkType: "writing"},
		{Title: "Wall in the freezer", Column: "doing", WorkType: "testing"},
		{Title: "Bury the raider in the graveyard", Column: "done", WorkType: "coding"},
	})
	if err != nil {
		return fmt.Errorf("seeding board Getting settled: %w", err)
	}
	return nil
}
