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
	"github.com/jevido/the-bakery/services/api/contexts/guilds"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
	moderationhttp "github.com/jevido/the-bakery/services/api/contexts/moderation/http"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/infra"
)

var service = app.NewService(infra.Operators{}, infra.Hasher{}, infra.Secrets{}, infra.TOTP{}, infra.AuditEntries{})

// RememberIP puts the client's IP in each request's context for the audit
// log. The router runs it on every API route.
var RememberIP contractshttp.Middleware = moderationhttp.RememberIP{}

func init() {
	// The contexts that record sensitive actions do it through this.
	identity.SetAuditRecorder(recordMember)
	guilds.SetAuditRecorder(recordMember)
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
	})
}
