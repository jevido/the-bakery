// Package http exposes identity over HTTP: register, login, me, and the
// RequireMember middleware other contexts put in front of their routes.
package http

import (
	"errors"
	"strconv"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/identity/app"
	"github.com/jevido/the-bakery/services/api/contexts/identity/domain"
)

// Guard is the auth guard (config/auth.go) that issues member tokens.
const Guard = "member"

type memberContextKey struct{}

type Controller struct {
	service *app.Service
}

func NewController(service *app.Service) *Controller {
	return &Controller{service: service}
}

type memberJSON struct {
	ID          uint64 `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func toJSON(m domain.Member) memberJSON {
	return memberJSON{ID: m.ID, Email: m.Email, DisplayName: m.DisplayName}
}

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

func (c *Controller) Register(ctx contractshttp.Context) contractshttp.Response {
	var req registerRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return ctx.Response().Json(contractshttp.StatusBadRequest, contractshttp.Json{"error": "request body must be JSON"})
	}
	m, err := c.service.Register(ctx.Context(), req.Email, req.DisplayName, req.Password)
	if err != nil {
		return registerError(ctx, err)
	}
	return c.withToken(ctx, contractshttp.StatusCreated, m)
}

func registerError(ctx contractshttp.Context, err error) contractshttp.Response {
	field := ""
	switch {
	case errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, app.ErrEmailTaken):
		field = "email"
	case errors.Is(err, domain.ErrInvalidDisplayName):
		field = "display_name"
	case errors.Is(err, domain.ErrPasswordTooShort):
		field = "password"
	default:
		return serverError(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusUnprocessableEntity, contractshttp.Json{"error": err.Error(), "field": field})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (c *Controller) Login(ctx contractshttp.Context) contractshttp.Response {
	var req loginRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return ctx.Response().Json(contractshttp.StatusBadRequest, contractshttp.Json{"error": "request body must be JSON"})
	}
	m, err := c.service.Login(ctx.Context(), req.Email, req.Password)
	if errors.Is(err, app.ErrBadCredentials) {
		return unauthorized(ctx, err.Error())
	}
	if err != nil {
		return serverError(ctx, err)
	}
	return c.withToken(ctx, contractshttp.StatusOK, m)
}

func (c *Controller) Me(ctx contractshttp.Context) contractshttp.Response {
	id, _ := MemberID(ctx)
	m, err := c.service.CurrentMember(ctx.Context(), id)
	if errors.Is(err, app.ErrMemberNotFound) {
		// A valid token for a member that no longer exists.
		return unauthorized(ctx, "not signed in")
	}
	if err != nil {
		return serverError(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"member": toJSON(m)})
}

func (c *Controller) withToken(ctx contractshttp.Context, status int, m domain.Member) contractshttp.Response {
	token, err := facades.Auth(ctx).Guard(Guard).LoginUsingID(m.ID)
	if err != nil {
		return serverError(ctx, err)
	}
	return ctx.Response().Json(status, contractshttp.Json{"token": token, "member": toJSON(m)})
}

// RequireMember lets a request through only with a valid member token, and
// puts the member id on the context for MemberID.
type RequireMember struct{}

func (RequireMember) Signature() string { return "identity.require_member" }

func (RequireMember) Handle(ctx contractshttp.Context) {
	token := ctx.Request().Header("Authorization")
	if token == "" {
		_ = unauthorized(ctx, "not signed in").Abort()
		return
	}
	payload, err := facades.Auth(ctx).Guard(Guard).Parse(token)
	if err != nil {
		_ = unauthorized(ctx, "not signed in").Abort()
		return
	}
	id, err := strconv.ParseUint(payload.Key, 10, 64)
	if err != nil {
		_ = unauthorized(ctx, "not signed in").Abort()
		return
	}
	ctx.WithValue(memberContextKey{}, id)
	ctx.Request().Next()
}

// MemberID returns the id of the member RequireMember let through.
func MemberID(ctx contractshttp.Context) (uint64, bool) {
	id, ok := ctx.Value(memberContextKey{}).(uint64)
	return id, ok
}

func unauthorized(ctx contractshttp.Context, msg string) contractshttp.AbortableResponse {
	return ctx.Response().Json(contractshttp.StatusUnauthorized, contractshttp.Json{"error": msg})
}

func serverError(ctx contractshttp.Context, err error) contractshttp.Response {
	facades.Log().WithContext(ctx.Context()).Error(err)
	return ctx.Response().Json(contractshttp.StatusInternalServerError, contractshttp.Json{"error": "something went wrong"})
}
