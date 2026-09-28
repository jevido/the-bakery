package workshop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// planSchema is what "Plan it" asks Claude to answer with: a short plan
// and 2 to 12 subtasks.
const planSchema = `{"type":"object","properties":{"plan":{"type":"string"},"subtasks":{"type":"array","maxItems":12,"items":{"type":"object","properties":{"title":{"type":"string","maxLength":200},"description":{"type":"string"},"work_type":{"type":"string"}},"required":["title","work_type"]}}},"required":["plan","subtasks"]}`

// PlanInput is what a plan run needs: the task, the agent, the guild's work
// types (keys) and the board config. The run reads the repository, in plan
// mode, and changes nothing.
type PlanInput struct {
	RunID     string
	Task      RunTask
	Agent     RunAgent
	Board     BoardConfig
	WorkTypes []string
	RunDir    string
	MCPURL    string
	Token     string
	// BoardInstructions is the board's CLAUDE.md.
	BoardInstructions string
}

// BuildPlanSpec is the claude command line for "Plan it".
func BuildPlanSpec(in PlanInput) (RunSpec, error) {
	if in.RunID == "" || in.RunDir == "" || in.Board.Repo == "" || in.Agent.Model == "" {
		return RunSpec{}, errors.New("a plan needs an id, a folder, a linked repository and an agent with a model")
	}
	mcpPath := filepath.Join(in.RunDir, "mcp.json")
	mcp, err := mcpConfig(in.MCPURL, in.Token, nil, nil)
	if err != nil {
		return RunSpec{}, err
	}
	args := []string{
		"-p", "--output-format", "json",
		"--permission-mode", "plan",
		"--json-schema", planSchema,
		"--model", in.Agent.Model,
		"--append-system-prompt", instructions(in.Agent, in.BoardInstructions),
		"--mcp-config", mcpPath,
		"--max-budget-usd", strconv.FormatFloat(in.Board.MaxBudgetUSD, 'f', 2, 64),
		"--session-id", in.RunID,
	}
	if in.Board.IsolateUserSettings {
		args = append(args, "--setting-sources", "project,local", "--strict-mcp-config")
	}
	args = append(args, "--allowedTools", "mcp__bakery__get_task", "mcp__bakery__get_board", "mcp__bakery__list_work_types")
	return RunSpec{Args: args, Dir: in.Board.Repo, Stdin: planPrompt(in), Files: map[string][]byte{mcpPath: mcp}}, nil
}

func planPrompt(in PlanInput) string {
	var b strings.Builder
	t := in.Task
	fmt.Fprintf(&b, "# Plan task %d: %s\n\n", t.ID, t.Title)
	if s := strings.TrimSpace(t.Description); s != "" {
		b.WriteString("## Description\n\n" + s + "\n\n")
	}
	if len(t.Subtasks) > 0 {
		b.WriteString("## Subtasks it already has\n\n")
		for _, s := range t.Subtasks {
			fmt.Fprintf(&b, "- %s\n", s.Title)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, `## What to do

Read the repository to understand what this task takes, but change nothing.
Propose 2 to 12 subtasks that together finish the task, in the order to do
them, each small enough for one agent to do in one go. For each: a title, a
short description of what "done" means, and the kind of work it needs as one
of these work type keys: %s.

Answer with a short plan (a few sentences) and the subtasks.
`, strings.Join(in.WorkTypes, ", "))
	return b.String()
}

// Proposal is a plan an agent proposed: nothing is saved until a person
// accepts it.
type Proposal struct {
	Plan     string        `json:"plan"`
	Subtasks []ProposedSub `json:"subtasks"`
	CostUSD  float64       `json:"cost_usd"`
}

type ProposedSub struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	WorkType    string `json:"work_type"`
}

// PlanResult is the end of a plan run as Claude reports it.
type PlanResult struct {
	Proposal Proposal
	Subtype  string
	IsError  bool
	Turns    int
	Error    string
}

// ParsePlanOutput reads claude's --output-format json answer: the last JSON
// line of stdout (a version manager may print a line before it). Work types
// the guild does not have become fallback; titles are trimmed and empty ones
// dropped.
func ParsePlanOutput(stdout []byte, workTypes []string, fallback string) (PlanResult, error) {
	lines := bytes.Split(bytes.TrimSpace(stdout), []byte("\n"))
	var out struct {
		Subtype          string  `json:"subtype"`
		IsError          bool    `json:"is_error"`
		TotalCostUSD     float64 `json:"total_cost_usd"`
		NumTurns         int     `json:"num_turns"`
		Result           string  `json:"result"`
		StructuredOutput *struct {
			Plan     string        `json:"plan"`
			Subtasks []ProposedSub `json:"subtasks"`
		} `json:"structured_output"`
	}
	found := false
	for i := len(lines) - 1; i >= 0; i-- {
		if json.Unmarshal(lines[i], &out) == nil && out.Subtype != "" {
			found = true
			break
		}
	}
	if !found {
		return PlanResult{}, errors.New("claude gave no answer")
	}
	res := PlanResult{Subtype: out.Subtype, IsError: out.IsError, Turns: out.NumTurns, Proposal: Proposal{CostUSD: out.TotalCostUSD}}
	if out.IsError || out.StructuredOutput == nil {
		res.Error = strings.TrimSpace(out.Result)
		if res.Error == "" {
			res.Error = "no plan came back (" + out.Subtype + ")"
		}
		return res, nil
	}
	res.Proposal.Plan = strings.TrimSpace(out.StructuredOutput.Plan)
	for _, s := range out.StructuredOutput.Subtasks {
		s.Title = strings.TrimSpace(s.Title)
		if s.Title == "" {
			continue
		}
		if len([]rune(s.Title)) > 200 {
			s.Title = string([]rune(s.Title)[:200])
		}
		if !slices.Contains(workTypes, s.WorkType) {
			s.WorkType = fallback
		}
		s.Description = strings.TrimSpace(s.Description)
		res.Proposal.Subtasks = append(res.Proposal.Subtasks, s)
		if len(res.Proposal.Subtasks) == 12 {
			break
		}
	}
	return res, nil
}
