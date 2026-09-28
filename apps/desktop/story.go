package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/workshop"
)

// eventStoryLetter carries a new event letter to the frontend.
const eventStoryLetter = "story:letter"

// keptStoryLetters is how many event letters a board's history keeps.
const keptStoryLetters = 30

// storyBook is the storyteller and what it needs to hear board events in
// its own terms: the board's column names, who this machine has seen on
// each board, and when each board last had its columns checked for a cold
// snap.
type storyBook struct {
	mu        sync.Mutex
	teller    workshop.Storyteller
	columns   map[uint64]boardColumns
	coldCheck map[uint64]time.Time
}

type boardColumns struct {
	name  string
	names map[uint64]string
	at    time.Time
}

func newStoryBook() *storyBook {
	return &storyBook{columns: map[uint64]boardColumns{}, coldCheck: map[uint64]time.Time{}}
}

// StoredEventLetter is an event letter in a board's history.
type StoredEventLetter struct {
	workshop.EventLetter
	Read bool `json:"read"`
}

// EventLetters is a board's event letter history, newest first.
func (s *WorkshopService) EventLetters(boardID uint64) []StoredEventLetter {
	s.story.mu.Lock()
	defer s.story.mu.Unlock()
	return s.readStoryLetters(boardID)
}

// ReadEventLetter marks an event letter as read: it leaves the stack of
// letters and stays in the history.
func (s *WorkshopService) ReadEventLetter(boardID uint64, id string) {
	s.story.mu.Lock()
	defer s.story.mu.Unlock()
	letters := s.readStoryLetters(boardID)
	for i := range letters {
		if letters[i].ID == id {
			letters[i].Read = true
		}
	}
	s.writeStoryLetters(boardID, letters)
}

// hearBoardEvent passes a board event to the storyteller. It reads what
// the event leaves out (a new task's title, a column's name) from the API,
// so it runs off the stream's goroutine.
func (s *WorkshopService) hearBoardEvent(ev api.BoardEvent) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		for _, e := range s.storyEventsOf(ctx, ev) {
			s.tell(e)
		}
	}()
}

func (s *WorkshopService) storyEventsOf(ctx context.Context, ev api.BoardEvent) []workshop.StoryEvent {
	token := s.session.Token()
	if token == "" {
		return nil
	}
	cols := s.columnsOf(ctx, ev.BoardID, false)
	at := ev.At
	if at.IsZero() {
		at = time.Now()
	}
	base := workshop.StoryEvent{BoardID: ev.BoardID, BoardName: cols.name, At: at}
	switch ev.Type {
	case "task.created":
		detail, err := s.client.GetTask(ctx, token, uint64(num(ev.Data["task_id"])))
		if err != nil {
			return nil
		}
		wt := ""
		if detail.Task.WorkType != nil {
			wt = *detail.Task.WorkType
		}
		base.Kind, base.BugLike = "task_created", workshop.BugLike(detail.Task.Title, wt)
		return []workshop.StoryEvent{base}
	case "task.moved":
		to, from := uint64(num(ev.Data["to"])), uint64(num(ev.Data["from"]))
		if to == from {
			return nil
		}
		name, ok := cols.names[to]
		if !ok {
			name = s.columnsOf(ctx, ev.BoardID, true).names[to]
		}
		if !strings.EqualFold(name, "Done") {
			return nil
		}
		base.Kind = "task_done"
		return []workshop.StoryEvent{base}
	case "presence":
		var out []workshop.StoryEvent
		for _, p := range s.newcomers(ev) {
			e := base
			e.Kind, e.Name = "newcomer", p
			out = append(out, e)
		}
		if ev.Data["state"] == "snapshot" {
			if e, ok := s.coldColumn(ctx, ev.BoardID, base); ok {
				out = append(out, e)
			}
		}
		return out
	}
	return nil
}

