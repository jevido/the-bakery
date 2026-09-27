// Package identity is what other contexts and the router may use from the
// identity context: its routes, the RequireMember middleware, the id of the
// signed-in member, and a lookup of a member id by email. Nothing else in
// contexts/identity is for outside use.
package identity

import (
	"context"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"

	"github.com/jevido/the-bakery/services/api/contexts/identity/app"
	identityhttp "github.com/jevido/the-bakery/services/api/contexts/identity/http"
	"github.com/jevido/the-bakery/services/api/contexts/identity/infra"
)

var service = app.NewService(infra.Members{}, infra.Hasher{})

// RequireMember is the middleware that refuses requests without a valid
// member token (401).
var RequireMember contractshttp.Middleware = identityhttp.RequireMember{}

// Routes registers register, login and me.
func Routes(r route.Router) {
	c := identityhttp.NewController(service)
	r.Post("/api/register", c.Register)
	r.Post("/api/login", c.Login)
	r.Middleware(RequireMember).Get("/api/me", c.Me)
}

// MemberID returns the id of the signed-in member. Only valid behind
// RequireMember.
func MemberID(ctx contractshttp.Context) (uint64, bool) {
	return identityhttp.MemberID(ctx)
}

// MemberIDByEmail resolves an email address to a member id.
func MemberIDByEmail(ctx context.Context, email string) (uint64, bool, error) {
	return service.MemberIDByEmail(ctx, email)
}

// SeedMember makes sure a member with this email exists and returns its id.
// An existing member is left as it is. For the dev seeder only.
func SeedMember(ctx context.Context, email, displayName, password string) (uint64, error) {
	if id, found, err := service.MemberIDByEmail(ctx, email); err != nil || found {
		return id, err
	}
	m, err := service.Register(ctx, email, displayName, password)
	return m.ID, err
}
