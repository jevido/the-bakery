// Package guilds is what other contexts and the router may use from the
// guilds context: its routes and the Memberships question. Boards import
// Memberships and nothing else.
package guilds

import (
	"context"

	"github.com/goravel/framework/contracts/route"

	"github.com/jevido/the-bakery/services/api/contexts/guilds/app"
	"github.com/jevido/the-bakery/services/api/contexts/guilds/domain"
	guildshttp "github.com/jevido/the-bakery/services/api/contexts/guilds/http"
	"github.com/jevido/the-bakery/services/api/contexts/guilds/infra"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

// Memberships answers whether a member belongs to a guild.
type Memberships interface {
	IsMember(ctx context.Context, guildID, memberID uint64) (bool, error)
}

func NewMemberships() Memberships { return infra.Memberships{} }

// memberLookup adapts identity's email lookup to guilds' MemberLookup.
type memberLookup struct{}

func (memberLookup) MemberIDByEmail(ctx context.Context, email string) (uint64, bool, error) {
	return identity.MemberIDByEmail(ctx, email)
}

var service = app.NewService(infra.Guilds{}, memberLookup{}, infra.LogEvents{})

// Routes registers the guild routes, all behind identity.RequireMember.
func Routes(r route.Router) {
	c := guildshttp.NewController(service, identity.MemberID)
	r.Middleware(identity.RequireMember).Group(func(r route.Router) {
		r.Post("/api/guilds", c.Found)
		r.Get("/api/guilds", c.Mine)
		r.Post("/api/guilds/{guild}/members", c.AddMember)
	})
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
