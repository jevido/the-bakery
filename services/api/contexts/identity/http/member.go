// Package http exposes identity over HTTP: register, login, me, and the
// RequireMember middleware other contexts put in front of their routes.
package http

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/identity/app"
	"github.com/jevido/the-bakery/services/api/contexts/identity/domain"
)

// Guard is the auth guard (config/auth.go) that issues member tokens.
const Guard = "member"

// SessionCookie carries a web session: the same JWT the desktop sends as a
// bearer token, in an httpOnly cookie the website's scripts cannot read.
const SessionCookie = "bakery_session"

// WebHeader must be on every cookie-authenticated request that changes
// something. The website's fetches set it; a form on another site cannot, and
// a cross-origin script cannot either without passing CORS. Bearer requests
// do not need it.
const WebHeader = "X-Bakery-Web"

type memberContextKey struct{}

// authKindKey records how the request signed in: "session" (desktop JWT or
// web cookie) or "token" (a personal token).
type authKindKey struct{}

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

// WebRegister is Register for the website: the token goes into the session
// cookie instead of the body.
func (c *Controller) WebRegister(ctx contractshttp.Context) contractshttp.Response {
	if !fromWebsite(ctx) {
		return missingWebHeader(ctx)
	}
	var req registerRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return ctx.Response().Json(contractshttp.StatusBadRequest, contractshttp.Json{"error": "request body must be JSON"})
	}
	m, err := c.service.Register(ctx.Context(), req.Email, req.DisplayName, req.Password)
	if err != nil {
		return registerError(ctx, err)
	}
	return c.withSession(ctx, contractshttp.StatusCreated, m)
}

// WebLogin is Login for the website.
func (c *Controller) WebLogin(ctx contractshttp.Context) contractshttp.Response {
	if !fromWebsite(ctx) {
		return missingWebHeader(ctx)
	}
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
	return c.withSession(ctx, contractshttp.StatusOK, m)
}

// WebLogout ends the web session by expiring its cookie.
func (c *Controller) WebLogout(ctx contractshttp.Context) contractshttp.Response {
	if !fromWebsite(ctx) {
		return missingWebHeader(ctx)
	}
	ctx.Response().Cookie(sessionCookie("", -1))
	return ctx.Response().NoContent()
}

// CreateHandoff gives the signed-in member (the desktop app, by bearer
// token) a one-time code to open the website signed in.
func (c *Controller) CreateHandoff(ctx contractshttp.Context) contractshttp.Response {
	id, _ := MemberID(ctx)
	code, err := c.service.CreateHandoff(ctx.Context(), id)
	if err != nil {
		return serverError(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"code": code, "expires_in": int(app.HandoffTTL.Seconds())})
}

type redeemRequest struct {
	Code string `json:"code"`
}

// RedeemHandoff trades a handoff code for a web session.
func (c *Controller) RedeemHandoff(ctx contractshttp.Context) contractshttp.Response {
	if !fromWebsite(ctx) {
		return missingWebHeader(ctx)
	}
	var req redeemRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return ctx.Response().Json(contractshttp.StatusBadRequest, contractshttp.Json{"error": "request body must be JSON"})
	}
	id, err := c.service.RedeemHandoff(ctx.Context(), req.Code)
	if errors.Is(err, app.ErrHandoffInvalid) {
		return unauthorized(ctx, err.Error())
	}
	if err != nil {
		return serverError(ctx, err)
	}
	m, err := c.service.CurrentMember(ctx.Context(), id)
	if err != nil {
		return unauthorized(ctx, app.ErrHandoffInvalid.Error())
	}
	return c.withSession(ctx, contractshttp.StatusOK, m)
}

