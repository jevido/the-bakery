// Package boards wires the boards context: its routes, and the dev seeder's
// board. It takes membership answers from guilds.Memberships and nothing
// else from guilds.
package boards

import (
	"context"

	"github.com/goravel/framework/contracts/route"

	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
	boardshttp "github.com/jevido/the-bakery/services/api/contexts/boards/http"
	"github.com/jevido/the-bakery/services/api/contexts/boards/infra"
	"github.com/jevido/the-bakery/services/api/contexts/guilds"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

var service = app.NewService(guilds.NewMemberships(), infra.Boards{}, infra.Tasks{}, infra.LogEvents{})

// Routes registers the board and task routes, all behind
// identity.RequireMember.
func Routes(r route.Router) {
	c := boardshttp.NewController(service, identity.MemberID)
	r.Middleware(identity.RequireMember).Group(func(r route.Router) {
		r.Get("/api/guilds/{guild}/boards", c.ListBoards)
		r.Post("/api/guilds/{guild}/boards", c.CreateBoard)
		r.Get("/api/boards/{board}", c.GetBoard)
		r.Post("/api/boards/{board}/tasks", c.CreateTask)
		r.Patch("/api/tasks/{task}", c.UpdateTask)
		r.Post("/api/tasks/{task}/move", c.MoveTask)
		r.Delete("/api/tasks/{task}", c.DeleteTask)
	})
}

// SeedTask is a task for SeedBoard.
type SeedTask struct {
	Title  string
	Column string
}

// SeedBoard makes sure the guild has a board with this name; a new board
// gets the given tasks, an existing one is left as it is. For the dev
// seeder only, which acts as memberID.
func SeedBoard(ctx context.Context, guildID, memberID uint64, name string, tasks []SeedTask) error {
	if _, found, err := (infra.Boards{}).ByName(ctx, guildID, name); err != nil || found {
		return err
	}
	b, err := service.CreateBoard(ctx, guildID, memberID, name)
	if err != nil {
		return err
	}
	for _, st := range tasks {
		t, err := service.CreateTask(ctx, b.ID, memberID, st.Title, "")
		if err != nil {
			return err
		}
		if st.Column != string(domain.Backlog) {
			if _, err := service.MoveTask(ctx, t.ID, memberID, st.Column, nil, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
