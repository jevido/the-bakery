// Package moderation wires the moderation context: the console's routes
// (/api/console/*, operators only) and the artisan commands that make
// operators. It keeps bad actors out; see docs/domain/contexts/moderation.
package moderation

import (
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/http/limit"
	"github.com/goravel/framework/http/middleware"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
	moderationhttp "github.com/jevido/the-bakery/services/api/contexts/moderation/http"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/infra"
)

var service = app.NewService(infra.Operators{}, infra.Hasher{}, infra.Secrets{}, infra.TOTP{})

// RequireOperator is the console's middleware.
var RequireOperator contractshttp.Middleware = moderationhttp.RequireOperator{}

// Routes registers the console's routes. Signing in is limited to five
// tries a minute per client, each step.
func Routes(r route.Router) {
	facades.RateLimiter().For("console-sign-in", func(ctx contractshttp.Context) contractshttp.Limit {
		return limit.PerMinute(5)
	})
	c := moderationhttp.NewController(service, nil)
	r.Middleware(middleware.Throttle("console-sign-in")).Group(func(r route.Router) {
		r.Post("/api/console/login", c.Login)
		r.Post("/api/console/totp/verify", c.VerifyTOTP)
	})
	r.Middleware(RequireOperator).Group(func(r route.Router) {
		r.Get("/api/console/me", c.Me)
	})
}
