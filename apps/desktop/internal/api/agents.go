package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// AgentFile is one skillset file, by its path under `.claude/skills`.
type AgentFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	SHA256  string `json:"sha256,omitempty"`
}

// AgentSkill is a skill by name and description.
type AgentSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Agent is an agent as the API returns it. Lists leave Files out.
type Agent struct {
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
	Skills         []AgentSkill   `json:"skills"`
	Revision       int            `json:"revision"`
	OriginAgentID  *uint64        `json:"origin_agent_id"`
	OriginRevision *int           `json:"origin_revision"`
	UpdatedAt      time.Time      `json:"updated_at"`
	Deleted        bool           `json:"deleted"`
	Files          []AgentFile    `json:"files"`
	// OwnerName is set on an agent shared with a guild.
	OwnerName string `json:"owner_name,omitempty"`
}

// AgentWrite is a whole agent as the app sends it.
type AgentWrite struct {
	Slug           string         `json:"slug,omitempty"`
	Name           string         `json:"name"`
	Title          string         `json:"title"`
	Backstory      string         `json:"backstory"`
	Traits         []string       `json:"traits"`
	Model          string         `json:"model"`
	PermissionMode string         `json:"permission_mode"`
	AllowedTools   []string       `json:"allowed_tools"`
	PortraitSeed   string         `json:"portrait_seed"`
	WorkPriorities map[string]int `json:"work_priorities"`
	Files          []AgentFile    `json:"files"`
}

// StaleError is a 409: the agent changed since the revision the write was
// based on. Current is the agent as it is now.
type StaleError struct {
	Current Agent
}

func (e *StaleError) Error() string {
	return fmt.Sprintf("agent %s changed on the server (now at revision %d)", e.Current.Slug, e.Current.Revision)
}

// stale turns a 409 into a *StaleError.
func stale(err error) error {
	var apiErr *Error
	if errors.As(err, &apiErr) && apiErr.Status == http.StatusConflict {
		var body struct {
			Agent Agent `json:"agent"`
		}
		if json.Unmarshal(apiErr.Body, &body) == nil && body.Agent.ID != 0 {
			return &StaleError{Current: body.Agent}
		}
	}
	return err
}

// ListAgents returns my agents without files; with since, every agent
// changed after then, deleted ones included.
func (c *Client) ListAgents(ctx context.Context, token string, since *time.Time) ([]Agent, error) {
	path := "/api/agents"
	if since != nil {
		path += "?since=" + url.QueryEscape(since.UTC().Format(time.RFC3339Nano))
	}
	var res struct {
		Agents []Agent `json:"agents"`
	}
	err := c.do(ctx, http.MethodGet, path, token, nil, &res)
	return res.Agents, err
}

// GetAgent returns one of my agents with its files and the guilds it is
// shared with.
func (c *Client) GetAgent(ctx context.Context, token string, id uint64) (Agent, []uint64, error) {
	var res struct {
		Agent      Agent    `json:"agent"`
		SharedWith []uint64 `json:"shared_with"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/agents/%d", id), token, nil, &res)
	return res.Agent, res.SharedWith, err
}

func (c *Client) CreateAgent(ctx context.Context, token string, a AgentWrite) (Agent, error) {
	var res struct {
		Agent Agent `json:"agent"`
	}
	err := c.do(ctx, http.MethodPost, "/api/agents", token, a, &res)
	return res.Agent, err
}

// ReviseAgent replaces the agent if it is still at basedOn; otherwise it
// returns a *StaleError.
func (c *Client) ReviseAgent(ctx context.Context, token string, id uint64, basedOn int, a AgentWrite) (Agent, error) {
	var res struct {
		Agent Agent `json:"agent"`
	}
	err := c.doWith(ctx, http.MethodPut, fmt.Sprintf("/api/agents/%d", id), token, map[string]string{"If-Match": strconv.Itoa(basedOn)}, a, &res)
	return res.Agent, stale(err)
}

// DeleteAgent deletes the agent if it is still at basedOn.
func (c *Client) DeleteAgent(ctx context.Context, token string, id uint64, basedOn int) error {
	err := c.doWith(ctx, http.MethodDelete, fmt.Sprintf("/api/agents/%d", id), token, map[string]string{"If-Match": strconv.Itoa(basedOn)}, nil, nil)
	return stale(err)
}

// Trait is one of the fixed traits an agent can have.
type Trait struct {
	Key           string   `json:"key"`
	Label         string   `json:"label"`
	Description   string   `json:"description"`
	ConflictsWith []string `json:"conflicts_with"`
}

// Traits returns the trait list and the permission modes agents may use.
func (c *Client) Traits(ctx context.Context, token string) ([]Trait, []string, error) {
	var res struct {
		Traits          []Trait  `json:"traits"`
		PermissionModes []string `json:"permission_modes"`
	}
	err := c.do(ctx, http.MethodGet, "/api/agent-traits", token, nil, &res)
	return res.Traits, res.PermissionModes, err
}

// ShareAgent shares (on) or unshares one of my agents with a guild.
func (c *Client) ShareAgent(ctx context.Context, token string, id, guildID uint64, on bool) error {
	method := http.MethodPut
	if !on {
		method = http.MethodDelete
	}
	return c.do(ctx, method, fmt.Sprintf("/api/agents/%d/shares/%d", id, guildID), token, nil, nil)
}

// GuildAgents lists the agents shared with a guild.
func (c *Client) GuildAgents(ctx context.Context, token string, guildID uint64) ([]Agent, error) {
	var res struct {
		Agents []Agent `json:"agents"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/guilds/%d/agents", guildID), token, nil, &res)
	return res.Agents, err
}

// RecruitAgent copies an agent shared with the guild into my roster.
func (c *Client) RecruitAgent(ctx context.Context, token string, agentID, guildID uint64) (Agent, error) {
	var res struct {
		Agent Agent `json:"agent"`
	}
	err := c.do(ctx, http.MethodPost, "/api/agents/recruit", token, map[string]uint64{"agent_id": agentID, "guild_id": guildID}, &res)
	return res.Agent, err
}

// Origin is how a recruited copy stands against its original.
type Origin struct {
	OriginAgentID   uint64 `json:"origin_agent_id"`
	Gone            bool   `json:"gone"`
	OriginRevision  int    `json:"origin_revision"`
	CurrentRevision int    `json:"current_revision"`
	Newer           bool   `json:"newer"`
}

func (c *Client) AgentOrigin(ctx context.Context, token string, id uint64) (Origin, error) {
	var res Origin
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/agents/%d/origin", id), token, nil, &res)
	return res, err
}

// PullOrigin updates a recruited copy from its original.
func (c *Client) PullOrigin(ctx context.Context, token string, id uint64, basedOn int) (Agent, error) {
	var res struct {
		Agent Agent `json:"agent"`
	}
	err := c.doWith(ctx, http.MethodPost, fmt.Sprintf("/api/agents/%d/pull-origin", id), token, map[string]string{"If-Match": strconv.Itoa(basedOn)}, nil, &res)
	return res.Agent, stale(err)
}
