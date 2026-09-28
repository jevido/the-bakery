// Package agents wires the agents context: its routes, the functions it
// publishes, and the dev seeder's agent. It takes the signed-in member and
// display names from identity, and membership answers from guilds.
package agents

import (
	"context"
	"errors"

	"github.com/goravel/framework/contracts/route"

	"github.com/jevido/the-bakery/services/api/contexts/agents/app"
	"github.com/jevido/the-bakery/services/api/contexts/agents/domain"
	agentshttp "github.com/jevido/the-bakery/services/api/contexts/agents/http"
	"github.com/jevido/the-bakery/services/api/contexts/agents/infra"
	"github.com/jevido/the-bakery/services/api/contexts/guilds"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

// memberNames adapts identity's display names to agents' MemberNames.
type memberNames struct{}

func (memberNames) DisplayNames(ctx context.Context, ids []uint64) (map[uint64]string, error) {
	return identity.DisplayNames(ctx, ids)
}

var service = app.NewService(infra.Agents{}, infra.Shares{}, guilds.NewMemberships(), memberNames{},
	app.NewDispatcher(infra.LogEvents{}.Handle))

// Routes registers the agents routes, all behind identity.RequireMember.
func Routes(r route.Router) {
	c := agentshttp.NewController(service, identity.MemberID)
	r.Middleware(identity.RequireMember).Group(func(r route.Router) {
		r.Get("/api/agent-traits", c.ListTraits)
		r.Get("/api/agents", c.ListAgents)
		r.Post("/api/agents", c.CreateAgent)
		r.Post("/api/agents/recruit", c.RecruitAgent)
		r.Get("/api/guilds/{guild}/agents", c.ListGuildAgents)
		r.Put("/api/agents/{agent}/shares/{guild}", c.ShareAgent)
		r.Delete("/api/agents/{agent}/shares/{guild}", c.UnshareAgent)
		r.Get("/api/agents/{agent}/origin", c.Origin)
		r.Post("/api/agents/{agent}/pull-origin", c.PullOrigin)
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

// Agent is an agent as other modules (the MCP server) see it: no file
// contents, its skills by name and description.
type Agent struct {
	ID             uint64
	Slug           string
	Name           string
	Title          string
	Backstory      string
	Traits         []string
	Model          string
	PermissionMode string
	AllowedTools   []string
	WorkPriorities map[string]int
	Skills         []Skill
	Revision       int
	OwnerName      string
	Recruited      bool
}

type Skill struct {
	Name        string
	Description string
}

func agentOf(a domain.Agent, ownerName string) Agent {
	out := Agent{
		ID: a.ID, Slug: a.Slug, Name: a.Name, Title: a.Title, Backstory: a.Backstory, Traits: a.Traits, Model: a.Model,
		PermissionMode: a.PermissionMode, AllowedTools: a.AllowedTools, WorkPriorities: a.WorkPriorities,
		Revision: a.Revision, OwnerName: ownerName, Recruited: a.OriginAgentID != nil,
	}
	for _, s := range a.Skills() {
		out.Skills = append(out.Skills, Skill{Name: s.Name, Description: s.Description})
	}
	return out
}

// ListAgents returns the member's own agents.
func ListAgents(ctx context.Context, memberID uint64) ([]Agent, error) {
	as, err := service.ListAgents(ctx, memberID, nil)
	if err != nil {
		return nil, err
	}
	out := make([]Agent, 0, len(as))
	for _, a := range as {
		full, err := service.GetAgent(ctx, a.ID, memberID)
		if err != nil {
			return nil, err
		}
		out = append(out, agentOf(full, ""))
	}
	return out, nil
}

// ListGuildAgents returns the agents shared with a guild the member is in.
func ListGuildAgents(ctx context.Context, guildID, memberID uint64) ([]Agent, error) {
	as, err := service.ListGuildAgents(ctx, guildID, memberID)
	if err != nil {
		return nil, err
	}
	out := make([]Agent, len(as))
	for i, a := range as {
		out[i] = agentOf(a.Agent, a.OwnerName)
	}
	return out, nil
}

// GetAgent returns one of the member's agents, or one shared with a guild
// they are in.
func GetAgent(ctx context.Context, id, memberID uint64) (Agent, error) {
	a, err := service.VisibleAgent(ctx, id, memberID)
	if err != nil {
		return Agent{}, err
	}
	return agentOf(a.Agent, a.OwnerName), nil
}

// Owns reports whether the agent is one of the member's own, not deleted.
// Boards asks it before a member's agent claims a task.
func Owns(ctx context.Context, memberID, agentID uint64) (bool, error) {
	_, err := service.GetAgent(ctx, agentID, memberID)
	if errors.Is(err, app.ErrAgentNotFound) {
		return false, nil
	}
	return err == nil, err
}
