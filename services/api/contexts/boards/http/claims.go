package http

import (
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

type claimJSON struct {
	ID        uint64    `json:"id"`
	AgentID   uint64    `json:"agent_id"`
	MemberID  uint64    `json:"member_id"`
	MachineID string    `json:"machine_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func claimToJSON(c domain.Claim) claimJSON {
	return claimJSON{ID: c.ID, AgentID: c.AgentID, MemberID: c.MemberID, MachineID: c.MachineID, ExpiresAt: c.ExpiresAt}
}

type claimRequest struct {
	AgentID   uint64 `json:"agent_id"`
	MachineID string `json:"machine_id"`
}

// ClaimTask gives the task to one of the member's agents on one machine.
func (c *Controller) ClaimTask(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	var req claimRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	cl, err := c.service.ClaimTask(ctx.Context(), taskID, c.me(ctx), req.AgentID, req.MachineID)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"claim": claimToJSON(cl)})
}

// HeartbeatClaim keeps a claim for another two minutes.
func (c *Controller) HeartbeatClaim(ctx contractshttp.Context) contractshttp.Response {
	claimID, ok := routeID(ctx, "claim")
	if !ok {
		return notFound(ctx)
	}
	cl, err := c.service.HeartbeatClaim(ctx.Context(), claimID, c.me(ctx))
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"claim": claimToJSON(cl)})
}

// ReleaseClaim lets the task go.
func (c *Controller) ReleaseClaim(ctx contractshttp.Context) contractshttp.Response {
	claimID, ok := routeID(ctx, "claim")
	if !ok {
		return notFound(ctx)
	}
	if err := c.service.ReleaseClaim(ctx.Context(), claimID, c.me(ctx)); err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().NoContent()
}
