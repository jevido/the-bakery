// Package http exposes agents over HTTP, behind identity's RequireMember.
package http

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	nethttp "net/http"
	"strconv"
	"strings"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/agents/app"
	"github.com/jevido/the-bakery/services/api/contexts/agents/domain"
)

// maxBody bounds an agent write: a skillset is at most 2 MB, plus JSON.
// Goravel's body_limit only covers multipart forms.
const maxBody = 3 << 20

type MemberID func(ctx contractshttp.Context) (uint64, bool)

type Controller struct {
	service  *app.Service
	memberID MemberID
}

func NewController(service *app.Service, memberID MemberID) *Controller {
	return &Controller{service: service, memberID: memberID}
}

type fileJSON struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	SHA256  string `json:"sha256,omitempty"`
}

type skillJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// agentJSON is an agent; files are left out of lists.
type agentJSON struct {
	ID             uint64         `json:"id"`
	Slug           string         `json:"slug"`
	Name           string         `json:"name"`
	Title          string         `json:"title"`
	Backstory      string         `json:"backstory"`
	Traits         []string       `json:"traits"`
	Model          string         `json:"model"`
	PermissionMode string         `json:"permission_mode"`
	AllowedTools   []string       `json:"allowed_tools"`
	PortraitSeed   string         `json:"portrait_seed"`
	WorkPriorities map[string]int `json:"work_priorities"`
	Skills         []skillJSON    `json:"skills"`
	Revision       int            `json:"revision"`
	OriginAgentID  *uint64        `json:"origin_agent_id"`
	OriginRevision *int           `json:"origin_revision"`
	UpdatedAt      time.Time      `json:"updated_at"`
	Deleted        bool           `json:"deleted"`
	Files          []fileJSON     `json:"files,omitempty"`
}

func agentToJSON(a domain.Agent, withFiles bool) agentJSON {
	out := agentJSON{
		ID: a.ID, Slug: a.Slug, Name: a.Name, Title: a.Title, Backstory: a.Backstory, Traits: a.Traits,
		Model: a.Model, PermissionMode: a.PermissionMode, AllowedTools: a.AllowedTools, PortraitSeed: a.PortraitSeed,
		WorkPriorities: a.WorkPriorities, Skills: []skillJSON{}, Revision: a.Revision, OriginAgentID: a.OriginAgentID,
		OriginRevision: a.OriginRevision, UpdatedAt: a.UpdatedAt, Deleted: a.DeletedAt != nil,
	}
	if out.Traits == nil {
		out.Traits = []string{}
	}
	if out.AllowedTools == nil {
		out.AllowedTools = []string{}
	}
	if out.WorkPriorities == nil {
		out.WorkPriorities = map[string]int{}
	}
	if withFiles {
		for _, s := range a.Skills() {
			out.Skills = append(out.Skills, skillJSON{Name: s.Name, Description: s.Description})
		}
		out.Files = []fileJSON{}
		for _, f := range a.Files {
			sum := sha256.Sum256([]byte(f.Content))
			out.Files = append(out.Files, fileJSON{Path: f.Path, Content: f.Content, SHA256: hex.EncodeToString(sum[:])})
		}
	}
	return out
}

// agentRequest is a whole agent as a client writes it.
type agentRequest struct {
	Slug           string         `json:"slug"`
	Name           string         `json:"name"`
	Title          string         `json:"title"`
	Backstory      string         `json:"backstory"`
	Traits         []string       `json:"traits"`
	Model          string         `json:"model"`
	PermissionMode string         `json:"permission_mode"`
	AllowedTools   []string       `json:"allowed_tools"`
	PortraitSeed   string         `json:"portrait_seed"`
	WorkPriorities map[string]int `json:"work_priorities"`
	Files          []fileJSON     `json:"files"`
}

func (r agentRequest) profile() domain.Profile {
	p := domain.Profile{
		Name: r.Name, Title: r.Title, Backstory: r.Backstory, Traits: r.Traits, Model: r.Model,
		PermissionMode: r.PermissionMode, AllowedTools: r.AllowedTools, PortraitSeed: r.PortraitSeed,
		WorkPriorities: r.WorkPriorities, Files: []domain.File{},
	}
	for _, f := range r.Files {
		p.Files = append(p.Files, domain.File{Path: f.Path, Content: f.Content})
	}
	return p
}

