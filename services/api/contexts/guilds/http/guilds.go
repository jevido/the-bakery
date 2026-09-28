// Package http exposes guilds over HTTP. Every route sits behind identity's
// RequireMember.
package http

import (
	"context"
	"errors"
	"strconv"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/app/refusal"
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
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
}

func toJSON(g domain.Guild) guildJSON { return guildJSON{ID: g.ID, Name: g.Name, Archived: g.Archived} }

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
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"guild": toJSON(g)})
}

func (c *Controller) Mine(ctx contractshttp.Context) contractshttp.Response {
	me, _ := c.memberID(ctx)
	// ?archived=1 includes archived guilds.
	guilds, err := c.service.ListGuildsOf(ctx.Context(), me, ctx.Request().Query("archived") == "1")
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
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	var req addMemberRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	me, _ := c.memberID(ctx)
	memberID, err := c.service.AddMember(ctx.Context(), guildID, me, req.Email)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{
		"membership": contractshttp.Json{"guild_id": guildID, "member_id": memberID},
	})
}

// Show returns one guild the member is in, archived or not.
func (c *Controller) Show(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	me, _ := c.memberID(ctx)
	g, err := c.service.Guild(ctx.Context(), guildID, me)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"guild": toJSON(g)})
}

type renameRequest struct {
	Name string `json:"name"`
}

func (c *Controller) Rename(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	var req renameRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	me, _ := c.memberID(ctx)
	g, err := c.service.RenameGuild(ctx.Context(), guildID, me, req.Name)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"guild": toJSON(g)})
}

func (c *Controller) Archive(ctx contractshttp.Context) contractshttp.Response {
	return c.guildAction(ctx, c.service.ArchiveGuild)
}

func (c *Controller) Restore(ctx contractshttp.Context) contractshttp.Response {
	return c.guildAction(ctx, c.service.RestoreGuild)
}

func (c *Controller) guildAction(ctx contractshttp.Context, f func(context.Context, uint64, uint64) (domain.Guild, error)) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	me, _ := c.memberID(ctx)
	g, err := f(ctx.Context(), guildID, me)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"guild": toJSON(g)})
}

type memberJSON struct {
	ID           uint64    `json:"id"`
	DisplayName  string    `json:"display_name"`
	PortraitSeed string    `json:"portrait_seed"`
	JoinedAt     time.Time `json:"joined_at"`
}

func (c *Controller) Members(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	me, _ := c.memberID(ctx)
	members, err := c.service.Members(ctx.Context(), guildID, me)
	if err != nil {
		return failure(ctx, err)
	}
	out := make([]memberJSON, len(members))
	for i, m := range members {
		out[i] = memberJSON{ID: m.MemberID, DisplayName: m.DisplayName, PortraitSeed: m.PortraitSeed, JoinedAt: m.JoinedAt}
	}
	return ctx.Response().Success().Json(contractshttp.Json{"members": out})
}

func (c *Controller) RemoveMember(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	memberID, ok2 := routeID(ctx, "member")
	if !ok || !ok2 {
		return notFound(ctx)
	}
	me, _ := c.memberID(ctx)
	if err := c.service.RemoveMember(ctx.Context(), guildID, me, memberID); err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().NoContent()
}

func (c *Controller) Leave(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	me, _ := c.memberID(ctx)
	if err := c.service.Leave(ctx.Context(), guildID, me); err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().NoContent()
}

type inviteJSON struct {
	ID        uint64     `json:"id"`
	GuildID   uint64     `json:"guild_id"`
	Code      string     `json:"code"`
	CreatedBy uint64     `json:"created_by"`
	ExpiresAt *time.Time `json:"expires_at"`
	MaxUses   *int       `json:"max_uses"`
	Uses      int        `json:"uses"`
	// State is active, expired, used_up or revoked.
	State string `json:"state"`
}

func inviteToJSON(inv domain.Invite) inviteJSON {
	state := "active"
	switch err := inv.Usable(time.Now()); {
	case errors.Is(err, domain.ErrInviteRevoked):
		state = "revoked"
	case errors.Is(err, domain.ErrInviteExpired):
		state = "expired"
	case errors.Is(err, domain.ErrInviteUsedUp):
		state = "used_up"
	}
	return inviteJSON{ID: inv.ID, GuildID: inv.GuildID, Code: inv.Code, CreatedBy: inv.CreatedBy,
		ExpiresAt: inv.ExpiresAt, MaxUses: inv.MaxUses, Uses: inv.Uses, State: state}
}

type createInviteRequest struct {
	ExpiresInHours *int `json:"expires_in_hours"`
	MaxUses        *int `json:"max_uses"`
}

