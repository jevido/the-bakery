package http

import (
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

type runJSON struct {
	ID           uint64     `json:"id"`
	TaskID       uint64     `json:"task_id"`
	MemberID     uint64     `json:"member_id"`
	MemberName   string     `json:"member_name"`
	AgentID      uint64     `json:"agent_id"`
	AgentName    string     `json:"agent_name"`
	Machine      string     `json:"machine"`
	Branch       string     `json:"branch"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	CostUSD      float64    `json:"cost_usd"`
	Turns        int        `json:"turns"`
	Summary      string     `json:"summary"`
	FilesChanged int        `json:"files_changed"`
	Additions    int        `json:"additions"`
	Deletions    int        `json:"deletions"`
}

func runToJSON(r app.RunView) runJSON {
	return runJSON{
		ID: r.ID, TaskID: r.TaskID, MemberID: r.MemberID, MemberName: r.MemberName,
		AgentID: r.AgentID, AgentName: r.AgentName, Machine: r.Machine, Branch: r.Branch,
		Status: string(r.Status), StartedAt: r.StartedAt, EndedAt: r.EndedAt,
		CostUSD: r.CostUSD, Turns: r.Turns, Summary: r.Summary,
		FilesChanged: r.FilesChanged, Additions: r.Additions, Deletions: r.Deletions,
	}
}

// ListRuns returns a task's runs, newest first.
func (c *Controller) ListRuns(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	rs, err := c.service.ListRuns(ctx.Context(), taskID, c.me(ctx), 0)
	if err != nil {
		return failure(ctx, err)
	}
	out := make([]runJSON, len(rs))
	for i, r := range rs {
		out[i] = runToJSON(r)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"runs": out})
}

type startRunRequest struct {
	AgentID   uint64 `json:"agent_id"`
	AgentName string `json:"agent_name"`
	Machine   string `json:"machine"`
	Branch    string `json:"branch"`
}

func (c *Controller) StartRun(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	var req startRunRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	r, err := c.service.StartRun(ctx.Context(), taskID, c.me(ctx), req.AgentID, req.AgentName, req.Machine, req.Branch)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"run": runToJSON(r)})
}

type finishRunRequest struct {
	Status       string  `json:"status"`
	CostUSD      float64 `json:"cost_usd"`
	Turns        int     `json:"turns"`
	Summary      string  `json:"summary"`
	FilesChanged int     `json:"files_changed"`
	Additions    int     `json:"additions"`
	Deletions    int     `json:"deletions"`
}

// FinishRun ends a run; only the member who started it can.
func (c *Controller) FinishRun(ctx contractshttp.Context) contractshttp.Response {
	runID, ok := routeID(ctx, "run")
	if !ok {
		return notFound(ctx)
	}
	var req finishRunRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	r, err := c.service.FinishRun(ctx.Context(), runID, c.me(ctx), domain.RunStatus(req.Status), domain.RunStats{
		CostUSD: req.CostUSD, Turns: req.Turns, Summary: req.Summary,
		FilesChanged: req.FilesChanged, Additions: req.Additions, Deletions: req.Deletions,
	})
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"run": runToJSON(r)})
}
