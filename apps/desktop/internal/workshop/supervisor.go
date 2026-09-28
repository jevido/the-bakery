package workshop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// supervisorSchema is what the supervisor answers with: a reply to the
// member and proposals, each one flat object whose kind says which fields
// count (assign: task_id and agent; split: task_id and subtasks; reorder:
// task_ids, the ready column's new order).
const supervisorSchema = `{"type":"object","properties":{"reply":{"type":"string"},"proposals":{"type":"array","maxItems":10,"items":{"type":"object","properties":{"kind":{"type":"string","enum":["assign","split","reorder"]},"task_id":{"type":"integer"},"agent":{"type":"string"},"subtasks":{"type":"array","maxItems":12,"items":{"type":"object","properties":{"title":{"type":"string","maxLength":200},"description":{"type":"string"},"work_type":{"type":"string"}},"required":["title"]}},"task_ids":{"type":"array","items":{"type":"integer"}},"why":{"type":"string"}},"required":["kind"]}}},"required":["reply","proposals"]}`

// supervisorPrompt is the supervisor's system prompt: who it is and what
// it may do. It never changes per message; the board comes with each one.
const supervisorPrompt = `You are the supervisor of one board in The Bakery, a workbench where a
guild plans work as tasks on boards and Claude agents work those tasks. You
talk with the member in charge of this board.

You never work a task yourself and you have no tools. You coordinate: you
suggest which agent takes which task, which tasks are too big and should be
split into subtasks, and in what order the ready column should be worked.
Nothing you suggest happens until the member approves it.

Each message comes with a snapshot of the board: its columns and tasks, and
the agents enabled on it with their work priorities (1 = wants that kind of
work first, 4 = last; a work type that is missing is off for that agent).

Rules for proposals:
- assign: a task in the ready column to one agent. Never a forbidden task,
  never a task claimed by someone, never an agent that is busy, and never an
  agent that has the task's work type off. Prefer the agent with the lowest
  number for the task's work type.
- split: a task (not a subtask) into 2 to 12 subtasks in the order to do
  them, each with a title, what done means, and a work type key from the
  snapshot. Only when the task is clearly bigger than one agent can finish in
  one go.
- reorder: the ids of every task in the ready column, in the order they
  should be worked (what others depend on first).
Only name task ids and agents from the snapshot. Propose nothing when nothing
is needed. Keep the reply short and plain; say why for each proposal in its
"why".`

// SupervisorTask is a task on the board, as the supervisor sees it.
type SupervisorTask struct {
	ID            uint64 `json:"id"`
	Title         string `json:"title"`
	WorkType      string `json:"work_type,omitempty"`
	Description   string `json:"description,omitempty"`
	SubtasksDone  int    `json:"subtasks_done,omitempty"`
	SubtasksTotal int    `json:"subtasks_total,omitempty"`
	Claimed       bool   `json:"claimed,omitempty"`
	Forbidden     bool   `json:"forbidden,omitempty"`
	// PrioritizedFor is the agent the task is set aside for, if any.
	PrioritizedFor string `json:"prioritized_for,omitempty"`
}

// SupervisorColumn is one column of the board, in order.
type SupervisorColumn struct {
	Name  string           `json:"name"`
	Tasks []SupervisorTask `json:"tasks"`
}

// SupervisorAgent is an agent enabled on the board on this machine.
type SupervisorAgent struct {
	Slug           string         `json:"slug"`
	Name           string         `json:"name"`
	WorkPriorities map[string]int `json:"work_priorities"`
	Busy           bool           `json:"busy"`
}

// SupervisorSnapshot is everything the supervisor knows of the board.
type SupervisorSnapshot struct {
	Board       string             `json:"board"`
	ReadyColumn string             `json:"ready_column"`
	WorkTypes   []string           `json:"work_types"`
	Columns     []SupervisorColumn `json:"columns"`
	Agents      []SupervisorAgent  `json:"agents"`
}

// SupervisorInput is one message to the supervisor.
type SupervisorInput struct {
	Snapshot SupervisorSnapshot
	Message  string
	Model    string
	// Session is the conversation's id; Resume says it already exists.
	Session string
	Resume  bool
	// Dir is where Claude runs: the board's config folder, never a
	// repository.
	Dir string
}

// BuildSupervisorSpec is the claude command line for one supervisor
// message: no tools at all, the answer in the supervisor's schema.
func BuildSupervisorSpec(in SupervisorInput) (RunSpec, error) {
	if in.Session == "" || in.Dir == "" || in.Model == "" || strings.TrimSpace(in.Message) == "" {
		return RunSpec{}, errors.New("a supervisor message needs a session, a folder, a model and some text")
	}
	args := []string{
		"-p", "--output-format", "json",
		"--json-schema", supervisorSchema,
		"--model", in.Model,
		"--tools", "",
		"--system-prompt", supervisorPrompt,
		"--setting-sources", "", "--strict-mcp-config",
	}
	if in.Resume {
		args = append(args, "--resume", in.Session)
	} else {
		args = append(args, "--session-id", in.Session)
	}
	snap, err := json.MarshalIndent(in.Snapshot, "", "  ")
	if err != nil {
		return RunSpec{}, err
	}
	stdin := "The board right now:\n\n```json\n" + string(snap) + "\n```\n\n" + strings.TrimSpace(in.Message) + "\n"
	return RunSpec{Args: args, Dir: in.Dir, Stdin: stdin}, nil
}

