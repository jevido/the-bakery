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

var service = app.NewService(infra.Members{}, infra.Hasher{}, infra.Handoffs{}, infra.Tokens{}, infra.Secrets{})

// RequireMember is the middleware that refuses requests without a valid
// member token (401).
var RequireMember contractshttp.Middleware = identityhttp.RequireMember{}

func init() {
	identityhttp.SetPersonalTokenVerifier(service.VerifyPersonalToken)
}

// Routes registers register, login and me.
func Routes(r route.Router) {
	c := identityhttp.NewController(service)
	r.Post("/api/register", c.Register)
	r.Post("/api/login", c.Login)
	r.Middleware(RequireMember).Get("/api/me", c.Me)
	r.Post("/api/web/register", c.WebRegister)
	r.Post("/api/web/login", c.WebLogin)
	r.Post("/api/web/logout", c.WebLogout)
	r.Middleware(RequireMember).Post("/api/web/handoff", c.CreateHandoff)
	r.Middleware(RequireMember).Group(func(r route.Router) {
		r.Get("/api/tokens", c.ListTokens)
		r.Post("/api/tokens", c.CreateToken)
		r.Delete("/api/tokens/{token}", c.RevokeToken)
	})
	r.Post("/api/web/handoff/redeem", c.RedeemHandoff)
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

// VerifyPersonalToken returns the member a personal token (`bky_…`) belongs
// to; the MCP server signs its callers in with it.
func VerifyPersonalToken(ctx context.Context, token string) (uint64, error) {
	id, err := service.VerifyPersonalToken(ctx, token)
	if err != nil {
		return 0, err
	}
	if memberCheck != nil {
		if err := memberCheck(ctx, id); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// memberCheck is the moderation context's answer to "may this member use
// The Bakery?": nil, or a *refusal.Refusal.
var memberCheck func(ctx context.Context, memberID uint64) error

// SetSanctionCheck sets the member check that RequireMember, every sign-in
// and VerifyPersonalToken (the MCP server's sign-in) ask.
func SetSanctionCheck(f func(ctx context.Context, memberID uint64) error) {
	memberCheck = f
	identityhttp.SetSanctionCheck(f)
}

// MemberExists reports whether a member with this id exists.
func MemberExists(ctx context.Context, id uint64) (bool, error) {
	names, err := service.DisplayNames(ctx, []uint64{id})
	if err != nil {
		return false, err
	}
	_, ok := names[id]
	return ok, nil
}

// DisplayNames maps member ids to display names; unknown ids are left out.
func DisplayNames(ctx context.Context, ids []uint64) (map[uint64]string, error) {
	return service.DisplayNames(ctx, ids)
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

// SetAuditRecorder sets where this context's sensitive actions are
// recorded: the moderation context's audit log, set when it is wired.
func SetAuditRecorder(record func(ctx context.Context, actorMemberID uint64, action, targetKind string, targetID uint64, meta map[string]any)) {
	service.SetAuditLog(auditRecorder(record))
}

type auditRecorder func(ctx context.Context, actorMemberID uint64, action, targetKind string, targetID uint64, meta map[string]any)

func (f auditRecorder) Record(ctx context.Context, e app.AuditRecord) {
	f(ctx, e.ActorID, e.Action, e.TargetKind, e.TargetID, e.Meta)
}
