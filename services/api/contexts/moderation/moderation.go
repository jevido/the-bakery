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
	"github.com/jevido/the-bakery/services/api/app/limits"
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
	Audit: infra.AuditEntries{}, Sanctions: infra.Sanctions{}, Reports: infra.Reports{}, Targets: targets{}, Effects: effects{}, Directory: directory{}, Signals: infra.Signals{},
})

// targets asks identity and guilds whether a target exists.
type targets struct{}

func (targets) MemberExists(ctx context.Context, id uint64) (bool, error) {
	return identity.MemberExists(ctx, id)
}
func (targets) GuildExists(ctx context.Context, id uint64) (bool, error) {
	return guilds.GuildExists(ctx, id)
}

// directory reads members, guilds and boards through the contracts the
// identity, guilds and boards contexts publish, and translates them.
type directory struct{}

func memberCards(cs []identity.MemberCard) []app.MemberCard {
	out := make([]app.MemberCard, len(cs))
	for i, c := range cs {
		out[i] = app.MemberCard(c)
	}
	return out
}

func guildCards(cs []guilds.GuildCard) []app.GuildCard {
	out := make([]app.GuildCard, len(cs))
	for i, c := range cs {
		out[i] = app.GuildCard(c)
	}
	return out
}

func (directory) FindMembers(ctx context.Context, q string, limit int) ([]app.MemberCard, error) {
	cs, err := identity.FindMembers(ctx, q, limit)
	return memberCards(cs), err
}
func (directory) Members(ctx context.Context, ids []uint64) ([]app.MemberCard, error) {
	cs, err := identity.MemberCards(ctx, ids)
	return memberCards(cs), err
}
func (directory) GuildCounts(ctx context.Context, memberIDs []uint64) (map[uint64]int, error) {
	return guilds.GuildCounts(ctx, memberIDs)
}
func (directory) FindGuilds(ctx context.Context, q string, limit int) ([]app.GuildCard, error) {
	cs, err := guilds.FindGuilds(ctx, q, limit)
	return guildCards(cs), err
}
func (directory) Guild(ctx context.Context, id uint64) (app.GuildCard, bool, error) {
	c, found, err := guilds.GuildCardOf(ctx, id)
	return app.GuildCard(c), found, err
}
func (directory) GuildsOfMember(ctx context.Context, memberID uint64) ([]app.GuildCard, error) {
	cs, err := guilds.GuildCardsOfMember(ctx, memberID)
	return guildCards(cs), err
}
func (directory) BoardCount(ctx context.Context, guildID uint64) (int, error) {
	return boards.BoardCount(ctx, guildID)
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
	// Every rate-limit refusal is an abuse signal.
	limits.OnHit(func(ctx context.Context, limit string, memberID uint64, ip string) {
		recordSignal(ctx, domain.SignalLimitHit, limit, memberID, ip)
	})
}

// signalOf is the abuse signal an audited member action also is.
var signalOf = map[string]string{
	"member.registered": domain.SignalSignUp,
	"guild.founded":     domain.SignalGuildFounded,
}

func recordSignal(ctx context.Context, kind, limit string, memberID uint64, ip string) {
	if err := service.RecordSignal(ctx, kind, limit, memberID, ip); err != nil {
		facades.Log().WithContext(ctx).Errorf("signal %s: %v", kind, err)
	}
}

// recordMember writes a member's sensitive action to the audit log, and a
// sign-up or a guild founded to the abuse signals too.
func recordMember(ctx context.Context, actorMemberID uint64, action, targetKind string, targetID uint64, meta map[string]any) {
	if kind, ok := signalOf[action]; ok {
		recordSignal(ctx, kind, "", actorMemberID, "")
	}
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
		r.Get("/api/console/signals", c.Signals)
		r.Get("/api/console/members", c.ListMembers)
		r.Get("/api/console/members/{member}", c.ShowMember)
		r.Get("/api/console/guilds", c.ListGuilds)
		r.Get("/api/console/guilds/{guild}", c.ShowGuild)
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
