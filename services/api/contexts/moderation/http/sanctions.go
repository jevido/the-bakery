package http

import (
	"errors"
	"strings"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

type sanctionJSON struct {
	ID           uint64     `json:"id"`
	TargetKind   string     `json:"target_kind"`
	TargetID     uint64     `json:"target_id"`
	Kind         string     `json:"kind"`
	Reason       string     `json:"reason"`
	Until        *time.Time `json:"until"`
	ByOperatorID uint64     `json:"by_operator_id"`
	LiftedAt     *time.Time `json:"lifted_at"`
	CreatedAt    time.Time  `json:"created_at"`
	Active       bool       `json:"active"`
}

func sanctionToJSON(s domain.Sanction) sanctionJSON {
	return sanctionJSON{ID: s.ID, TargetKind: s.TargetKind, TargetID: s.TargetID, Kind: s.Kind, Reason: s.Reason, Until: s.Until,
		ByOperatorID: s.ByOperatorID, LiftedAt: s.LiftedAt, CreatedAt: s.CreatedAt, Active: s.ActiveAt(time.Now())}
}

// parseTarget reads "member:12" or "guild:3".
func parseTarget(v string) (string, uint64, bool) {
	kind, id, ok := strings.Cut(v, ":")
	if !ok {
		return "", 0, false
	}
	n := uintString(id)
	return kind, n, n != 0 && (kind == domain.TargetMember || kind == domain.TargetGuild)
}

// ListSanctions lists a target's sanctions, newest first: ?target=member:12.
func (c *Controller) ListSanctions(ctx contractshttp.Context) contractshttp.Response {
	kind, id, ok := parseTarget(ctx.Request().Query("target"))
	if !ok {
		return invalid(ctx, "target", "target is member:<id> or guild:<id>")
	}
	ss, err := c.service.SanctionsOf(ctx.Context(), kind, id)
	if err != nil {
		return serverError(ctx, err)
	}
	out := make([]sanctionJSON, len(ss))
	for i, s := range ss {
		out[i] = sanctionToJSON(s)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"sanctions": out})
}

type sanctionRequest struct {
	Target string     `json:"target"`
	Kind   string     `json:"kind"`
	Reason string     `json:"reason"`
	Until  *time.Time `json:"until"`
}

// Sanction suspends or bans a member or a guild.
func (c *Controller) Sanction(ctx contractshttp.Context) contractshttp.Response {
	var req sanctionRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	kind, id, ok := parseTarget(req.Target)
	if !ok {
		return invalid(ctx, "target", "target is member:<id> or guild:<id>")
	}
	s, err := c.service.Sanction(ctx.Context(), OperatorID(ctx), kind, id, req.Kind, req.Reason, req.Until)
	if err != nil {
		return sanctionFailure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"sanction": sanctionToJSON(s)})
}

type liftRequest struct {
	Reason string `json:"reason"`
}

// LiftSanction ends a sanction early.
func (c *Controller) LiftSanction(ctx contractshttp.Context) contractshttp.Response {
	id := uintString(ctx.Request().Route("sanction"))
	var req liftRequest
	_ = ctx.Request().Bind(&req)
	s, err := c.service.LiftSanction(ctx.Context(), OperatorID(ctx), id, req.Reason)
	if err != nil {
		return sanctionFailure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"sanction": sanctionToJSON(s)})
}

func sanctionFailure(ctx contractshttp.Context, err error) contractshttp.Response {
	switch {
	case errors.Is(err, app.ErrTargetNotFound), errors.Is(err, app.ErrSanctionNotFound):
		return ctx.Response().Json(contractshttp.StatusNotFound, contractshttp.Json{"error": err.Error()})
	case errors.Is(err, domain.ErrAlreadySanctioned), errors.Is(err, domain.ErrSanctionLifted):
		return ctx.Response().Json(contractshttp.StatusConflict, contractshttp.Json{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidReason):
		return invalid(ctx, "reason", err.Error())
	case errors.Is(err, domain.ErrSuspensionNeedsUntil), errors.Is(err, domain.ErrBanHasNoUntil):
		return invalid(ctx, "until", err.Error())
	case errors.Is(err, domain.ErrInvalidSanctionKind):
		return invalid(ctx, "kind", err.Error())
	}
	return serverError(ctx, err)
}

func invalid(ctx contractshttp.Context, field, msg string) contractshttp.Response {
	return ctx.Response().Json(contractshttp.StatusUnprocessableEntity, contractshttp.Json{"error": msg, "field": field})
}

func uintString(s string) uint64 {
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + uint64(c-'0')
	}
	return n
}
