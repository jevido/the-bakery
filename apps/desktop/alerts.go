package main

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/workshop"
)

// eventAlerts carries the alerts to the frontend whenever they change.
const eventAlerts = "alerts:changed"

// alertsEvery is how often alerts are worked out again besides on board,
// run and letter events: idle and review times move with the clock.
const alertsEvery = 30 * time.Second

// alertBook is what alerts remember between rounds: since when an agent
// stands idle without work in reach, since when a task waits for review,
// which failed runs were opened, and the alerts last sent.
type alertBook struct {
	mu          sync.Mutex
	idleSince   map[string]time.Time
	reviewSince map[uint64]time.Time
	seen        map[string]bool
	last        []workshop.Alert
	workTypes   map[uint64]workTypeNames
	nudge       chan struct{}
}

type workTypeNames struct {
	names map[string]string
	at    time.Time
}

func newAlertBook() *alertBook {
	return &alertBook{
		idleSince: map[string]time.Time{}, reviewSince: map[uint64]time.Time{}, seen: map[string]bool{},
		workTypes: map[uint64]workTypeNames{}, nudge: make(chan struct{}, 1),
	}
}

// Alerts lists what needs attention now, most urgent first.
func (s *WorkshopService) Alerts(ctx context.Context) []workshop.Alert {
	s.alerts.mu.Lock()
	last := s.alerts.last
	s.alerts.mu.Unlock()
	if last == nil {
		return s.refreshAlerts(ctx)
	}
	return last
}

// SeenRun marks a run as opened: a failed run then stops being an alert.
// Only opening it after it ended counts; watching it run is not reading
// how it went.
func (s *WorkshopService) SeenRun(id string) {
	s.runsMu.Lock()
	run := s.runs[id]
	s.runsMu.Unlock()
	if run == nil || run.snapshot().Status == "running" {
		return
	}
	s.alerts.mu.Lock()
	s.alerts.seen[id] = true
	s.alerts.mu.Unlock()
	s.nudgeAlerts()
}

func (s *WorkshopService) nudgeAlerts() {
	select {
	case s.alerts.nudge <- struct{}{}:
	default:
	}
}

// watchAlerts works the alerts out on every nudge and every alertsEvery,
// and tells the frontend when they changed.
func (s *WorkshopService) watchAlerts(ctx context.Context) {
	tick := time.NewTicker(alertsEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		case <-s.alerts.nudge:
		}
		s.refreshAlerts(ctx)
	}
}

func (s *WorkshopService) refreshAlerts(ctx context.Context) []workshop.Alert {
	if s.session.Token() == "" {
		return []workshop.Alert{}
	}
	now := time.Now()
	runs := s.Runs()
	boards, guildOf := s.alertBoards(ctx, now, runs)
	st := workshop.AlertState{Now: now, Boards: boards}
	s.alerts.mu.Lock()
	for _, r := range runs {
		if r.Kind == "plan" || r.Status == "running" {
			continue
		}
		st.Runs = append(st.Runs, workshop.AlertRun{ID: r.ID, AgentName: r.AgentName, TaskTitle: r.TaskTitle, BoardID: r.BoardID,
			TaskID: r.TaskID, Status: r.Status, EndedAt: r.EndedAt, Seen: s.alerts.seen[r.ID]})
	}
	s.alerts.mu.Unlock()
	byID := map[string]RunInfo{}
	for _, r := range runs {
		byID[r.ID] = r
	}
	for _, l := range s.desk.Open() {
		r := byID[l.RunID]
		st.Letters = append(st.Letters, workshop.AlertLetter{ID: l.ID, RunID: l.RunID, AgentName: r.AgentName, TaskTitle: r.TaskTitle,
			BoardID: r.BoardID, TaskID: r.TaskID, Kind: l.Kind})
	}
	alerts := workshop.Alerts(st)
	if alerts == nil {
		alerts = []workshop.Alert{}
	}
	for i := range alerts {
		alerts[i].GuildID = guildOf[alerts[i].BoardID]
	}
	s.alerts.mu.Lock()
	changed := !slices.Equal(s.alerts.last, alerts)
	s.alerts.last = alerts
	s.alerts.mu.Unlock()
	if changed && s.app != nil {
		s.app.Event.Emit(eventAlerts, alerts)
	}
	return alerts
}