// bind decodes the body, refusing more than maxBody.
func bind(ctx contractshttp.Context, v any) (contractshttp.Response, bool) {
	req := ctx.Request().Origin()
	req.Body = nethttp.MaxBytesReader(ctx.Response().Writer(), req.Body, maxBody)
	if err := json.NewDecoder(req.Body).Decode(v); err != nil {
		var tooBig *nethttp.MaxBytesError
		if errors.As(err, &tooBig) {
			return ctx.Response().Json(contractshttp.StatusRequestEntityTooLarge, contractshttp.Json{"error": "an agent is at most 3 MB"}), false
		}
		return ctx.Response().Json(contractshttp.StatusBadRequest, contractshttp.Json{"error": "request body must be JSON"}), false
	}
	return nil, true
}

// ifMatch reads the revision a write is based on from If-Match (quotes
// allowed, as for an ETag).
func ifMatch(ctx contractshttp.Context) (int, bool) {
	v := strings.Trim(strings.TrimPrefix(strings.TrimSpace(ctx.Request().Header("If-Match", "")), "W/"), `"`)
	n, err := strconv.Atoi(v)
	return n, err == nil && n > 0
}

func (c *Controller) me(ctx contractshttp.Context) uint64 {
	id, _ := c.memberID(ctx)
	return id
}

func agentID(ctx contractshttp.Context) (uint64, bool) {
	id, err := strconv.ParseUint(ctx.Request().Route("agent"), 10, 64)
	return id, err == nil
}

// ListAgents: my agents without files; ?since=<RFC3339> for sync, with
// deleted ones.
func (c *Controller) ListAgents(ctx contractshttp.Context) contractshttp.Response {
	var since *time.Time
	if v := ctx.Request().Query("since", ""); v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return ctx.Response().Json(contractshttp.StatusUnprocessableEntity, contractshttp.Json{"error": "since must be an RFC 3339 time", "field": "since"})
		}
		since = &t
	}
	agents, err := c.service.ListAgents(ctx.Context(), c.me(ctx), since)
	if err != nil {
		return failure(ctx, err, domain.Agent{})
	}
	out := make([]agentJSON, len(agents))
	for i, a := range agents {
		out[i] = agentToJSON(a, false)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"agents": out})
}

func (c *Controller) GetAgent(ctx contractshttp.Context) contractshttp.Response {
	id, ok := agentID(ctx)
	if !ok {
		return notFound(ctx)
	}
	a, err := c.service.GetAgent(ctx.Context(), id, c.me(ctx))
	if err != nil {
		return failure(ctx, err, a)
	}
	guilds, err := c.service.SharedWith(ctx.Context(), id, c.me(ctx))
	if err != nil {
		return failure(ctx, err, a)
	}
	if guilds == nil {
		guilds = []uint64{}
	}
	return ctx.Response().Success().Json(contractshttp.Json{"agent": agentToJSON(a, true), "shared_with": guilds})
}

func (c *Controller) CreateAgent(ctx contractshttp.Context) contractshttp.Response {
	var req agentRequest
	if res, ok := bind(ctx, &req); !ok {
		return res
	}
	a, err := c.service.CreateAgent(ctx.Context(), c.me(ctx), req.Slug, req.profile())
	if err != nil {
		return failure(ctx, err, a)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"agent": agentToJSON(a, true)})
}

// ReviseAgent replaces the whole agent; If-Match names the revision it is
// based on.
func (c *Controller) ReviseAgent(ctx contractshttp.Context) contractshttp.Response {
	id, ok := agentID(ctx)
	if !ok {
		return notFound(ctx)
	}
	basedOn, ok := ifMatch(ctx)
	if !ok {
		return ctx.Response().Json(contractshttp.StatusPreconditionRequired, contractshttp.Json{"error": "If-Match must name the revision this change is based on"})
	}
	var req agentRequest
	if res, ok := bind(ctx, &req); !ok {
		return res
	}
	a, err := c.service.ReviseAgent(ctx.Context(), id, c.me(ctx), basedOn, req.profile())
	if err != nil {
		return failure(ctx, err, a)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"agent": agentToJSON(a, true)})
}

