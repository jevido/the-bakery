// Package http exposes guilds over HTTP. Every route sits behind identity's
// RequireMember.
package http

import (
	"errors"
	"strconv"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/guilds/app"
	"github.com/jevido/the-bakery/services/api/contexts/guilds/domain"
)

// MemberID returns the signed-in member's id. It is identity's, handed in by
// the context's wiring so this package does not import identity.
type MemberID func(ctx contractshttp.Context) (uint64, bool)

type Controller struct {
	service  *app.Service
	memberID MemberID
}

func NewController(service *app.Service, memberID MemberID) *Controller {
	return &Controller{service: service, memberID: memberID}
}

type guildJSON struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

func toJSON(g domain.Guild) guildJSON { return guildJSON{ID: g.ID, Name: g.Name} }

type foundRequest struct {
	Name string `json:"name"`
}

func (c *Controller) Found(ctx contractshttp.Context) contractshttp.Response {
	var req foundRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	me, _ := c.memberID(ctx)
	g, err := c.service.FoundGuild(ctx.Context(), req.Name, me)
	if errors.Is(err, domain.ErrInvalidName) {
		return invalid(ctx, err, "name")
	}
	if err != nil {
		return serverError(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"guild": toJSON(g)})
}

func (c *Controller) Mine(ctx contractshttp.Context) contractshttp.Response {
	me, _ := c.memberID(ctx)
	guilds, err := c.service.ListGuildsOf(ctx.Context(), me)
	if err != nil {
		return serverError(ctx, err)
	}
	out := make([]guildJSON, len(guilds))
	for i, g := range guilds {
		out[i] = toJSON(g)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"guilds": out})
}

type addMemberRequest struct {
	Email string `json:"email"`
}

func (c *Controller) AddMember(ctx contractshttp.Context) contractshttp.Response {
	guildID, err := strconv.ParseUint(ctx.Request().Route("guild"), 10, 64)
	if err != nil {
		return ctx.Response().Json(contractshttp.StatusNotFound, contractshttp.Json{"error": "no such guild"})
	}
	var req addMemberRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	me, _ := c.memberID(ctx)
	memberID, err := c.service.AddMember(ctx.Context(), guildID, me, req.Email)
	switch {
	case errors.Is(err, app.ErrNotMember):
		return ctx.Response().Json(contractshttp.StatusForbidden, contractshttp.Json{"error": err.Error()})
	case errors.Is(err, app.ErrUnknownEmail):
		return invalid(ctx, err, "email")
	case errors.Is(err, domain.ErrAlreadyMember):
		return ctx.Response().Json(contractshttp.StatusConflict, contractshttp.Json{"error": err.Error()})
	case err != nil:
		return serverError(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{
		"membership": contractshttp.Json{"guild_id": guildID, "member_id": memberID},
	})
}

func badRequest(ctx contractshttp.Context) contractshttp.Response {
	return ctx.Response().Json(contractshttp.StatusBadRequest, contractshttp.Json{"error": "request body must be JSON"})
}

func invalid(ctx contractshttp.Context, err error, field string) contractshttp.Response {
	return ctx.Response().Json(contractshttp.StatusUnprocessableEntity, contractshttp.Json{"error": err.Error(), "field": field})
}

func serverError(ctx contractshttp.Context, err error) contractshttp.Response {
	facades.Log().WithContext(ctx.Context()).Error(err)
	return ctx.Response().Json(contractshttp.StatusInternalServerError, contractshttp.Json{"error": "something went wrong"})
}
