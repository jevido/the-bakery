package http

import (
	"strconv"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/agents/app"
	"github.com/jevido/the-bakery/services/api/contexts/agents/domain"
)

func guildID(ctx contractshttp.Context) (uint64, bool) {
	id, err := strconv.ParseUint(ctx.Request().Route("guild"), 10, 64)
	return id, err == nil
}

// sharedAgentJSON is an agent as a guild sees it: no files, its skills by
// name and description, and whose it is.
type sharedAgentJSON struct {
	agentJSON
	OwnerID   uint64 `json:"owner_id"`
	OwnerName string `json:"owner_name"`
}

func sharedToJSON(a app.SharedAgent) sharedAgentJSON {
	out := sharedAgentJSON{agentJSON: agentToJSON(a.Agent, false), OwnerID: a.OwnerID, OwnerName: a.OwnerName}
	for _, s := range a.Skills() {
		out.Skills = append(out.Skills, skillJSON{Name: s.Name, Description: s.Description})
	}
	return out
}

func (c *Controller) ShareAgent(ctx contractshttp.Context) contractshttp.Response {
	id, ok := agentID(ctx)
	guild, ok2 := guildID(ctx)
	if !ok || !ok2 {
		return notFound(ctx)
	}
	if err := c.service.ShareAgent(ctx.Context(), id, c.me(ctx), guild); err != nil {
		return failure(ctx, err, domain.Agent{})
	}
	return ctx.Response().NoContent()
}

func (c *Controller) UnshareAgent(ctx contractshttp.Context) contractshttp.Response {
	id, ok := agentID(ctx)
	guild, ok2 := guildID(ctx)
	if !ok || !ok2 {
		return notFound(ctx)
	}
	if err := c.service.UnshareAgent(ctx.Context(), id, c.me(ctx), guild); err != nil {
		return failure(ctx, err, domain.Agent{})
	}
	return ctx.Response().NoContent()
}

// ListGuildAgents: the agents shared with a guild, for its members.
func (c *Controller) ListGuildAgents(ctx contractshttp.Context) contractshttp.Response {
	guild, ok := guildID(ctx)
	if !ok {
		return notFound(ctx)
	}
	agents, err := c.service.ListGuildAgents(ctx.Context(), guild, c.me(ctx))
	if err != nil {
		return failure(ctx, err, domain.Agent{})
	}
	out := make([]sharedAgentJSON, len(agents))
	for i, a := range agents {
		out[i] = sharedToJSON(a)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"agents": out})
}

type recruitRequest struct {
	AgentID uint64 `json:"agent_id"`
	GuildID uint64 `json:"guild_id"`
	Slug    string `json:"slug"`
}

func (c *Controller) RecruitAgent(ctx contractshttp.Context) contractshttp.Response {
	var req recruitRequest
	if res, ok := bind(ctx, &req); !ok {
		return res
	}
	a, err := c.service.RecruitAgent(ctx.Context(), c.me(ctx), req.AgentID, req.GuildID, req.Slug)
	if err != nil {
		return failure(ctx, err, domain.Agent{})
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"agent": agentToJSON(a, true)})
}

func (c *Controller) Origin(ctx contractshttp.Context) contractshttp.Response {
	id, ok := agentID(ctx)
	if !ok {
		return notFound(ctx)
	}
	st, err := c.service.Origin(ctx.Context(), id, c.me(ctx))
	if err != nil {
		return failure(ctx, err, domain.Agent{})
	}
	if st.Gone {
		return ctx.Response().Success().Json(contractshttp.Json{"origin_agent_id": st.OriginAgentID, "gone": true})
	}
	return ctx.Response().Success().Json(contractshttp.Json{
		"origin_agent_id": st.OriginAgentID, "gone": false, "origin_revision": st.OriginRevision,
		"current_revision": st.Current, "newer": st.Newer,
	})
}

// PullOrigin takes the origin's fields and files, keeping my priorities.
func (c *Controller) PullOrigin(ctx contractshttp.Context) contractshttp.Response {
	id, ok := agentID(ctx)
	if !ok {
		return notFound(ctx)
	}
	basedOn, ok := ifMatch(ctx)
	if !ok {
		return ctx.Response().Json(contractshttp.StatusPreconditionRequired, contractshttp.Json{"error": "If-Match must name the revision this change is based on"})
	}
	a, err := c.service.PullOrigin(ctx.Context(), id, c.me(ctx), basedOn)
	if err != nil {
		return failure(ctx, err, a)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"agent": agentToJSON(a, true)})
}
