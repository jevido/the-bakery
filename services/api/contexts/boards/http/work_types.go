package http

import (
	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

type workTypeJSON struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func workTypeToJSON(wt domain.WorkType) workTypeJSON {
	return workTypeJSON{Key: wt.Key, Name: wt.Name}
}

// ListWorkTypes returns the guild's work types in order.
func (c *Controller) ListWorkTypes(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	wts, err := c.service.ListWorkTypes(ctx.Context(), guildID, c.me(ctx))
	if err != nil {
		return failure(ctx, err)
	}
	out := make([]workTypeJSON, len(wts))
	for i, wt := range wts {
		out[i] = workTypeToJSON(wt)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"work_types": out})
}

func (c *Controller) AddWorkType(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	var req workTypeJSON
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	wt, err := c.service.AddWorkType(ctx.Context(), guildID, c.me(ctx), req.Key, req.Name)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"work_type": workTypeToJSON(wt)})
}

// updateWorkTypeRequest: name renames, position (0-based) moves it in the list.
type updateWorkTypeRequest struct {
	Name     *string `json:"name"`
	Position *int    `json:"position"`
}

func (c *Controller) UpdateWorkType(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	var req updateWorkTypeRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	wt, err := c.service.UpdateWorkType(ctx.Context(), guildID, c.me(ctx), ctx.Request().Route("key"), req.Name, req.Position)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"work_type": workTypeToJSON(wt)})
}

// DeleteWorkType removes a work type no task has (409 otherwise).
func (c *Controller) DeleteWorkType(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	if err := c.service.DeleteWorkType(ctx.Context(), guildID, c.me(ctx), ctx.Request().Route("key")); err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().NoContent()
}