func (c *Controller) CreateInvite(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	var req createInviteRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	var expiresIn *time.Duration
	if req.ExpiresInHours != nil {
		d := time.Duration(*req.ExpiresInHours) * time.Hour
		expiresIn = &d
	}
	me, _ := c.memberID(ctx)
	inv, err := c.service.CreateInvite(ctx.Context(), guildID, me, expiresIn, req.MaxUses)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"invite": inviteToJSON(inv)})
}

func (c *Controller) ListInvites(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	me, _ := c.memberID(ctx)
	invites, err := c.service.ListInvites(ctx.Context(), guildID, me)
	if err != nil {
		return failure(ctx, err)
	}
	out := make([]inviteJSON, len(invites))
	for i, inv := range invites {
		out[i] = inviteToJSON(inv)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"invites": out})
}

func (c *Controller) RevokeInvite(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	inviteID, ok2 := routeID(ctx, "invite")
	if !ok || !ok2 {
		return notFound(ctx)
	}
	me, _ := c.memberID(ctx)
	if err := c.service.RevokeInvite(ctx.Context(), guildID, inviteID, me); err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().NoContent()
}

// ShowInvite is public: anyone holding the link sees the guild's name and
// size, and whether the invite still works.
func (c *Controller) ShowInvite(ctx contractshttp.Context) contractshttp.Response {
	info, err := c.service.Invite(ctx.Context(), ctx.Request().Route("code"))
	if err != nil {
		return failure(ctx, err)
	}
	body := contractshttp.Json{"guild_name": info.GuildName, "member_count": info.MemberCount, "valid": info.Problem == nil}
	if info.Problem != nil {
		body["problem"] = info.Problem.Error()
	}
	return ctx.Response().Success().Json(contractshttp.Json{"invite": body})
}

// AcceptInvite makes the signed-in member a member of the invite's guild.
func (c *Controller) AcceptInvite(ctx contractshttp.Context) contractshttp.Response {
	me, _ := c.memberID(ctx)
	g, err := c.service.AcceptInvite(ctx.Context(), ctx.Request().Route("code"), me)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"guild": toJSON(g)})
}

func routeID(ctx contractshttp.Context, key string) (uint64, bool) {
	id, err := strconv.ParseUint(ctx.Request().Route(key), 10, 64)
	return id, err == nil
}

// failure maps guilds' errors to HTTP answers.
func failure(ctx contractshttp.Context, err error) contractshttp.Response {
	// A sanction (of the member or the guild) answers for itself.
	if r, ok := refusal.As(err); ok {
		return refusal.Respond(ctx, r)
	}
	status := contractshttp.StatusInternalServerError
	field := ""
	switch {
	case errors.Is(err, app.ErrNotMember):
		status = contractshttp.StatusForbidden
	case errors.Is(err, domain.ErrNotAMember):
		status = contractshttp.StatusNotFound
	case errors.Is(err, domain.ErrAlreadyMember), errors.Is(err, domain.ErrArchived), errors.Is(err, domain.ErrNotArchived):
		status = contractshttp.StatusConflict
	case errors.Is(err, domain.ErrLastMember):
		status = contractshttp.StatusUnprocessableEntity
	case errors.Is(err, app.ErrInviteNotFound):
		status = contractshttp.StatusNotFound
	case errors.Is(err, domain.ErrInviteRevoked), errors.Is(err, domain.ErrInviteExpired), errors.Is(err, domain.ErrInviteUsedUp):
		status = contractshttp.StatusGone
	case errors.Is(err, domain.ErrInvalidInviteRule):
		status, field = contractshttp.StatusUnprocessableEntity, "expires_in_hours"
	case errors.Is(err, domain.ErrInvalidName):
		status, field = contractshttp.StatusUnprocessableEntity, "name"
	case errors.Is(err, app.ErrUnknownEmail):
		status, field = contractshttp.StatusUnprocessableEntity, "email"
	}
	if status == contractshttp.StatusInternalServerError {
		return serverError(ctx, err)
	}
	body := contractshttp.Json{"error": err.Error()}
	if field != "" {
		body["field"] = field
	}
	return ctx.Response().Json(status, body)
}

func notFound(ctx contractshttp.Context) contractshttp.Response {
	return ctx.Response().Json(contractshttp.StatusNotFound, contractshttp.Json{"error": "not found"})
}

func badRequest(ctx contractshttp.Context) contractshttp.Response {
	return ctx.Response().Json(contractshttp.StatusBadRequest, contractshttp.Json{"error": "request body must be JSON"})
}

func serverError(ctx contractshttp.Context, err error) contractshttp.Response {
	facades.Log().WithContext(ctx.Context()).Error(err)
	return ctx.Response().Json(contractshttp.StatusInternalServerError, contractshttp.Json{"error": "something went wrong"})
}
