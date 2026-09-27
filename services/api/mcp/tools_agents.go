package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jevido/the-bakery/services/api/contexts/agents"
)

type skillOut struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type agentOut struct {
	ID             uint64         `json:"id" jsonschema:"the agent id, for get_agent"`
	Slug           string         `json:"slug"`
	Name           string         `json:"name"`
	Title          string         `json:"title"`
	Traits         []string       `json:"traits"`
	Model          string         `json:"model"`
	WorkPriorities map[string]int `json:"work_priorities" jsonschema:"work type key to 1 (first) … 4 (last); a missing key is off"`
	Skills         []skillOut     `json:"skills" jsonschema:"the agent's skills by name and description; file contents are not shown"`
	OwnerName      string         `json:"owner_name,omitempty" jsonschema:"who owns it, for an agent shared with a guild"`
	Recruited      bool           `json:"recruited,omitempty" jsonschema:"a copy recruited from another member's agent"`
}

type agentDetailOut struct {
	ID             uint64         `json:"id"`
	Slug           string         `json:"slug"`
	Name           string         `json:"name"`
	Title          string         `json:"title"`
	Backstory      string         `json:"backstory"`
	Traits         []string       `json:"traits"`
	Model          string         `json:"model"`
	PermissionMode string         `json:"permission_mode"`
	AllowedTools   []string       `json:"allowed_tools"`
	WorkPriorities map[string]int `json:"work_priorities" jsonschema:"work type key to 1 (first) … 4 (last); a missing key is off"`
	Skills         []skillOut     `json:"skills"`
	OwnerName      string         `json:"owner_name,omitempty"`
	Recruited      bool           `json:"recruited,omitempty"`
	Revision       int            `json:"revision"`
}

// nonNil makes empty lists and maps come out as [] and {}: the SDK checks
// tool output against its schema, where null is not a list or an object.
func nonNil(a agents.Agent) agents.Agent {
	if a.Traits == nil {
		a.Traits = []string{}
	}
	if a.AllowedTools == nil {
		a.AllowedTools = []string{}
	}
	if a.WorkPriorities == nil {
		a.WorkPriorities = map[string]int{}
	}
	return a
}

func agentOutOf(a agents.Agent) agentOut {
	a = nonNil(a)
	out := agentOut{
		ID: a.ID, Slug: a.Slug, Name: a.Name, Title: a.Title, Traits: a.Traits, Model: a.Model,
		WorkPriorities: a.WorkPriorities, Skills: []skillOut{}, OwnerName: a.OwnerName, Recruited: a.Recruited,
	}
	for _, s := range a.Skills {
		out.Skills = append(out.Skills, skillOut{Name: s.Name, Description: s.Description})
	}
	return out
}

type listAgentsIn struct {
	GuildID uint64 `json:"guild_id,omitempty" jsonschema:"list the agents shared with this guild instead of your own"`
}

type listAgentsOut struct {
	Agents []agentOut `json:"agents"`
}

type getAgentIn struct {
	AgentID uint64 `json:"agent_id" jsonschema:"an agent id from list_agents"`
}

type getAgentOut struct {
	Agent agentDetailOut `json:"agent"`
}

func addAgentTools(s *sdk.Server) {
	sdk.AddTool(s, &sdk.Tool{
		Name:  "list_agents",
		Title: "List agents",
		Description: "Lists your agents (configured Claude workers: name, title, traits, model, work priorities, skills), " +
			"or, with guild_id, the agents members have shared with that guild.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in listAgentsIn) (*sdk.CallToolResult, listAgentsOut, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, listAgentsOut{}, err
		}
		var as []agents.Agent
		if in.GuildID != 0 {
			as, err = agents.ListGuildAgents(ctx, in.GuildID, me)
		} else {
			as, err = agents.ListAgents(ctx, me)
		}
		if err != nil {
			r, _ := failed(err)
			return r, listAgentsOut{}, nil
		}
		out := listAgentsOut{Agents: []agentOut{}}
		for _, a := range as {
			out.Agents = append(out.Agents, agentOutOf(a))
		}
		return nil, out, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name:        "get_agent",
		Title:       "Get an agent",
		Description: "Returns one agent in full (backstory, permission mode, allowed tools, skills): one of yours, or one shared with a guild you are in.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in getAgentIn) (*sdk.CallToolResult, getAgentOut, error) {
		me, err := memberID(ctx)
		if err != nil {
			return nil, getAgentOut{}, err
		}
		a, err := agents.GetAgent(ctx, in.AgentID, me)
		if err != nil {
			// Even an error's output is checked against the schema, where a
			// nil map is not an object.
			r, _ := failed(err)
			return r, getAgentOut{Agent: agentDetailOut{WorkPriorities: map[string]int{}}}, nil
		}
		short := agentOutOf(a)
		a = nonNil(a)
		return nil, getAgentOut{Agent: agentDetailOut{
			ID: a.ID, Slug: a.Slug, Name: a.Name, Title: a.Title, Backstory: a.Backstory, Traits: a.Traits,
			Model: a.Model, PermissionMode: a.PermissionMode, AllowedTools: a.AllowedTools,
			WorkPriorities: a.WorkPriorities, Skills: short.Skills, OwnerName: a.OwnerName,
			Recruited: a.Recruited, Revision: a.Revision,
		}}, nil
	})
}
