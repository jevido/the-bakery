package routes

import (
	"github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

func Api() {
	facades.Route().Get("/api/health", health)
	identity.Routes(facades.Route())
}

// health answers ok only when the database answers too, so a green health
// check means the API can actually serve requests.
func health(ctx http.Context) http.Response {
	db, err := facades.Orm().DB()
	if err == nil {
		err = db.PingContext(ctx.Context())
	}
	if err != nil {
		return ctx.Response().Json(http.StatusServiceUnavailable, http.Json{"ok": false})
	}
	return ctx.Response().Success().Json(http.Json{"ok": true})
}