type tokenJSON struct {
	ID         uint64     `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

func tokenToJSON(t domain.PersonalToken) tokenJSON {
	return tokenJSON{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt, LastUsedAt: t.LastUsedAt}
}

// A personal token cannot see or make tokens: a leaked one must not be able
// to mint more.
func (c *Controller) refuseTokenAuth(ctx contractshttp.Context) contractshttp.Response {
	return ctx.Response().Json(contractshttp.StatusForbidden, contractshttp.Json{"error": "personal tokens cannot manage tokens; sign in on the website or the desktop app"})
}

func (c *Controller) ListTokens(ctx contractshttp.Context) contractshttp.Response {
	if signedInWithToken(ctx) {
		return c.refuseTokenAuth(ctx)
	}
	id, _ := MemberID(ctx)
	tokens, err := c.service.ListPersonalTokens(ctx.Context(), id)
	if err != nil {
		return serverError(ctx, err)
	}
	out := make([]tokenJSON, len(tokens))
	for i, t := range tokens {
		out[i] = tokenToJSON(t)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"tokens": out})
}

type createTokenRequest struct {
	Name string `json:"name"`
}

// CreateToken returns the secret once, in `token`; it cannot be shown again.
func (c *Controller) CreateToken(ctx contractshttp.Context) contractshttp.Response {
	if signedInWithToken(ctx) {
		return c.refuseTokenAuth(ctx)
	}
	var req createTokenRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return ctx.Response().Json(contractshttp.StatusBadRequest, contractshttp.Json{"error": "request body must be JSON"})
	}
	id, _ := MemberID(ctx)
	t, secret, err := c.service.CreatePersonalToken(ctx.Context(), id, req.Name)
	if errors.Is(err, domain.ErrInvalidTokenName) {
		return ctx.Response().Json(contractshttp.StatusUnprocessableEntity, contractshttp.Json{"error": err.Error(), "field": "name"})
	}
	if err != nil {
		return serverError(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"token": secret, "personal_token": tokenToJSON(t)})
}

func (c *Controller) RevokeToken(ctx contractshttp.Context) contractshttp.Response {
	if signedInWithToken(ctx) {
		return c.refuseTokenAuth(ctx)
	}
	tokenID, err := strconv.ParseUint(ctx.Request().Route("token"), 10, 64)
	id, _ := MemberID(ctx)
	if err == nil {
		err = c.service.RevokePersonalToken(ctx.Context(), id, tokenID)
	} else {
		err = app.ErrTokenNotFound
	}
	if errors.Is(err, app.ErrTokenNotFound) {
		return ctx.Response().Json(contractshttp.StatusNotFound, contractshttp.Json{"error": err.Error()})
	}
	if err != nil {
		return serverError(ctx, err)
	}
	return ctx.Response().NoContent()
}

func (c *Controller) withSession(ctx contractshttp.Context, status int, m domain.Member) contractshttp.Response {
	token, err := facades.Auth(ctx).Guard(Guard).LoginUsingID(m.ID)
	if err != nil {
		return serverError(ctx, err)
	}
	ctx.Response().Cookie(sessionCookie(token, facades.Config().GetInt("jwt.ttl")*60))
	return ctx.Response().Json(status, contractshttp.Json{"member": toJSON(m)})
}

// sessionCookie is host-only on the API (no Domain): the website and the API
// are same-site, so SameSite=Lax cookies ride along on the website's
// credentialed fetches, and next's cookie never reaches prod or back.
func sessionCookie(value string, maxAge int) contractshttp.Cookie {
	return contractshttp.Cookie{
		Name:     SessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   facades.Config().GetString("app.env") != "local",
		SameSite: "lax",
	}
}

func fromWebsite(ctx contractshttp.Context) bool {
	return ctx.Request().Header(WebHeader) == "1"
}

func missingWebHeader(ctx contractshttp.Context) contractshttp.AbortableResponse {
	return ctx.Response().Json(contractshttp.StatusForbidden, contractshttp.Json{"error": "cookie requests that change something need the " + WebHeader + " header"})
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
	// A bearer token (the desktop app or a personal token) wins; otherwise
	// the web session cookie.
	token := ctx.Request().Header("Authorization")
	if secret, ok := strings.CutPrefix(token, "Bearer "); ok && strings.HasPrefix(secret, domain.PersonalTokenPrefix) {
		id, err := verifyPersonalToken(ctx.Context(), secret)
		if err != nil {
			_ = unauthorized(ctx, "not signed in").Abort()
			return
		}
		ctx.WithValue(memberContextKey{}, id)
		ctx.WithValue(authKindKey{}, "token")
		ctx.Request().Next()
		return
	}
	if token == "" {
		token = ctx.Request().Cookie(SessionCookie)
		if token == "" {
			_ = unauthorized(ctx, "not signed in").Abort()
			return
		}
		if changesSomething(ctx.Request().Method()) && !fromWebsite(ctx) {
			_ = missingWebHeader(ctx).Abort()
			return
		}
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
	ctx.WithValue(authKindKey{}, "session")
	ctx.Request().Next()
}

// verifyPersonalToken is set by the context's wiring (identity.go), since
// this middleware has no service of its own.
var verifyPersonalToken func(ctx context.Context, secret string) (uint64, error)

// SetPersonalTokenVerifier wires the middleware to the token use case.
func SetPersonalTokenVerifier(f func(ctx context.Context, secret string) (uint64, error)) {
	verifyPersonalToken = f
}

func signedInWithToken(ctx contractshttp.Context) bool {
	kind, _ := ctx.Value(authKindKey{}).(string)
	return kind == "token"
}

func changesSomething(method string) bool {
	switch method {
	case "GET", "HEAD", "OPTIONS":
		return false
	}
	return true
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
