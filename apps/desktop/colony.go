package main

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/workshop"
)

// schedulerEvery is how often the scheduler looks for work when nothing
// woke it sooner.
const schedulerEvery = 10 * time.Second

// Wake asks the scheduler to look for work now.
func (s *WorkshopService) Wake() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// schedule is the colony: every board on this machine that is not paused
// and has agents enabled hands its free tasks to idle agents. It runs until
// ctx ends.
func (s *WorkshopService) schedule(ctx context.Context) {
	tick := time.NewTicker(schedulerEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		case <-s.wake:
		}
		if s.session.Token() == "" {
			continue
		}
		s.scheduleOnce(ctx)
	}
}

func (s *WorkshopService) scheduleOnce(ctx context.Context) {
	configs, _ := filepath.Glob(filepath.Join(s.boards.Root(), "*", "board.toml"))
	for _, path := range configs {
		boardID, err := strconv.ParseUint(filepath.Base(filepath.Dir(path)), 10, 64)
		if err != nil {
			continue
		}
		if err := s.scheduleBoard(ctx, boardID); err != nil && ctx.Err() == nil {
			s.logger().Warn("colony: board skipped", "board", boardID, "err", err)
		}
	}
}

// scheduleBoard starts runs for one board, as PickNext says.
func (s *WorkshopService) scheduleBoard(ctx context.Context, boardID uint64) error {
	cfg, err := s.boards.Load(boardID)
	if err != nil {
		return err
	}
	limit := cfg.RunLimit()
	if limit == 0 || len(cfg.Agents) == 0 || !cfg.Linked() {
		return nil
	}
	if err := cfg.Validate(ctx); err != nil {
		return err
	}
	view, err := s.client.GetBoard(ctx, s.session.Token(), boardID)
	if err != nil {
		return err
	}
	var tasks []workshop.TaskState
	for _, c := range view.Columns {
		if !strings.EqualFold(c.Name, cfg.ReadyColumn) {
			continue
		}
		for _, t := range c.Tasks {
			ts := workshop.TaskState{ID: t.ID, Taken: t.Claim != nil, Forbidden: t.Forbidden}
			if t.WorkType != nil {
				ts.WorkType = *t.WorkType
			}
			if t.PrioritizedAgentID != nil {
				ts.PrioritizedAgentID = *t.PrioritizedAgentID
			}
			tasks = append(tasks, ts)
		}
	}
	if len(tasks) == 0 {
		return nil
	}

	running := 0
	busy := map[string]bool{}
	for _, r := range s.Runs() {
		if r.Status != "running" {
			continue
		}
		busy[r.AgentSlug] = true
		if r.BoardID == boardID {
			running++
		}
	}
	var agents []workshop.AgentState
	for _, slug := range cfg.Agents {
		f, err := s.agents.Folder(slug)
		if err != nil || f.Sync == nil || f.Sync.AgentID == 0 {
			continue // gone, or not synced yet
		}
		agents = append(agents, workshop.AgentState{Slug: slug, AgentID: f.Sync.AgentID, Priorities: f.Manifest.WorkPriorities, Busy: busy[slug]})
	}

	for _, a := range workshop.PickNext(agents, tasks, running, limit) {
		if _, err := s.StartRun(ctx, boardID, a.TaskID, a.AgentSlug); err != nil {
			var apiErr *api.Error
			if errors.As(err, &apiErr) && apiErr.Status == 409 {
				continue // another machine took it first
			}
			s.logger().Warn("colony: could not start a run", "board", boardID, "task", a.TaskID, "agent", a.AgentSlug, "err", err)
		}
	}
	return nil
}
