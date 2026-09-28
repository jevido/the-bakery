package http

import (
	"errors"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

type reportJSON struct {
	ID         uint64     `json:"id"`
	ByMemberID uint64     `json:"by_member_id"`
	TargetKind string     `json:"target_kind"`
	TargetID   uint64     `json:"target_id"`
	Reason     string     `json:"reason"`
	Status     string     `json:"status"`
	HandledBy  uint64     `json:"handled_by,omitempty"`
	HandledAt  *time.Time `json:"handled_at"`
	SanctionID uint64     `json:"sanction_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func reportToJSON(r domain.Report) reportJSON {
	return reportJSON{ID: r.ID, ByMemberID: r.ByMemberID, TargetKind: r.TargetKind, TargetID: r.TargetID, Reason: r.Reason,
		Status: r.Status, HandledBy: r.HandledBy, HandledAt: r.HandledAt, SanctionID: r.SanctionID, CreatedAt: r.CreatedAt}
}

// Members controller: filing a report, with a member's sign-in.
type MemberController struct {
	service  *app.Service
	memberID func(contractshttp.Context) (uint64, bool)
}

func NewMemberController(service *app.Service, memberID func(contractshttp.Context) (uint64, bool)) *MemberController {
	return &MemberController{service: service, memberID: memberID}
}

type fileReportRequest struct {
	TargetKind string `json:"target_kind"`
	TargetID   uint64 `json:"target_id"`
	Reason     string `json:"reason"`
}

// FileReport lets a member report a member or a guild to the operators.
func (c *MemberController) FileReport(ctx contractshttp.Context) contractshttp.Response {
	var req fileReportRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	me, _ := c.memberID(ctx)
	r, err := c.service.FileReport(ctx.Context(), me, req.TargetKind, req.TargetID, req.Reason)
	if err != nil {
		return reportFailure(ctx, err)
	}
	// A member sees their report, not how it will be handled.
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"report": contractshttp.Json{
		"id": r.ID, "target_kind": r.TargetKind, "target_id": r.TargetID, "created_at": r.CreatedAt,
	}})
}

// ListReports lists reports for the console: ?status=open (default all).
func (c *Controller) ListReports(ctx contractshttp.Context) contractshttp.Response {
	rs, err := c.service.ReportsWithStatus(ctx.Context(), ctx.Request().Query("status"))
	if err != nil {
		return serverError(ctx, err)
	}
	out := make([]reportJSON, len(rs))
	for i, r := range rs {
		out[i] = reportToJSON(r)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"reports": out})
}

// DismissReport closes a report without action.
func (c *Controller) DismissReport(ctx contractshttp.Context) contractshttp.Response {
	r, err := c.service.DismissReport(ctx.Context(), OperatorID(ctx), uintString(ctx.Request().Route("report")))
	if err != nil {
		return reportFailure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"report": reportToJSON(r)})
}

type actionRequest struct {
	SanctionID uint64 `json:"sanction_id"`
}

// ActionReport closes a report with the sanction it led to.
func (c *Controller) ActionReport(ctx contractshttp.Context) contractshttp.Response {
	var req actionRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	r, err := c.service.ActionReport(ctx.Context(), OperatorID(ctx), uintString(ctx.Request().Route("report")), req.SanctionID)
	if err != nil {
		return reportFailure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"report": reportToJSON(r)})
}

func reportFailure(ctx contractshttp.Context, err error) contractshttp.Response {
	switch {
	case errors.Is(err, domain.ErrTooManyReports):
		return ctx.Response().Json(contractshttp.StatusTooManyRequests, contractshttp.Json{"error": err.Error()})
	case errors.Is(err, app.ErrReportNotFound), errors.Is(err, app.ErrTargetNotFound), errors.Is(err, app.ErrSanctionNotFound):
		return ctx.Response().Json(contractshttp.StatusNotFound, contractshttp.Json{"error": err.Error()})
	case errors.Is(err, domain.ErrReportHandled):
		return ctx.Response().Json(contractshttp.StatusConflict, contractshttp.Json{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidReason):
		return invalid(ctx, "reason", err.Error())
	case errors.Is(err, domain.ErrInvalidReportTarget), errors.Is(err, domain.ErrReportSelf):
		return invalid(ctx, "target_id", err.Error())
	}
	return serverError(ctx, err)
}