// newcomers are the members in a presence event this machine has never
// seen on the board. The first time a board is opened here, everyone
// present is simply remembered.
func (s *WorkshopService) newcomers(ev api.BoardEvent) []string {
	me := uint64(0)
	if m := s.session.Member(); m != nil {
		me = m.ID
	}
	var present []map[string]any
	switch ev.Data["state"] {
	case "snapshot":
		for _, p := range asList(ev.Data["present"]) {
			present = append(present, p)
		}
	case "joined":
		present = []map[string]any{ev.Data}
	default:
		return nil
	}
	path := filepath.Join(s.boards.Dir(ev.BoardID), "members-seen.json")
	s.story.mu.Lock()
	defer s.story.mu.Unlock()
	var seen []uint64
	raw, err := os.ReadFile(path)
	first := err != nil
	if !first {
		_ = json.Unmarshal(raw, &seen)
	}
	var out []string
	for _, p := range present {
		id := uint64(num(p["member_id"]))
		if id == 0 || slices.Contains(seen, id) {
			continue
		}
		seen = append(seen, id)
		if !first && id != me {
			out = append(out, fmt.Sprint(p["display_name"]))
		}
	}
	if raw, err := json.Marshal(seen); err == nil {
		_ = os.MkdirAll(filepath.Dir(path), 0o700)
		_ = os.WriteFile(path, raw, 0o600)
	}
	return out
}

// coldColumn looks, at most once a day per board, for a column other than
// Done whose tasks have all stood still for a week.
func (s *WorkshopService) coldColumn(ctx context.Context, boardID uint64, base workshop.StoryEvent) (workshop.StoryEvent, bool) {
	s.story.mu.Lock()
	if at, ok := s.story.coldCheck[boardID]; ok && time.Since(at) < 24*time.Hour {
		s.story.mu.Unlock()
		return workshop.StoryEvent{}, false
	}
	s.story.coldCheck[boardID] = time.Now()
	s.story.mu.Unlock()
	token := s.session.Token()
	view, err := s.client.GetBoard(ctx, token, boardID)
	if err != nil {
		return workshop.StoryEvent{}, false
	}
	for _, c := range view.Columns {
		if strings.EqualFold(c.Name, "Done") || len(c.Tasks) < workshop.ColdSnapTasks {
			continue
		}
		cold := true
		for _, t := range c.Tasks {
			acts, err := s.client.ListActivity(ctx, token, t.ID, 0, 1)
			if err != nil || len(acts) == 0 || time.Since(acts[0].At) < workshop.ColdSnapAfter {
				cold = false
				break
			}
		}
		if cold {
			base.Kind, base.Column, base.Count = "cold_column", c.Name, len(c.Tasks)
			return base, true
		}
	}
	return workshop.StoryEvent{}, false
}

// columnsOf names a board and its columns, asked again after a minute or
// when fresh is set.
func (s *WorkshopService) columnsOf(ctx context.Context, boardID uint64, fresh bool) boardColumns {
	s.story.mu.Lock()
	cached, ok := s.story.columns[boardID]
	s.story.mu.Unlock()
	if ok && !fresh && time.Since(cached.at) < time.Minute {
		return cached
	}
	view, err := s.client.GetBoard(ctx, s.session.Token(), boardID)
	if err != nil {
		return cached
	}
	cols := boardColumns{name: view.Board.Name, names: map[uint64]string{}, at: time.Now()}
	for _, c := range view.Columns {
		cols.names[c.ID] = c.Name
	}
	s.story.mu.Lock()
	s.story.columns[boardID] = cols
	s.story.mu.Unlock()
	return cols
}

// tell hands an event to the storyteller and sends the letter it writes.
func (s *WorkshopService) tell(e workshop.StoryEvent) {
	s.story.mu.Lock()
	l := s.story.teller.Observe(e)
	if l != nil {
		letters := append([]StoredEventLetter{{EventLetter: *l}}, s.readStoryLetters(e.BoardID)...)
		s.writeStoryLetters(e.BoardID, letters)
	}
	s.story.mu.Unlock()
	if l != nil && s.app != nil {
		s.app.Event.Emit(eventStoryLetter, StoredEventLetter{EventLetter: *l})
	}
}

// readStoryLetters and writeStoryLetters keep a board's history in its
// board config folder. The caller holds s.story.mu.
func (s *WorkshopService) readStoryLetters(boardID uint64) []StoredEventLetter {
	raw, err := os.ReadFile(filepath.Join(s.boards.Dir(boardID), "event-letters.json"))
	var out []StoredEventLetter
	if err == nil {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = []StoredEventLetter{}
	}
	return out
}

func (s *WorkshopService) writeStoryLetters(boardID uint64, letters []StoredEventLetter) {
	if len(letters) > keptStoryLetters {
		letters = letters[:keptStoryLetters]
	}
	raw, err := json.MarshalIndent(letters, "", "  ")
	if err != nil {
		return
	}
	dir := s.boards.Dir(boardID)
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, "event-letters.json"), raw, 0o600)
}

func num(v any) float64 {
	f, _ := v.(float64)
	return f
}

func asList(v any) []map[string]any {
	items, _ := v.([]any)
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}
