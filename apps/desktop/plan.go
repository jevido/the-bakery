package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/workshop"
)

// planTimeout bounds one "Plan it" run.
const planTimeout = 10 * time.Minute

// PlanTask has an agent read the board's repository in plan mode and
// propose subtasks for the task. Nothing changes on the board: the member
// reviews the proposal and accepts it (TaskService.ExpandTask) or not. The
// plan counts as a run, recorded with its cost.
func (s *WorkshopService) PlanTask(ctx context.Context, boardID, taskID uint64, agentSlug string) (workshop.Proposal, error) {
	token := s.session.Token()
	if token == "" {
		return workshop.Proposal{}, ErrSignedOut
	}
	cfg, err := s.boards.Load(boardID)
	if err != nil {
		return workshop.Proposal{}, err
	}
	if !cfg.Linked() {
		return workshop.Proposal{}, errors.New("this board is not linked to a repository on this machine; open Board settings")
	}
	if err := cfg.Validate(ctx); err != nil {
		return workshop.Proposal{}, fmt.Errorf("board settings: %w", err)
	}
	if _, err := exec.LookPath(s.claudeBinary()); err != nil {
		return workshop.Proposal{}, errors.New("claude is not installed or not on the PATH of this app")
	}
	folder, err := s.agents.Folder(agentSlug)
	if err != nil {
		return workshop.Proposal{}, fmt.Errorf("agent %s: %w", agentSlug, err)
	}
	if folder.Sync == nil || folder.Sync.AgentID == 0 {
		return workshop.Proposal{}, fmt.Errorf("%s has not synced yet; wait for the sync, then try again", folder.Manifest.Name)
	}

	view, err := s.client.GetBoard(ctx, token, boardID)
	if err != nil {
		return workshop.Proposal{}, err
	}
	detail, err := s.client.GetTask(ctx, token, taskID)
	if err != nil {
		return workshop.Proposal{}, err
	}
	if detail.Task.ParentID != nil {
		return workshop.Proposal{}, errors.New("a subtask cannot be planned into subtasks")
	}
	workTypes, err := s.client.ListWorkTypes(ctx, token, view.Board.GuildID)
	if err != nil {
		return workshop.Proposal{}, err
	}
	keys := make([]string, len(workTypes))
	for i, w := range workTypes {
		keys[i] = w.Key
	}
	traits, _, err := s.client.Traits(ctx, token)
	if err != nil {
		return workshop.Proposal{}, err
	}
	runnerToken, err := s.session.RunnerToken(ctx)
	if err != nil {
		return workshop.Proposal{}, err
	}

	m := folder.Manifest
	agent := workshop.RunAgent{Slug: agentSlug, Name: m.Name, Title: m.Title, Backstory: m.Backstory, Model: m.Model, PermissionMode: "plan"}
	for _, key := range m.Traits {
		for _, t := range traits {
			if t.Key == key && t.Instruction != "" {
				agent.TraitInstructions = append(agent.TraitInstructions, t.Instruction)
			}
		}
	}
	task := workshop.RunTask{ID: taskID, Title: detail.Task.Title, Description: detail.Task.Description, BoardName: view.Board.Name}
	for _, st := range detail.Subtasks {
		task.Subtasks = append(task.Subtasks, workshop.RunSubtask{Title: st.Title, Done: st.Done})
	}
	id := newUUID()
	boardDir := s.boards.Dir(boardID)
	runDir := filepath.Join(boardDir, "runs", id)
	instructions, _ := os.ReadFile(filepath.Join(boardDir, "CLAUDE.md"))
	spec, err := workshop.BuildPlanSpec(workshop.PlanInput{
		RunID: id, Task: task, Agent: agent, Board: cfg, WorkTypes: keys, RunDir: runDir,
		MCPURL: strings.TrimRight(s.client.BaseURL(), "/") + "/mcp", Token: runnerToken, BoardInstructions: string(instructions),
	})
	if err != nil {
		return workshop.Proposal{}, err
	}
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return workshop.Proposal{}, err
	}
	for path, content := range spec.Files {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			return workshop.Proposal{}, err
		}
	}

	// A plan takes a place in the board's runs while it goes.
	s.startMu.Lock()
	running := 0
	for _, r := range s.Runs() {
		if r.Status == "running" && r.BoardID == boardID {
			running++
		}
	}
	if running >= max(cfg.RunLimit(), cfg.MaxConcurrentRuns) {
		s.startMu.Unlock()
		return workshop.Proposal{}, fmt.Errorf("this board already has %d runs going on this machine, its limit", running)
	}
	host, _ := os.Hostname()
	apiRun, err := s.client.StartRun(ctx, token, taskID, folder.Sync.AgentID, m.Name, "plan", host, "")
	if err != nil {
		s.startMu.Unlock()
		return workshop.Proposal{}, fmt.Errorf("recording the plan: %w", err)
	}
	run := &localRun{info: RunInfo{
		ID: id, APIRunID: apiRun.ID, BoardID: boardID, TaskID: taskID, TaskTitle: detail.Task.Title,
		AgentSlug: agentSlug, AgentName: m.Name, Worktree: cfg.Repo, Model: m.Model, PermissionMode: "plan",
		StartedAt: time.Now(), Status: "running", State: "thinking", Kind: "plan",
	}, dir: runDir}
	s.runsMu.Lock()
	s.runs[id] = run
	s.runsMu.Unlock()
	s.startMu.Unlock()
	s.emitRuns()

	pctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), planTimeout)
	defer cancel()
	cmd := exec.CommandContext(pctx, s.claudeBinary(), spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Stdin = strings.NewReader(spec.Stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	_ = os.WriteFile(filepath.Join(runDir, "result.json"), stdout.Bytes(), 0o600)

	fallback := ""
	if detail.Task.WorkType != nil {
		fallback = *detail.Task.WorkType
	}
	res, perr := workshop.ParsePlanOutput(stdout.Bytes(), keys, fallback)
	status, summary := "succeeded", ""
	switch {
	case perr != nil:
		status, summary = "failed", strings.TrimSpace(lastLine(stderr.String()))
		if summary == "" && runErr != nil {
			summary = runErr.Error()
		}
	case res.Error != "":
		status, summary = "failed", res.Error
	default:
		summary = res.Proposal.Plan
	}
	run.mu.Lock()
	run.info.Status, run.info.CostUSD, run.info.Turns = status, res.Proposal.CostUSD, res.Turns
	run.info.State = map[bool]string{true: "done", false: "failed"}[status == "succeeded"]
	run.mu.Unlock()
	s.emitRuns()
	fctx, fcancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer fcancel()
	_, _ = s.client.FinishRun(fctx, token, apiRun.ID, api.RunEnd{Status: status, CostUSD: res.Proposal.CostUSD, Turns: res.Turns, Summary: summary})
	s.Wake()

	if status != "succeeded" {
		return workshop.Proposal{}, fmt.Errorf("%s could not make a plan: %s", m.Name, summary)
	}
	return res.Proposal, nil
}
