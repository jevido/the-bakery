// Package http is the console's side of the API: /api/console/*, for
// operators only.
package http

import (
	"errors"
	"strconv"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

// The guards: an operator's token, and the short-lived one between the
// password and the code. Neither accepts a member's token, nor a member's
// guard theirs: a token names its guard.
const (
	OperatorGuard  = "operator"
	ChallengeGuard = "operator_challenge"
)

type operatorKey struct{}

// OperatorID is the signed-in operator, behind RequireOperator.
func OperatorID(ctx contractshttp.Context) uint64 {
	id, _ := ctx.Value(operatorKey{}).(uint64)
	return id
}

type Controller struct {
	service *app.Service
	// onSignIn hears every operator sign-in, for the audit log.
	onSignIn func(ctx contractshttp.Context, operatorID uint64)
}

func NewController(service *app.Service, onSignIn func(contractshttp.Context, uint64)) *Controller {
	return &Controller{service: service, onSignIn: onSignIn}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login is the first step: email and password give a challenge, good for
// five minutes, for the second step.
func (c *Controller) Login(ctx contractshttp.Context) contractshttp.Response {
	var req loginRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	id, err := c.service.CheckPassword(ctx.Context(), req.Email, req.Password)
	if err != nil {
		return failure(ctx, err)
	}
	challenge, err := facades.Auth(ctx).Guard(ChallengeGuard).LoginUsingID(id)
	if err != nil {
		return serverError(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"challenge": challenge})
}

type verifyRequest struct {
	Challenge string `json:"challenge"`
	Code      string `json:"code"`
}

// VerifyTOTP is the second step: the challenge and a code from the
// authenticator give the operator's token.
func (c *Controller) VerifyTOTP(ctx contractshttp.Context) contractshttp.Response {
	var req verifyRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	payload, err := facades.Auth(ctx).Guard(ChallengeGuard).Parse(req.Challenge)
	if err != nil {
		return unauthorized(ctx, "sign in again")
	}
	id, err := strconv.ParseUint(payload.Key, 10, 64)
	if err != nil {
		return unauthorized(ctx, "sign in again")
	}
	if err := c.service.CheckCode(ctx.Context(), id, req.Code); err != nil {
		return failure(ctx, err)
	}
	token, err := facades.Auth(ctx).Guard(OperatorGuard).LoginUsingID(id)
	if err != nil {
		return serverError(ctx, err)
	}
	if c.onSignIn != nil {
		c.onSignIn(ctx, id)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"token": token})
}

// Me is the signed-in operator.
func (c *Controller) Me(ctx contractshttp.Context) contractshttp.Response {
	o, err := c.service.Operator(ctx.Context(), OperatorID(ctx))
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"operator": contractshttp.Json{"id": o.ID, "email": o.Email}})
}

// RequireOperator refuses anything but an operator's token (401).
type RequireOperator struct{}

func (RequireOperator) Signature() string { return "moderation.require_operator" }

func (RequireOperator) Handle(ctx contractshttp.Context) {
	token, ok := strings.CutPrefix(ctx.Request().Header("Authorization"), "Bearer ")
	if !ok || token == "" {
		_ = unauthorized(ctx, "operators only").Abort()
		return
	}
	payload, err := facades.Auth(ctx).Guard(OperatorGuard).Parse(token)
	if err != nil {
		_ = unauthorized(ctx, "operators only").Abort()
		return
	}
	id, err := strconv.ParseUint(payload.Key, 10, 64)
	if err != nil {
		_ = unauthorized(ctx, "operators only").Abort()
		return
	}
	ctx.WithValue(operatorKey{}, id)
	ctx.Request().Next()
}

func failure(ctx contractshttp.Context, err error) contractshttp.Response {
	switch {
	case errors.Is(err, app.ErrBadCredentials):
		return unauthorized(ctx, err.Error())
	case errors.Is(err, app.ErrOperatorNotFound):
		return unauthorized(ctx, "operators only")
	case errors.Is(err, domain.ErrInvalidOperatorEmail), errors.Is(err, domain.ErrWeakOperatorPassword):
		return ctx.Response().Json(contractshttp.StatusUnprocessableEntity, contractshttp.Json{"error": err.Error()})
	}
	return serverError(ctx, err)
}

func unauthorized(ctx contractshttp.Context, msg string) contractshttp.AbortableResponse {
	return ctx.Response().Json(contractshttp.StatusUnauthorized, contractshttp.Json{"error": msg})
}

func badRequest(ctx contractshttp.Context) contractshttp.Response {
	return ctx.Response().Json(contractshttp.StatusBadRequest, contractshttp.Json{"error": "the body is not valid JSON"})
}

func serverError(ctx contractshttp.Context, err error) contractshttp.Response {
	facades.Log().WithContext(ctx.Context()).Error(err)
	return ctx.Response().Json(contractshttp.StatusInternalServerError, contractshttp.Json{"error": "something went wrong"})
}
