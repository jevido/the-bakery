package http

import (
	"strconv"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
)

type activityJSON struct {
	ID        uint64         `json:"id"`
	Kind      string         `json:"kind"`
	ActorID   uint64         `json:"actor_id"`
	ActorName string         `json:"actor_name"`
	At        time.Time      `json:"at"`
	Data      map[string]any `json:"data"`
}

// ListActivity returns a task's history, newest first:
// ?before=<id> for the next page, ?limit= up to 100 (default 50).
func (c *Controller) ListActivity(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	before, _ := strconv.ParseUint(ctx.Request().Query("before", "0"), 10, 64)
	limit, _ := strconv.Atoi(ctx.Request().Query("limit", "0"))
	entries, err := c.service.ListActivity(ctx.Context(), taskID, c.me(ctx), before, limit)
	if err != nil {
		return failure(ctx, err)
	}
	out := make([]activityJSON, len(entries))
	for i, e := range entries {
		out[i] = activityJSON{ID: e.ID, Kind: string(e.Kind), ActorID: e.ActorID, ActorName: e.ActorName, At: e.At, Data: e.Data}
	}
	return ctx.Response().Success().Json(contractshttp.Json{"activity": out})
}