// alertBoards reads every board with agents enabled on this machine, and
// which guild each is in. A board the API cannot give right now is left
// out this round.
func (s *WorkshopService) alertBoards(ctx context.Context, now time.Time, runs []RunInfo) ([]workshop.AlertBoard, map[uint64]uint64) {
	guildOf := map[uint64]uint64{}
	token := s.session.Token()
	busy := map[string]bool{}
	movedAt := map[uint64]time.Time{}
	for _, r := range runs {
		if r.Status == "running" {
			busy[r.AgentSlug] = true
		}
		if r.MovedTo != "" && r.EndedAt.After(movedAt[r.TaskID]) {
			movedAt[r.TaskID] = r.EndedAt
		}
	}
	paths, _ := filepath.Glob(filepath.Join(s.boards.Root(), "*", "board.toml"))
	var out []workshop.AlertBoard
	inReview := map[uint64]bool{}
	for _, path := range paths {
		var id uint64
		if _, err := fmt.Sscan(filepath.Base(filepath.Dir(path)), &id); err != nil {
			continue
		}
		cfg, err := s.boards.Load(id)
		if err != nil || len(cfg.Agents) == 0 {
			continue
		}
		rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		view, err := s.client.GetBoard(rctx, token, id)
		cancel()
		if err != nil {
			continue
		}
		guildOf[id] = view.Board.GuildID
		b := workshop.AlertBoard{ID: id, Name: view.Board.Name, Paused: cfg.Speed == "paused", WorkTypes: s.workTypeNames(ctx, view.Board.GuildID, now)}
		var ready []workshop.TaskState
		review := cfg.FinishColumn
		if review == "" {
			review = "Review"
		}
		for _, c := range view.Columns {
			switch {
			case strings.EqualFold(c.Name, cfg.ReadyColumn):
				for _, t := range c.Tasks {
					wt := ""
					if t.WorkType != nil {
						wt = *t.WorkType
					}
					b.Ready = append(b.Ready, workshop.AlertTask{ID: t.ID, Title: t.Title, WorkType: wt})
					ts := workshop.TaskState{ID: t.ID, WorkType: wt, Taken: t.Claim != nil, Forbidden: t.Forbidden}
					if t.PrioritizedAgentID != nil {
						ts.PrioritizedAgentID = *t.PrioritizedAgentID
					}
					ready = append(ready, ts)
				}
			case strings.EqualFold(c.Name, review):
				for _, t := range c.Tasks {
					inReview[t.ID] = true
					b.Review = append(b.Review, workshop.AlertTask{ID: t.ID, Title: t.Title, Since: s.reviewSince(t, movedAt, now)})
				}
			}
		}
		for _, slug := range cfg.Agents {
			f, err := s.agents.Folder(slug)
			if err != nil {
				continue
			}
			a := workshop.AlertAgent{Slug: slug, Name: f.Manifest.Name, Priorities: f.Manifest.WorkPriorities}
			var agentID uint64
			if f.Sync != nil {
				agentID = f.Sync.AgentID
			}
			// Idle with no task it would take: PickNext gives it nothing
			// even with the board to itself.
			stuck := !busy[slug] && len(workshop.PickNext(
				[]workshop.AgentState{{Slug: slug, AgentID: agentID, Priorities: f.Manifest.WorkPriorities}}, ready, 0, 1)) == 0
			a.IdleSince = s.idleSince(id, slug, stuck, now)
			b.Agents = append(b.Agents, a)
		}
		out = append(out, b)
	}
	// A task that left review starts afresh when it comes back.
	s.alerts.mu.Lock()
	for id := range s.alerts.reviewSince {
		if !inReview[id] {
			delete(s.alerts.reviewSince, id)
		}
	}
	s.alerts.mu.Unlock()
	return out, guildOf
}

// reviewSince is when a task was first seen in review: when a run here
// moved it there, or else the first round that saw it.
func (s *WorkshopService) reviewSince(t api.Task, movedAt map[uint64]time.Time, now time.Time) time.Time {
	s.alerts.mu.Lock()
	defer s.alerts.mu.Unlock()
	if since, ok := s.alerts.reviewSince[t.ID]; ok {
		return since
	}
	since := now
	if at, ok := movedAt[t.ID]; ok {
		since = at
	}
	s.alerts.reviewSince[t.ID] = since
	return since
}

func (s *WorkshopService) idleSince(boardID uint64, slug string, stuck bool, now time.Time) time.Time {
	key := fmt.Sprintf("%d:%s", boardID, slug)
	s.alerts.mu.Lock()
	defer s.alerts.mu.Unlock()
	if !stuck {
		delete(s.alerts.idleSince, key)
		return time.Time{}
	}
	since, ok := s.alerts.idleSince[key]
	if !ok {
		since = now
		s.alerts.idleSince[key] = since
	}
	return since
}

// workTypeNames are a guild's work type names by key, asked again after
// five minutes.
func (s *WorkshopService) workTypeNames(ctx context.Context, guildID uint64, now time.Time) map[string]string {
	s.alerts.mu.Lock()
	cached, ok := s.alerts.workTypes[guildID]
	s.alerts.mu.Unlock()
	if ok && now.Sub(cached.at) < 5*time.Minute {
		return cached.names
	}
	wts, err := s.client.ListWorkTypes(ctx, s.session.Token(), guildID)
	if err != nil {
		return cached.names
	}
	names := map[string]string{}
	for _, w := range wts {
		names[w.Key] = w.Name
	}
	s.alerts.mu.Lock()
	s.alerts.workTypes[guildID] = workTypeNames{names: names, at: now}
	s.alerts.mu.Unlock()
	return names
}
