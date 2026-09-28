package http

import (
	"strconv"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

// RememberIP keeps the client's IP in the request's context, where the
// audit log reads it.
type RememberIP struct{}

func (RememberIP) Signature() string { return "moderation.remember_ip" }

func (RememberIP) Handle(ctx contractshttp.Context) {
	ctx.WithValue(app.IPKey, ctx.Request().Ip())
	ctx.Request().Next()
}

type auditJSON struct {
	ID         uint64         `json:"id"`
	ActorKind  string         `json:"actor_kind"`
	ActorID    uint64         `json:"actor_id"`
	Action     string         `json:"action"`
	TargetKind string         `json:"target_kind"`
	TargetID   uint64         `json:"target_id"`
	Reason     string         `json:"reason"`
	Meta       map[string]any `json:"meta"`
	IP         string         `json:"ip"`
	At         time.Time      `json:"at"`
}

// Audit lists the audit log, newest first: filters actor_kind, actor_id,
// action, target_kind, target_id, from and to (RFC 3339), and before (an
// entry id) and limit for paging.
func (c *Controller) Audit(ctx contractshttp.Context) contractshttp.Response {
	q := ctx.Request()
	f := app.AuditFilter{
		ActorKind: q.Query("actor_kind"), Action: q.Query("action"), TargetKind: q.Query("target_kind"),
		ActorID: uintQuery(ctx, "actor_id"), TargetID: uintQuery(ctx, "target_id"), Before: uintQuery(ctx, "before"),
		Limit: int(uintQuery(ctx, "limit")),
	}
	for key, dst := range map[string]**time.Time{"from": &f.From, "to": &f.To} {
		if v := q.Query(key); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return ctx.Response().Json(contractshttp.StatusUnprocessableEntity, contractshttp.Json{"error": key + " must be an RFC 3339 time", "field": key})
			}
			*dst = &t
		}
	}
	entries, err := c.service.Audit(ctx.Context(), f)
	if err != nil {
		return serverError(ctx, err)
	}
	names, err := c.namesOf(ctx, entries, nil)
	if err != nil {
		return serverError(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"entries": auditToJSON(entries), "names": names})
}

func auditToJSON(entries []domain.AuditEntry) []auditJSON {
	out := make([]auditJSON, len(entries))
	for i, e := range entries {
		out[i] = auditJSON{ID: e.ID, ActorKind: e.ActorKind, ActorID: e.ActorID, Action: e.Action, TargetKind: e.TargetKind,
			TargetID: e.TargetID, Reason: e.Reason, Meta: e.Meta, IP: e.IP, At: e.At}
	}
	return out
}

func uintQuery(ctx contractshttp.Context, key string) uint64 {
	n, _ := strconv.ParseUint(ctx.Request().Query(key), 10, 64)
	return n
}
