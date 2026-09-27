package app

import (
	"context"
	"errors"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

var (
	ErrRunNotFound  = errors.New("run not found")
	ErrRunOnSubtask = errors.New("agents work tasks on the board, not subtasks")
)

// Runs stores runs.
type Runs interface {
	Add(ctx context.Context, r domain.Run) (domain.Run, error)
	Save(ctx context.Context, r domain.Run) error
	ByID(ctx context.Context, id uint64) (domain.Run, bool, error)
	// OfTask returns a task's runs, newest first; limit 0 means all.
	OfTask(ctx context.Context, taskID uint64, limit int) ([]domain.Run, error)
}

// RunView is a run for reading: its status as it reads now, and the name
// of the member who started it.
type RunView struct {
	domain.Run
	MemberName string
}

// StartRun records that the member put an agent on a task. The agent is
// the agents context's; boards keeps its id and its name as it is now.
func (s *Service) StartRun(ctx context.Context, taskID, memberID, agentID uint64, agentName, machine, branch string) (RunView, error) {
	t, err := s.task(ctx, taskID, memberID)
	if err != nil {
		return RunView{}, err
	}
	if t.ParentID != nil {
		return RunView{}, ErrRunOnSubtask
	}
	r, err := domain.StartRun(t.ID, memberID, agentID, agentName, machine, branch, s.now())
	if err != nil {
		return RunView{}, err
	}
	if r, err = s.runs.Add(ctx, r); err != nil {
		return RunView{}, err
	}
	s.events.Publish(ctx, domain.RunStarted{RunID: r.ID, TaskID: t.ID, BoardID: t.BoardID, ActorID: memberID, AgentID: agentID, AgentName: r.AgentName})
	out, err := s.runViews(ctx, r)
	if err != nil {
		return RunView{}, err
	}
	return out[0], nil
}

// FinishRun ends a run with its outcome. Only the member who started it
// can. It works in an archived guild too: the run happened either way.
func (s *Service) FinishRun(ctx context.Context, runID, memberID uint64, status domain.RunStatus, stats domain.RunStats) (RunView, error) {
	r, found, err := s.runs.ByID(ctx, runID)
	if err != nil {
		return RunView{}, err
	}
	if !found {
		return RunView{}, ErrRunNotFound
	}
	t, err := s.readableTask(ctx, r.TaskID, memberID)
	if err != nil {
		return RunView{}, err
	}
	if err := r.Finish(memberID, status, stats, s.now()); err != nil {
		return RunView{}, err
	}
	if err := s.runs.Save(ctx, r); err != nil {
		return RunView{}, err
	}
	s.events.Publish(ctx, domain.RunFinished{RunID: r.ID, TaskID: t.ID, BoardID: t.BoardID, ActorID: memberID, AgentName: r.AgentName, Status: r.Status, CostUSD: r.CostUSD})
	out, err := s.runViews(ctx, r)
	if err != nil {
		return RunView{}, err
	}
	return out[0], nil
}

// ListRuns returns a task's runs, newest first; limit 0 means all.
func (s *Service) ListRuns(ctx context.Context, taskID, memberID uint64, limit int) ([]RunView, error) {
	t, err := s.readableTask(ctx, taskID, memberID)
	if err != nil {
		return nil, err
	}
	rs, err := s.runs.OfTask(ctx, t.ID, limit)
	if err != nil {
		return nil, err
	}
	return s.runViews(ctx, rs...)
}

func (s *Service) runViews(ctx context.Context, rs ...domain.Run) ([]RunView, error) {
	ids := make([]uint64, 0, len(rs))
	for _, r := range rs {
		ids = append(ids, r.MemberID)
	}
	names, err := s.names.DisplayNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]RunView, len(rs))
	for i, r := range rs {
		r.Status = r.StatusAt(now)
		out[i] = RunView{Run: r, MemberName: names[r.MemberID]}
	}
	return out, nil
}
