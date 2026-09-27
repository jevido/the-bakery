package workshop

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// RunSpec is everything one run needs: the claude arguments, where it runs,
// what it reads on stdin, and the files to write before it starts (keyed by
// absolute path).
type RunSpec struct {
	Args  []string          `json:"args"`
	Dir   string            `json:"dir"`
	Stdin string            `json:"stdin"`
	Files map[string][]byte `json:"-"`
}

// RunTask is the task a run works, in the workshop's own words.
type RunTask struct {
	ID          uint64
	Title       string
	Description string
	BoardName   string
	ColumnName  string
	Subtasks    []RunSubtask
	// Comments oldest first; the prompt takes the last ten.
	Comments []RunComment
}

type RunSubtask struct {
	Title string
	Done  bool
}

type RunComment struct {
	Author string
	Body   string
}

// RunAgent is the agent a run uses, as its folder has it.
type RunAgent struct {
	Slug           string
	Name           string
	Title          string
	Backstory      string
	Model          string
	PermissionMode string
	AllowedTools   []string
	// TraitInstructions are the lines the agent's traits add, in order.
	TraitInstructions []string
}

// RunInput is what BuildRunSpec needs. It does no I/O: the caller reads
// the board's CLAUDE.md and mcp.json and passes their contents.
type RunInput struct {
	// RunID is a UUID: the run's folder name and Claude's session id.
	RunID    string
	Task     RunTask
	Agent    RunAgent
	Board    BoardConfig
	Worktree Worktree
	// RunDir is the run's folder under the board config's runs/.
	RunDir string
	// BoardInstructions is the board's CLAUDE.md, "" when there is none.
	BoardInstructions string
	// BoardMCP is the board's mcp.json, nil when there is none.
	BoardMCP []byte
	// MCPURL is the Bakery MCP server; Token the runner token for it.
	MCPURL string
	Token  string
}

// bakeryTools are the Bakery MCP tools a run may use without asking: read
// the task and its board, tick subtasks, add subtasks and comments, edit the
// task. Moving and deleting tasks stay with the member.
var bakeryTools = []string{
	"mcp__bakery__get_task",
	"mcp__bakery__get_board",
	"mcp__bakery__list_work_types",
	"mcp__bakery__set_subtask_done",
	"mcp__bakery__expand_task",
	"mcp__bakery__add_comment",
	"mcp__bakery__update_task",
}

// promptComments is how many of the latest comments go into the prompt.
const promptComments = 10

// BuildRunSpec turns a task, an agent and the board config into the claude
// command line, prompt and MCP config for one run.
func BuildRunSpec(in RunInput) (RunSpec, error) {
	if in.RunID == "" || in.Worktree.Path == "" || in.RunDir == "" {
		return RunSpec{}, errors.New("a run needs an id, a worktree and a run folder")
	}
	if in.Agent.Model == "" || in.Agent.PermissionMode == "" {
		return RunSpec{}, errors.New("the agent needs a model and a permission mode")
	}
	mcpPath := filepath.Join(in.RunDir, "mcp.json")
	mcp, err := mcpConfig(in.MCPURL, in.Token, in.BoardMCP)
	if err != nil {
		return RunSpec{}, err
	}

	args := []string{
		"-p", "--output-format", "stream-json", "--verbose",
		"--model", in.Agent.Model,
		"--permission-mode", in.Agent.PermissionMode,
		"--append-system-prompt", instructions(in.Agent, in.BoardInstructions),
		"--mcp-config", mcpPath,
		"--max-budget-usd", strconv.FormatFloat(in.Board.MaxBudgetUSD, 'f', 2, 64),
		"--session-id", in.RunID,
	}
	if in.Board.IsolateUserSettings {
		// Leaves out the member's ~/.claude settings, skills, hooks and
		// plugins and their own MCP servers; the worktree's project skills
		// (where the run's skills are) still load.
		args = append(args, "--setting-sources", "project,local", "--strict-mcp-config")
	}
	// Last, because the flag takes every argument up to the next flag.
	args = append(args, "--allowedTools")
	args = append(args, in.Agent.AllowedTools...)
	args = append(args, bakeryTools...)

	return RunSpec{
		Args:  args,
		Dir:   in.Worktree.Path,
		Stdin: prompt(in.Task, in.Worktree),
		Files: map[string][]byte{mcpPath: mcp},
	}, nil
}

// instructions is the system prompt the run appends: who the agent is, its
// traits, and the board's own instructions.
func instructions(a RunAgent, board string) string {
	var b strings.Builder
	b.WriteString("You are " + a.Name)
	if a.Title != "" {
		b.WriteString(", " + a.Title)
	}
	b.WriteString(".\n")
	if s := strings.TrimSpace(a.Backstory); s != "" {
		b.WriteString("\n" + s + "\n")
	}
	if len(a.TraitInstructions) > 0 {
		b.WriteString("\n")
		for _, line := range a.TraitInstructions {
			b.WriteString("- " + line + "\n")
		}
	}
	if s := strings.TrimSpace(board); s != "" {
		b.WriteString("\n# Instructions for this board\n\n" + s + "\n")
	}
	return b.String()
}

// prompt is what the run is asked to do.
func prompt(t RunTask, wt Worktree) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Task %d: %s\n\n", t.ID, t.Title)
	fmt.Fprintf(&b, "On the board \"%s\"", t.BoardName)
	if t.ColumnName != "" {
		fmt.Fprintf(&b, ", in %s", t.ColumnName)
	}
	b.WriteString(".\n")
	if s := strings.TrimSpace(t.Description); s != "" {
		b.WriteString("\n## Description\n\n" + s + "\n")
	}
	if len(t.Subtasks) > 0 {
		b.WriteString("\n## Subtasks\n\n")
		for _, s := range t.Subtasks {
			mark := " "
			if s.Done {
				mark = "x"
			}
			fmt.Fprintf(&b, "- [%s] %s\n", mark, s.Title)
		}
	}
	comments := t.Comments
	if len(comments) > promptComments {
		comments = comments[len(comments)-promptComments:]
	}
	if len(comments) > 0 {
		b.WriteString("\n## Latest comments\n")
		for _, c := range comments {
			fmt.Fprintf(&b, "\n**%s:**\n%s\n", c.Author, strings.TrimSpace(c.Body))
		}
	}
	fmt.Fprintf(&b, `
## How to work

- You are in a git worktree on branch %s, made from %s. Work only here.
- Commit your work on this branch with clear messages. Do not push.
- Keep the task up to date with the bakery MCP tools: tick off subtasks as
  you finish them (task %d's subtasks, from get_task), and add a comment when
  you are blocked or need a decision.
- End with a short summary of what you did and what is left.
`, wt.Branch, wt.Base, t.ID)
	return b.String()
}

// mcpConfig is the run's --mcp-config file: the board's servers plus the
// Bakery server as the member. The Bakery entry always wins its name.
func mcpConfig(url, token string, board []byte) ([]byte, error) {
	servers := map[string]any{}
	if len(board) > 0 {
		var cfg struct {
			MCPServers map[string]any `json:"mcpServers"`
		}
		if err := json.Unmarshal(board, &cfg); err != nil {
			return nil, fmt.Errorf("the board's mcp.json: %w", err)
		}
		for k, v := range cfg.MCPServers {
			servers[k] = v
		}
	}
	servers["bakery"] = map[string]any{
		"type":    "http",
		"url":     url,
		"headers": map[string]string{"Authorization": "Bearer " + token},
	}
	return json.MarshalIndent(map[string]any{"mcpServers": servers}, "", "  ")
}
