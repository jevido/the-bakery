package routes

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
	goravelgin "github.com/goravel/gin"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/agents"
	"github.com/jevido/the-bakery/services/api/contexts/boards"
	"github.com/jevido/the-bakery/services/api/contexts/guilds"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
	"github.com/jevido/the-bakery/services/api/contexts/moderation"
	"github.com/jevido/the-bakery/services/api/mcp"
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
			// Board event streams never finish by themselves; end them so
			// their clients reconnect to the process taking over.
			boards.CloseStreams()
		}
	}()
	// The same on a stop: graceful shutdown waits for open requests, and a
	// stream would keep it waiting until the process is killed.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-stop
		boards.CloseStreams()
	}()
}

// requestTimeout bounds every request except the board event stream.
// config/http.go turns Goravel's global timeout off (request_timeout 0)
// because that middleware buffers the whole response, which a stream
// cannot live with; it is applied here per group instead.
const requestTimeout = 3 * time.Second

func Api() {
	facades.Route().Middleware(goravelgin.Timeout(requestTimeout), moderation.RememberIP).Group(func(r route.Router) {
		r.Get("/api/health", health)
		identity.Routes(r)
		guilds.Routes(r)
		boards.Routes(r)
		agents.Routes(r)
		moderation.Routes(r)

		// MCP over streamable HTTP. The SDK handler does its own auth (personal
		// tokens) and writes the response itself.
		mcpHandler := mcp.Handler()
		r.Any("/mcp", func(ctx http.Context) http.Response {
			mcpHandler.ServeHTTP(ctx.Response().Writer(), ctx.Request().Origin())
			return nil
		})
	})
	boards.StreamRoutes(facades.Route())
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
