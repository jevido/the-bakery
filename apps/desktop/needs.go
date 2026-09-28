package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jevido/the-bakery/apps/desktop/internal/workshop"
)

// eventAgentNeeds carries every agent's needs and mood to the frontend.
const eventAgentNeeds = "agent:needs"

// AgentNeeds is one agent's needs, mood and why, for the agent card and the
// colony view. Presentation only: nothing the workshop does reads it.
type AgentNeeds struct {
	Agent  string             `json:"agent"`
	Needs  workshop.Needs     `json:"needs"`
	Mood   workshop.MoodLevel `json:"mood"`
	Reason string             `json:"reason"`
}

// needsEvery is how often needs are worked out again besides on run events:
// rest and budget move with the clock.
const needsEvery = 30 * time.Second

// AllNeeds works out every agent's needs on this machine.
func (s *WorkshopService) AllNeeds() []AgentNeeds {
	slugs, err := s.agents.Slugs()
	if err != nil {
		return nil
	}
	now := time.Now()
	runs := s.needsRuns(now)
	limits := s.dailyLimits()
	out := make([]AgentNeeds, 0, len(slugs))
	for _, slug := range slugs {
		in := workshop.NeedsInput{Now: now, Runs: runs[slug], DailyLimitUSD: limits[slug]}
		n := workshop.ComputeNeeds(in)
		out = append(out, AgentNeeds{Agent: slug, Needs: n, Mood: workshop.Mood(n), Reason: workshop.MoodReason(in, n)})
	}
	return out
}

// needsRuns gathers the last day's runs per agent: the ones this app runs
// or ran since it opened, and older ones from their run.json.
func (s *WorkshopService) needsRuns(now time.Time) map[string][]workshop.NeedsRun {
	out := map[string][]workshop.NeedsRun{}
	seen := map[string]bool{}
	for _, r := range s.Runs() {
		if r.Kind == "plan" {
			continue
		}
		seen[r.ID] = true
		out[r.AgentSlug] = append(out[r.AgentSlug], workshop.NeedsRun{
			Status: r.Status, StartedAt: r.StartedAt, EndedAt: r.EndedAt, CostUSD: r.CostUSD,
			ContextTokens: r.ContextTokens, ContextWindow: workshop.ContextWindow(r.Model),
		})
	}
	files, _ := filepath.Glob(filepath.Join(s.boards.Root(), "*", "runs", "*", "run.json"))
	since := now.Add(-24 * time.Hour)
	for _, path := range files {
		if seen[filepath.Base(filepath.Dir(path))] {
			continue
		}
		if st, err := os.Stat(path); err != nil || st.ModTime().Before(since) {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var f runFile
		if json.Unmarshal(raw, &f) != nil || f.AgentSlug == "" || f.StartedAt.IsZero() {
			continue
		}
		end := f.EndedAt
		if end.IsZero() && f.Status != "running" {
			end = f.StartedAt
		}
		out[f.AgentSlug] = append(out[f.AgentSlug], workshop.NeedsRun{
			Status: f.Status, StartedAt: f.StartedAt, EndedAt: end, CostUSD: f.CostUSD,
			ContextTokens: f.ContextTokens, ContextWindow: workshop.ContextWindow(f.Model),
		})
	}
	return out
}

// dailyLimits is what each agent may spend in a day: for each board it is
// enabled on here, the board's budget per run times its run limit; the
// largest counts.
func (s *WorkshopService) dailyLimits() map[string]float64 {
	out := map[string]float64{}
	dirs, _ := filepath.Glob(filepath.Join(s.boards.Root(), "*", "board.toml"))
	for _, path := range dirs {
		var id uint64
		if _, err := fmt.Sscan(filepath.Base(filepath.Dir(path)), &id); err != nil {
			continue
		}
		cfg, err := s.boards.Load(id)
		if err != nil {
			continue
		}
		limit := cfg.MaxBudgetUSD * float64(max(cfg.RunLimit(), 1))
		for _, slug := range cfg.Agents {
			out[slug] = max(out[slug], limit)
		}
	}
	return out
}

func (s *WorkshopService) nudgeNeeds() {
	select {
	case s.needsNudge <- struct{}{}:
	default:
	}
}

// watchNeeds sends the agents' needs to the frontend when a run changes and
// every needsEvery.
func (s *WorkshopService) watchNeeds(ctx context.Context) {
	tick := time.NewTicker(needsEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		case <-s.needsNudge:
		}
		if s.app != nil {
			s.app.Event.Emit(eventAgentNeeds, s.AllNeeds())
		}
	}
}
