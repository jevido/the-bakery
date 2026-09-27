package routes

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/boards"
	"github.com/jevido/the-bakery/services/api/contexts/guilds"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

// draining is set when the container is about to stop (SIGUSR1 from
// infra/images/api/entrypoint.sh). Health then fails so the proxy stops
// sending traffic here, while requests that still arrive are served.
var draining atomic.Bool

func init() {
	usr1 := make(chan os.Signal, 1)
	signal.Notify(usr1, syscall.SIGUSR1)
	go func() {
		for range usr1 {
			draining.Store(true)
		}
	}()
}

func Api() {
	facades.Route().Get("/api/health", health)
	identity.Routes(facades.Route())
	guilds.Routes(facades.Route())
	boards.Routes(facades.Route())
}

// health answers ok only when the database answers too, so a green health
// check means the API can actually serve requests. While draining it answers
// 503 so the container is taken out of rotation before it stops.
func health(ctx http.Context) http.Response {
	if draining.Load() {
		return ctx.Response().Json(http.StatusServiceUnavailable, http.Json{"ok": false, "draining": true})
	}
	db, err := facades.Orm().DB()
	if err == nil {
		err = db.PingContext(ctx.Context())
	}
	if err != nil {
		return ctx.Response().Json(http.StatusServiceUnavailable, http.Json{"ok": false})
	}
	return ctx.Response().Success().Json(http.Json{"ok": true})
}
