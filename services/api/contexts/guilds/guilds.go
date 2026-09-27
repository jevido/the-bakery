// Package guilds is what other contexts and the router may use from the
// guilds context: its routes and the Memberships question. Boards import
// Memberships and nothing else.
package guilds

import (
	"context"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/http/limit"
	"github.com/goravel/framework/http/middleware"

	"github.com/jevido/the-bakery/services/api/app/facades"

	"github.com/jevido/the-bakery/services/api/contexts/guilds/app"
	"github.com/jevido/the-bakery/services/api/contexts/guilds/domain"
	guildshttp "github.com/jevido/the-bakery/services/api/contexts/guilds/http"
	"github.com/jevido/the-bakery/services/api/contexts/guilds/infra"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

// Memberships answers whether a member belongs to a guild, and whether a
// guild is archived (read-only for everyone).
type Memberships interface {
	IsMember(ctx context.Context, guildID, memberID uint64) (bool, error)
	IsArchived(ctx context.Context, guildID uint64) (bool, error)
}

func NewMemberships() Memberships { return infra.Memberships{} }

// memberLookup adapts identity's email lookup to guilds' MemberLookup.
type memberLookup struct{}

func (memberLookup) MemberIDByEmail(ctx context.Context, email string) (uint64, bool, error) {
	return identity.MemberIDByEmail(ctx, email)
}

func (memberLookup) DisplayNames(ctx context.Context, ids []uint64) (map[uint64]string, error) {
	return identity.DisplayNames(ctx, ids)
}

var service = app.NewService(infra.Guilds{}, infra.Invites{}, infra.Codes{}, memberLookup{}, infra.LogEvents{})

// Routes registers the guild routes, all behind identity.RequireMember.
func Routes(r route.Router) {
	c := guildshttp.NewController(service, identity.MemberID)
	r.Middleware(identity.RequireMember).Group(func(r route.Router) {
		r.Post("/api/guilds", c.Found)
		r.Get("/api/guilds", c.Mine)
		r.Get("/api/guilds/{guild}", c.Show)
		r.Patch("/api/guilds/{guild}", c.Rename)
		r.Post("/api/guilds/{guild}/archive", c.Archive)
		r.Post("/api/guilds/{guild}/restore", c.Restore)
		r.Get("/api/guilds/{guild}/members", c.Members)
		r.Post("/api/guilds/{guild}/members", c.AddMember)
		r.Delete("/api/guilds/{guild}/members/{member}", c.RemoveMember)
		r.Post("/api/guilds/{guild}/leave", c.Leave)
		r.Get("/api/guilds/{guild}/invites", c.ListInvites)
		r.Post("/api/guilds/{guild}/invites", c.CreateInvite)
		r.Delete("/api/guilds/{guild}/invites/{invite}", c.RevokeInvite)
	})

	// Invite codes can be looked up without signing in, so both code routes
	// are rate-limited per client against guessing.
	facades.RateLimiter().For("invite-codes", func(ctx contractshttp.Context) contractshttp.Limit {
		return limit.PerMinute(30)
	})
	r.Middleware(middleware.Throttle("invite-codes")).Get("/api/invites/{code}", c.ShowInvite)
	r.Middleware(middleware.Throttle("invite-codes"), identity.RequireMember).Post("/api/invites/{code}/accept", c.AcceptInvite)
}

// SeedGuild makes sure a guild with this name exists (founded by the first
// member) and that every given member is in it. For the dev seeder only.
func SeedGuild(ctx context.Context, name string, memberIDs ...uint64) (uint64, error) {
	g, found, err := infra.Guilds{}.ByName(ctx, name)
	if err != nil {
		return 0, err
	}
	if !found {
		if g, err = service.FoundGuild(ctx, name, memberIDs[0]); err != nil {
			return 0, err
		}
	}
	for _, id := range memberIDs {
		if g.HasMember(id) {
			continue
		}
		ev, err := g.AddMember(id)
		if err != nil {
			return 0, err
		}
		if err := (infra.Guilds{}).AddMembership(ctx, ev); err != nil && err != domain.ErrAlreadyMember {
			return 0, err
		}
	}
	return g.ID, nil
}

// GuildSummary is a guild as other modules (the MCP server) see it.
type GuildSummary struct {
	ID       uint64
	Name     string
	Archived bool
}

// ListGuildsOf returns the member's guilds, archived ones only when asked.
func ListGuildsOf(ctx context.Context, memberID uint64, includeArchived bool) ([]GuildSummary, error) {
	gs, err := service.ListGuildsOf(ctx, memberID, includeArchived)
	if err != nil {
		return nil, err
	}
	out := make([]GuildSummary, len(gs))
	for i, g := range gs {
		out[i] = GuildSummary{ID: g.ID, Name: g.Name, Archived: g.Archived}
	}
	return out, nil
}
