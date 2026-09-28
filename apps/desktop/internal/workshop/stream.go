package workshop

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// RunEvent is one thing that happened in a run, as the run panel shows it.
// Kind says which fields are set.
type RunEvent struct {
	// Kind is "init", "text", "tool_call", "tool_result", "result",
	// "note" (something the workshop itself says: a missing skill, an exit),
	// or "usage" (how full the context is; the panel does not show it).
	Kind string `json:"kind"`
	// Seq numbers a run's events from 1, so a panel that loads the past and
	// listens for the new can drop the ones it has twice.
	Seq int `json:"seq"`

	// init
	Model          string      `json:"model,omitempty"`
	PermissionMode string      `json:"permission_mode,omitempty"`
	Skills         []string    `json:"skills,omitempty"`
	MCPServers     []MCPServer `json:"mcp_servers,omitempty"`

	// text, note, and the final text of a result
	Text string `json:"text,omitempty"`

	// tool_call and tool_result
	ToolID string `json:"tool_id,omitempty"`
	Tool   string `json:"tool,omitempty"`
	// Input is the tool call's input as JSON, cut to toolTextMax.
	Input   string `json:"input,omitempty"`
	IsError bool   `json:"is_error,omitempty"`

	// usage: the tokens the turn's context held.
	ContextTokens int `json:"context_tokens,omitempty"`

	// result
	Subtype    string  `json:"subtype,omitempty"`
	CostUSD    float64 `json:"cost_usd,omitempty"`
	Turns      int     `json:"turns,omitempty"`
	DurationMS int64   `json:"duration_ms,omitempty"`
}

// MCPServer is one MCP server a run has, and whether it connected.
type MCPServer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// toolTextMax is how much of a tool's input or result the panel gets.
const toolTextMax = 4096

// streamLine is the part of a stream-json line the decoder reads. The shapes
// were checked against Claude CLI 2.1.280; anything else is ignored.
type streamLine struct {
	Type           string      `json:"type"`
	Subtype        string      `json:"subtype"`
	Model          string      `json:"model"`
	PermissionMode string      `json:"permissionMode"`
	Skills         []string    `json:"skills"`
	MCPServers     []MCPServer `json:"mcp_servers"`
	Message        struct {
		Content json.RawMessage `json:"content"`
		Usage   *struct {
			InputTokens              int `json:"input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
	IsError      bool    `json:"is_error"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	NumTurns     int     `json:"num_turns"`
	DurationMS   int64   `json:"duration_ms"`
	Result       string  `json:"result"`
}

type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

// DecodeLine turns one line of Claude's stream-json output into run events.
// A line that is not JSON (a version manager's banner, say) or of a type
// the panel does not show gives none; ok is false only for a line that is
// not JSON at all.
func DecodeLine(line []byte) (events []RunEvent, ok bool) {
	var l streamLine
	if json.Unmarshal(line, &l) != nil {
		return nil, false
	}
	switch l.Type {
	case "system":
		if l.Subtype == "init" {
			return []RunEvent{{Kind: "init", Model: l.Model, PermissionMode: l.PermissionMode, Skills: l.Skills, MCPServers: l.MCPServers}}, true
		}
	case "assistant":
		var blocks []contentBlock
		if json.Unmarshal(l.Message.Content, &blocks) != nil {
			return nil, true
		}
		for _, b := range blocks {
			switch b.Type {
			case "text":
				if strings.TrimSpace(b.Text) != "" {
					events = append(events, RunEvent{Kind: "text", Text: b.Text})
				}
			case "tool_use":
				events = append(events, RunEvent{Kind: "tool_call", ToolID: b.ID, Tool: b.Name, Input: cut(string(b.Input))})
			}
		}
		if u := l.Message.Usage; u != nil {
			if n := u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens; n > 0 {
				events = append(events, RunEvent{Kind: "usage", ContextTokens: n})
			}
		}
	case "user":
		var blocks []contentBlock
		if json.Unmarshal(l.Message.Content, &blocks) != nil {
			return nil, true
		}
		for _, b := range blocks {
			if b.Type == "tool_result" {
				events = append(events, RunEvent{Kind: "tool_result", ToolID: b.ToolUseID, Text: cut(resultText(b.Content)), IsError: b.IsError})
			}
		}
	case "result":
		return []RunEvent{{
			Kind: "result", Subtype: l.Subtype, IsError: l.IsError, CostUSD: l.TotalCostUSD,
			Turns: l.NumTurns, DurationMS: l.DurationMS, Text: l.Result,
		}}, true
	}
	return events, true
}

// resultText is a tool result's content: a string, or text blocks.
func resultText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var blocks []contentBlock
	if json.Unmarshal(raw, &blocks) == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return string(raw)
}

// cut keeps at most toolTextMax bytes, on a character boundary.
func cut(s string) string {
	if len(s) <= toolTextMax {
		return s
	}
	s = s[:toolTextMax]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s + "…"
}

// StateAfter is what an agent is doing after a run event, for the colony
// view: "thinking" while Claude writes or reads, "editing" while it changes
// files, "running" while a command runs; "" keeps the state it had.
func StateAfter(current string, ev RunEvent) string {
	switch ev.Kind {
	case "init", "text", "tool_result":
		return "thinking"
	case "tool_call":
		switch ev.Tool {
		case "Edit", "Write", "MultiEdit", "NotebookEdit":
			return "editing"
		case "Bash":
			return "running"
		}
		return "thinking"
	}
	return ""
}