func (c *Controller) DeleteAgent(ctx contractshttp.Context) contractshttp.Response {
	id, ok := agentID(ctx)
	if !ok {
		return notFound(ctx)
	}
	basedOn, ok := ifMatch(ctx)
	if !ok {
		return ctx.Response().Json(contractshttp.StatusPreconditionRequired, contractshttp.Json{"error": "If-Match must name the revision this change is based on"})
	}
	a, err := c.service.DeleteAgent(ctx.Context(), id, c.me(ctx), basedOn)
	if err != nil {
		return failure(ctx, err, a)
	}
	return ctx.Response().NoContent()
}

type traitJSON struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Instruction string   `json:"instruction"`
	Conflicts   []string `json:"conflicts_with"`
}

// ListTraits returns the fixed trait list.
func (c *Controller) ListTraits(ctx contractshttp.Context) contractshttp.Response {
	out := make([]traitJSON, len(domain.Traits))
	for i, t := range domain.Traits {
		conflicts := domain.ConflictsOf(t.Key)
		if conflicts == nil {
			conflicts = []string{}
		}
		out[i] = traitJSON{Key: t.Key, Label: t.Label, Description: t.Description, Instruction: t.Instruction, Conflicts: conflicts}
	}
	return ctx.Response().Success().Json(contractshttp.Json{"traits": out, "permission_modes": domain.PermissionModes})
}

// failure maps errors to answers. A stale write answers 409 with the
// agent as it is now, so the client can show both.
func failure(ctx contractshttp.Context, err error, current domain.Agent) contractshttp.Response {
	var skillset *domain.SkillsetError
	switch {
	case errors.Is(err, domain.ErrStale):
		body := contractshttp.Json{"error": err.Error()}
		if current.ID != 0 {
			body["agent"] = agentToJSON(current, true)
		}
		return ctx.Response().Json(contractshttp.StatusConflict, body)
	case errors.Is(err, app.ErrAgentNotFound), errors.Is(err, domain.ErrDeleted):
		return notFound(ctx)
	case errors.Is(err, app.ErrNotMember):
		return ctx.Response().Json(contractshttp.StatusForbidden, contractshttp.Json{"error": err.Error()})
	case errors.Is(err, app.ErrNotShared):
		return ctx.Response().Json(contractshttp.StatusNotFound, contractshttp.Json{"error": err.Error()})
	case errors.As(err, &skillset):
		return ctx.Response().Json(contractshttp.StatusUnprocessableEntity, contractshttp.Json{"error": err.Error(), "field": "files", "path": skillset.Path})
	case errors.Is(err, app.ErrSlugTaken), errors.Is(err, domain.ErrInvalidSlug):
		return invalid(ctx, err, "slug")
	case errors.Is(err, domain.ErrInvalidName):
		return invalid(ctx, err, "name")
	case errors.Is(err, domain.ErrInvalidTitle):
		return invalid(ctx, err, "title")
	case errors.Is(err, domain.ErrInvalidBackstory):
		return invalid(ctx, err, "backstory")
	case errors.Is(err, domain.ErrUnknownTrait), errors.Is(err, domain.ErrConflictingTraits):
		return invalid(ctx, err, "traits")
	case errors.Is(err, domain.ErrInvalidModel):
		return invalid(ctx, err, "model")
	case errors.Is(err, domain.ErrInvalidPermissionMode):
		return invalid(ctx, err, "permission_mode")
	case errors.Is(err, domain.ErrTooManyTools), errors.Is(err, domain.ErrInvalidTool):
		return invalid(ctx, err, "allowed_tools")
	case errors.Is(err, domain.ErrInvalidPriority):
		return invalid(ctx, err, "work_priorities")
	}
	facades.Log().WithContext(ctx.Context()).Error(err)
	return ctx.Response().Json(contractshttp.StatusInternalServerError, contractshttp.Json{"error": "something went wrong"})
}

func invalid(ctx contractshttp.Context, err error, field string) contractshttp.Response {
	return ctx.Response().Json(contractshttp.StatusUnprocessableEntity, contractshttp.Json{"error": err.Error(), "field": field})
}

func notFound(ctx contractshttp.Context) contractshttp.Response {
	return ctx.Response().Json(contractshttp.StatusNotFound, contractshttp.Json{"error": "agent not found"})
}
