// Package moderation wires the moderation context: the console's routes
// (/api/console/*, operators only) and the artisan commands that make
// operators. It keeps bad actors out; see docs/domain/contexts/moderation.
package moderation

import (
	"context"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/http/limit"
	"github.com/goravel/framework/http/middleware"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/app/refusal"
	"github.com/jevido/the-bakery/services/api/contexts/boards"
	"github.com/jevido/the-bakery/services/api/contexts/guilds"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
	moderationhttp "github.com/jevido/the-bakery/services/api/contexts/moderation/http"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/infra"
)

var service = app.NewService(app.Deps{
	Operators: infra.Operators{}, Hasher: infra.Hasher{}, Secrets: infra.Secrets{}, TOTP: infra.TOTP{},
	Audit: infra.AuditEntries{}, Sanctions: infra.Sanctions{}, Reports: infra.Reports{}, Targets: targets{}, Effects: effects{},
})

// targets asks identity and guilds whether a target exists.
type targets struct{}

func (targets) MemberExists(ctx context.Context, id uint64) (bool, error) {
	return identity.MemberExists(ctx, id)
}
func (targets) GuildExists(ctx context.Context, id uint64) (bool, error) {
	return guilds.GuildExists(ctx, id)
}

// effects ends a sanctioned target's live board streams.
type effects struct{}

func (effects) Sanctioned(ctx context.Context, targetKind string, targetID uint64) error {
	if targetKind == domain.TargetMember {
		return boards.CloseStreamsOf(ctx, targetID, 0)
	}
	return boards.CloseStreamsOf(ctx, 0, targetID)
}

// refusalFor turns an active sanction into the refusal identity and guilds
// answer with.
func refusalFor(ctx context.Context, targetKind string, targetID uint64) error {
	s, err := service.ActiveFor(ctx, targetKind, targetID)
	if err != nil || s == nil {
		return err
	}
	r := &refusal.Refusal{Until: s.Until}
	switch {
	case targetKind == domain.TargetMember && s.Kind == domain.Ban:
		r.Code, r.Message = refusal.AccountBanned, "This account is banned from The Bakery. Reason: "+s.Reason
	case targetKind == domain.TargetMember:
		r.Code, r.Message = refusal.AccountSuspended, "This account is suspended until "+s.Until.UTC().Format("2 January 2006 15:04 UTC")+". Reason: "+s.Reason
	case s.Kind == domain.Ban:
		r.Code, r.Message = refusal.GuildBanned, "This guild is banned from The Bakery. Reason: "+s.Reason
	default:
		r.Code, r.Message = refusal.GuildSuspended, "This guild is suspended until "+s.Until.UTC().Format("2 January 2006 15:04 UTC")+". Reason: "+s.Reason
	}
	return r
}

// RememberIP puts the client's IP in each request's context for the audit
// log. The router runs it on every API route.
var RememberIP contractshttp.Middleware = moderationhttp.RememberIP{}

func init() {
	// The contexts that record sensitive actions do it through this.
	identity.SetAuditRecorder(recordMember)
	guilds.SetAuditRecorder(recordMember)
	// And a sanction stops its target at every door.
	identity.SetSanctionCheck(func(ctx context.Context, memberID uint64) error {
		return refusalFor(ctx, domain.TargetMember, memberID)
	})
	guilds.SetSanctionCheck(func(ctx context.Context, guildID uint64) error {
		return refusalFor(ctx, domain.TargetGuild, guildID)
	})
}

// recordMember writes a member's sensitive action to the audit log.
func recordMember(ctx context.Context, actorMemberID uint64, action, targetKind string, targetID uint64, meta map[string]any) {
	err := service.Record(ctx, domain.AuditEntry{ActorKind: domain.ActorMember, ActorID: actorMemberID, Action: action, TargetKind: targetKind, TargetID: targetID, Meta: meta})
	if err != nil {
		facades.Log().WithContext(ctx).Errorf("audit %s: %v", action, err)
	}
}

// recordOperator writes an operator's action to the audit log.
func recordOperator(ctx context.Context, operatorID uint64, action, targetKind string, targetID uint64, reason string, meta map[string]any) {
	err := service.Record(ctx, domain.AuditEntry{ActorKind: domain.ActorOperator, ActorID: operatorID, Action: action, TargetKind: targetKind, TargetID: targetID, Reason: reason, Meta: meta})
	if err != nil {
		facades.Log().WithContext(ctx).Errorf("audit %s: %v", action, err)
	}
}

// RequireOperator is the console's middleware.
var RequireOperator contractshttp.Middleware = moderationhttp.RequireOperator{}

// Routes registers the console's routes. Signing in is limited to five
// tries a minute per client, each step.
func Routes(r route.Router) {
	facades.RateLimiter().For("console-sign-in", func(ctx contractshttp.Context) contractshttp.Limit {
		return limit.PerMinute(5)
	})
	c := moderationhttp.NewController(service, func(ctx contractshttp.Context, operatorID uint64) {
		recordOperator(ctx.Context(), operatorID, "operator.signed_in", domain.TargetOperator, operatorID, "", nil)
	})
	r.Middleware(middleware.Throttle("console-sign-in")).Group(func(r route.Router) {
		r.Post("/api/console/login", c.Login)
		r.Post("/api/console/totp/verify", c.VerifyTOTP)
	})
	r.Middleware(RequireOperator).Group(func(r route.Router) {
		r.Get("/api/console/me", c.Me)
		r.Get("/api/console/audit", c.Audit)
		r.Get("/api/console/sanctions", c.ListSanctions)
		r.Post("/api/console/sanctions", c.Sanction)
		r.Post("/api/console/sanctions/{sanction}/lift", c.LiftSanction)
		r.Get("/api/console/reports", c.ListReports)
		r.Post("/api/console/reports/{report}/dismiss", c.DismissReport)
		r.Post("/api/console/reports/{report}/action", c.ActionReport)
	})
	// Members report members and guilds.
	mc := moderationhttp.NewMemberController(service, identity.MemberID)
	r.Middleware(identity.RequireMember).Group(func(r route.Router) {
		r.Post("/api/reports", mc.FileReport)
	})
}
