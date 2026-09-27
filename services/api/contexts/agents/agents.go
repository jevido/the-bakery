// Package agents wires the agents context: its routes, the functions it
// publishes, and the dev seeder's agent. It takes the signed-in member from
// identity and nothing else from other contexts yet.
package agents

import (
	"context"

	"github.com/goravel/framework/contracts/route"

	"github.com/jevido/the-bakery/services/api/contexts/agents/app"
	"github.com/jevido/the-bakery/services/api/contexts/agents/domain"
	agentshttp "github.com/jevido/the-bakery/services/api/contexts/agents/http"
	"github.com/jevido/the-bakery/services/api/contexts/agents/infra"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

var service = app.NewService(infra.Agents{}, app.NewDispatcher(infra.LogEvents{}.Handle))

// Routes registers the agents routes, all behind identity.RequireMember.
func Routes(r route.Router) {
	c := agentshttp.NewController(service, identity.MemberID)
	r.Middleware(identity.RequireMember).Group(func(r route.Router) {
		r.Get("/api/agent-traits", c.ListTraits)
		r.Get("/api/agents", c.ListAgents)
		r.Post("/api/agents", c.CreateAgent)
		r.Get("/api/agents/{agent}", c.GetAgent)
		r.Put("/api/agents/{agent}", c.ReviseAgent)
		r.Delete("/api/agents/{agent}", c.DeleteAgent)
	})
}

// SeedSkill is one file of a SeedAgent's skillset.
type SeedSkill struct {
	Path    string
	Content string
}

// SeedAgent makes sure the member has an agent with this slug, for the dev
// seeder only.
func SeedAgent(ctx context.Context, ownerID uint64, slug, name, title string, traits []string, model string, priorities map[string]int, files []SeedSkill) error {
	agents, err := service.ListAgents(ctx, ownerID, nil)
	if err != nil {
		return err
	}
	for _, a := range agents {
		if a.Slug == slug {
			return nil
		}
	}
	p := domain.Profile{Name: name, Title: title, Traits: traits, Model: model, WorkPriorities: priorities}
	for _, f := range files {
		p.Files = append(p.Files, domain.File{Path: f.Path, Content: f.Content})
	}
	_, err = service.CreateAgent(ctx, ownerID, slug, p)
	return err
}