// SupervisorProposal is one change the supervisor suggests.
type SupervisorProposal struct {
	Kind     string        `json:"kind"`
	TaskID   uint64        `json:"task_id,omitempty"`
	Agent    string        `json:"agent,omitempty"`
	Subtasks []ProposedSub `json:"subtasks,omitempty"`
	TaskIDs  []uint64      `json:"task_ids,omitempty"`
	Why      string        `json:"why,omitempty"`
}

// SupervisorAnswer is the supervisor's answer to one message.
type SupervisorAnswer struct {
	Reply     string               `json:"reply"`
	Proposals []SupervisorProposal `json:"proposals"`
	CostUSD   float64              `json:"cost_usd"`
	Error     string               `json:"error,omitempty"`
}

// ParseSupervisorOutput reads claude's --output-format json answer and
// keeps only the proposals that fit the snapshot: known tasks and agents,
// assignments of ready tasks that are free, splits of tasks on the board,
// and a reorder that names exactly the ready column's tasks.
func ParseSupervisorOutput(stdout []byte, snap SupervisorSnapshot) (SupervisorAnswer, error) {
	lines := bytes.Split(bytes.TrimSpace(stdout), []byte("\n"))
	var out struct {
		Subtype          string  `json:"subtype"`
		IsError          bool    `json:"is_error"`
		TotalCostUSD     float64 `json:"total_cost_usd"`
		Result           string  `json:"result"`
		StructuredOutput *struct {
			Reply     string               `json:"reply"`
			Proposals []SupervisorProposal `json:"proposals"`
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
		return SupervisorAnswer{}, errors.New("claude gave no answer")
	}
	ans := SupervisorAnswer{CostUSD: out.TotalCostUSD}
	if out.IsError || out.StructuredOutput == nil {
		ans.Error = strings.TrimSpace(out.Result)
		if ans.Error == "" {
			ans.Error = "the supervisor gave no answer (" + out.Subtype + ")"
		}
		return ans, nil
	}
	ans.Reply = strings.TrimSpace(out.StructuredOutput.Reply)
	for _, p := range out.StructuredOutput.Proposals {
		if q, ok := fit(p, snap); ok {
			ans.Proposals = append(ans.Proposals, q)
		}
	}
	return ans, nil
}

// fit checks a proposal against the snapshot and tidies it.
func fit(p SupervisorProposal, snap SupervisorSnapshot) (SupervisorProposal, bool) {
	var ready []SupervisorTask
	all := map[uint64]SupervisorTask{}
	for _, c := range snap.Columns {
		for _, t := range c.Tasks {
			all[t.ID] = t
		}
		if strings.EqualFold(c.Name, snap.ReadyColumn) {
			ready = c.Tasks
		}
	}
	p.Why = strings.TrimSpace(p.Why)
	switch p.Kind {
	case "assign":
		t, ok := all[p.TaskID]
		if !ok || t.Forbidden || t.Claimed || !slices.ContainsFunc(ready, func(r SupervisorTask) bool { return r.ID == t.ID }) {
			return p, false
		}
		i := slices.IndexFunc(snap.Agents, func(a SupervisorAgent) bool { return a.Slug == p.Agent })
		if i < 0 {
			return p, false
		}
		if a := snap.Agents[i]; t.WorkType != "" && a.WorkPriorities[t.WorkType] == 0 {
			return p, false
		}
		p.Subtasks, p.TaskIDs = nil, nil
		return p, true
	case "split":
		if _, ok := all[p.TaskID]; !ok {
			return p, false
		}
		var subs []ProposedSub
		for _, s := range p.Subtasks {
			s.Title = strings.TrimSpace(s.Title)
			if s.Title == "" {
				continue
			}
			if len([]rune(s.Title)) > 200 {
				s.Title = string([]rune(s.Title)[:200])
			}
			if !slices.Contains(snap.WorkTypes, s.WorkType) {
				s.WorkType = all[p.TaskID].WorkType
			}
			s.Description = strings.TrimSpace(s.Description)
			subs = append(subs, s)
		}
		if len(subs) < 2 {
			return p, false
		}
		p.Subtasks, p.Agent, p.TaskIDs = subs, "", nil
		return p, true
	case "reorder":
		if len(p.TaskIDs) != len(ready) || len(ready) < 2 {
			return p, false
		}
		seen := map[uint64]bool{}
		for _, id := range p.TaskIDs {
			if seen[id] || !slices.ContainsFunc(ready, func(r SupervisorTask) bool { return r.ID == id }) {
				return p, false
			}
			seen[id] = true
		}
		same := true
		for i, t := range ready {
			same = same && p.TaskIDs[i] == t.ID
		}
		if same {
			return p, false
		}
		p.TaskID, p.Agent, p.Subtasks = 0, "", nil
		return p, true
	}
	return p, false
}

// Describe says a proposal in words, for the chat and the history.
func (p SupervisorProposal) Describe(snap SupervisorSnapshot) string {
	title := func(id uint64) string {
		for _, c := range snap.Columns {
			for _, t := range c.Tasks {
				if t.ID == id {
					return "“" + t.Title + "”"
				}
			}
		}
		return fmt.Sprintf("task %d", id)
	}
	name := p.Agent
	for _, a := range snap.Agents {
		if a.Slug == p.Agent && a.Name != "" {
			name = a.Name
		}
	}
	switch p.Kind {
	case "assign":
		return name + " takes " + title(p.TaskID)
	case "split":
		return fmt.Sprintf("Split %s into %d", title(p.TaskID), len(p.Subtasks))
	case "reorder":
		return "Reorder the ready column"
	}
	return p.Kind
}
