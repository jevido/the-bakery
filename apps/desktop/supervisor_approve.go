package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/workshop"
)

// Events the canvas acts out: the supervisor answering, and handing a task
// to an agent because the member approved it.
const (
	eventSupervisorThinking = "supervisor:thinking"
	eventSupervisorAssigned = "supervisor:assigned"
)

// Approve carries out one proposal the member approved and records what
// became of it. The board is read again first; a proposal that no longer
// fits is refused with the reason, and nothing is changed. subtasks, for a
// split, are the member's edited rows (nil keeps the proposal's).
func (s *SupervisorService) Approve(ctx context.Context, boardID uint64, entryID string, index int, subtasks []api.SubtaskDraft) (string, error) {
	p, err := s.proposal(boardID, entryID, index)
	if err != nil {
		return "", err
	}
	outcome, err := s.apply(ctx, boardID, p, subtasks)
	if err != nil {
		outcome = "refused: " + err.Error()
	}
	if serr := s.SetOutcome(boardID, entryID, index, outcome); serr != nil {
		return "", serr
	}
	return outcome, err
}

// Decline records that the member said no; nothing else happens.
func (s *SupervisorService) Decline(boardID uint64, entryID string, index int) error {
	if _, err := s.proposal(boardID, entryID, index); err != nil {
		return err
	}
	return s.SetOutcome(boardID, entryID, index, "declined")
}

func (s *SupervisorService) proposal(boardID uint64, entryID string, index int) (ChatProposal, error) {
	history, err := s.History(boardID)
	if err != nil {
		return ChatProposal{}, err
	}
	for _, e := range history {
		if e.ID == entryID && index >= 0 && index < len(e.Proposals) {
			p := e.Proposals[index]
			if p.Outcome != "" {
				return ChatProposal{}, errors.New("that proposal was already " + strings.SplitN(p.Outcome, ":", 2)[0])
			}
			return p, nil
		}
	}
	return ChatProposal{}, errors.New("that proposal is not in the chat any more")
}

func (s *SupervisorService) apply(ctx context.Context, boardID uint64, p ChatProposal, subtasks []api.SubtaskDraft) (string, error) {
	token := s.work.session.Token()
	if token == "" {
		return "", ErrSignedOut
	}
	snap, err := s.snapshot(ctx, boardID)
	if err != nil {
		return "", err
	}
	switch p.Kind {
	case "assign":
		return s.assign(ctx, boardID, snap, p.SupervisorProposal)
	case "split":
		if _, ok := findTask(snap, p.TaskID); !ok {
			return "", errors.New("the task is no longer on the board")
		}
		if subtasks == nil {
			for _, st := range p.Subtasks {
				subtasks = append(subtasks, api.SubtaskDraft{Title: st.Title, Description: st.Description, WorkType: st.WorkType})
			}
		}
		added, err := s.work.client.ExpandTask(ctx, token, p.TaskID, subtasks)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("added %d subtasks", len(added)), nil
	case "reorder":
		return s.reorder(ctx, token, boardID, snap, p.TaskIDs)
	}
	return "", errors.New("the supervisor proposed something this app cannot do")
}

// assign starts the run now when it can; when the agent is busy or the
// board is at its run limit, it sets the task aside for the agent instead,
// and the scheduler starts it when a place frees up.
func (s *SupervisorService) assign(ctx context.Context, boardID uint64, snap workshop.SupervisorSnapshot, p workshop.SupervisorProposal) (string, error) {
	t, ok := findTask(snap, p.TaskID)
	switch {
	case !ok:
		return "", errors.New("the task is no longer on the board")
	case t.Forbidden:
		return "", errors.New("the task is forbidden for agents")
	case t.Claimed:
		return "", errors.New("someone's agent is already on the task")
	}
	i := slices.IndexFunc(snap.Agents, func(a workshop.SupervisorAgent) bool { return a.Slug == p.Agent })
	if i < 0 {
		return "", errors.New(p.Agent + " is not enabled on this board here")
	}
	agent := snap.Agents[i]
	if !agent.Busy {
		_, err := s.work.StartRun(ctx, boardID, p.TaskID, p.Agent)
		if err == nil {
			if s.work.app != nil {
				s.work.app.Event.Emit(eventSupervisorAssigned, map[string]any{"board_id": boardID, "agent": p.Agent, "task_id": p.TaskID})
			}
			return "started", nil
		}
		if !strings.Contains(err.Error(), "its limit") {
			return "", err
		}
	}
	f, err := s.work.agents.Folder(p.Agent)
	if err != nil || f.Sync == nil || f.Sync.AgentID == 0 {
		return "", errors.New(agent.Name + " has not synced yet")
	}
	id := f.Sync.AgentID
	if _, err := s.work.client.DraftTask(ctx, s.work.session.Token(), p.TaskID, &id, nil); err != nil {
		return "", err
	}
	s.work.Wake()
	return "queued for " + agent.Name, nil
}

// reorder moves the ready column's tasks into the proposed order, one after
// the other.
func (s *SupervisorService) reorder(ctx context.Context, token string, boardID uint64, snap workshop.SupervisorSnapshot, ids []uint64) (string, error) {
	var ready []workshop.SupervisorTask
	for _, c := range snap.Columns {
		if strings.EqualFold(c.Name, snap.ReadyColumn) {
			ready = c.Tasks
		}
	}
	if len(ready) != len(ids) {
		return "", errors.New("the ready column changed since; ask the supervisor again")
	}
	for _, id := range ids {
		if !slices.ContainsFunc(ready, func(t workshop.SupervisorTask) bool { return t.ID == id }) {
			return "", errors.New("the ready column changed since; ask the supervisor again")
		}
	}
	view, err := s.work.client.GetBoard(ctx, token, boardID)
	if err != nil {
		return "", err
	}
	var column uint64
	for _, c := range view.Columns {
		if strings.EqualFold(c.Name, snap.ReadyColumn) {
			column = c.ID
		}
	}
	var after *uint64
	for i := range ids {
		id := ids[i]
		if _, err := s.work.client.MoveTask(ctx, token, id, column, after, nil); err != nil {
			return "", fmt.Errorf("reordering stopped part way: %w", err)
		}
		after = &ids[i]
	}
	return "reordered", nil
}

func findTask(snap workshop.SupervisorSnapshot, id uint64) (workshop.SupervisorTask, bool) {
	for _, c := range snap.Columns {
		for _, t := range c.Tasks {
			if t.ID == id {
				return t, true
			}
		}
	}
	return workshop.SupervisorTask{}, false
}
