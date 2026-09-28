package workshop

import (
	"fmt"
	"hash/fnv"
	"slices"
	"strings"
	"time"
)

// Event letter kinds: moments on a board worth a line, told like a colony
// sim's storyteller.
const (
	StoryRaid        = "raid"
	StoryInspiration = "inspiration"
	StoryColdSnap    = "cold_snap"
	StoryWanderer    = "wanderer"
)

// What a storyteller watches for, and how often it may speak.
const (
	raidTasks        = 10
	raidWithin       = 10 * time.Minute
	inspirationTasks = 5
	inspirationIn    = 2 * time.Hour
	ColdSnapTasks    = 5
	ColdSnapAfter    = 7 * 24 * time.Hour
	letterGap        = 30 * time.Minute
	kindGap          = 24 * time.Hour
)

// StoryEvent is something the storyteller hears about a board.
type StoryEvent struct {
	// Kind is "task_created" (BugLike says whether it reads as a bug),
	// "task_done" (a task moved into the Done column), "newcomer" (a member
	// this machine never saw on the board opened it; Name is who) or
	// "cold_column" (Column held Count tasks, none moved for a week).
	Kind      string
	BoardID   uint64
	BoardName string
	At        time.Time
	BugLike   bool
	Name      string
	Column    string
	Count     int
}

// EventLetter is a one-off notice about a moment on a board. It never
// blocks anything; reading it is all it asks.
type EventLetter struct {
	ID      string    `json:"id"`
	Kind    string    `json:"kind"`
	BoardID uint64    `json:"board_id"`
	Title   string    `json:"title"`
	Text    string    `json:"text"`
	At      time.Time `json:"at"`
}

// Storyteller turns board events into event letters, at most one per
// board per half hour and one of each kind per board per day. It keeps
// only times; it does no I/O. The zero value is ready to use.
type Storyteller struct {
	bugs, done map[uint64][]time.Time
	lastLetter map[uint64]time.Time
	lastKind   map[string]time.Time
}

// BugLike reports whether a task reads as a bug: its work type or a word
// of its title is bug, bugs, fix or hotfix.
func BugLike(title, workType string) bool {
	words := []string{"bug", "bugs", "fix", "hotfix"}
	if slices.Contains(words, strings.ToLower(workType)) {
		return true
	}
	for _, w := range strings.FieldsFunc(strings.ToLower(title), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		if slices.Contains(words, w) {
			return true
		}
	}
	return false
}

// Observe hears one event and returns the letter it calls for, if any.
func (s *Storyteller) Observe(e StoryEvent) *EventLetter {
	if s.bugs == nil {
		s.bugs, s.done = map[uint64][]time.Time{}, map[uint64][]time.Time{}
		s.lastLetter, s.lastKind = map[uint64]time.Time{}, map[string]time.Time{}
	}
	switch e.Kind {
	case "task_created":
		if !e.BugLike {
			return nil
		}
		s.bugs[e.BoardID] = within(append(s.bugs[e.BoardID], e.At), e.At, raidWithin)
		if len(s.bugs[e.BoardID]) < raidTasks {
			return nil
		}
		return s.tell(e, StoryRaid, len(s.bugs[e.BoardID]), func() { s.bugs[e.BoardID] = nil })
	case "task_done":
		s.done[e.BoardID] = within(append(s.done[e.BoardID], e.At), e.At, inspirationIn)
		if len(s.done[e.BoardID]) < inspirationTasks {
			return nil
		}
		return s.tell(e, StoryInspiration, len(s.done[e.BoardID]), func() { s.done[e.BoardID] = nil })
	case "newcomer":
		return s.tell(e, StoryWanderer, 0, nil)
	case "cold_column":
		if e.Count < ColdSnapTasks {
			return nil
		}
		return s.tell(e, StoryColdSnap, e.Count, nil)
	}
	return nil
}

// tell writes the letter, unless the board has had one lately or this
// kind today. spent runs when the letter goes out.
func (s *Storyteller) tell(e StoryEvent, kind string, n int, spent func()) *EventLetter {
	key := fmt.Sprintf("%d:%s", e.BoardID, kind)
	if at, ok := s.lastLetter[e.BoardID]; ok && e.At.Sub(at) < letterGap {
		return nil
	}
	if at, ok := s.lastKind[key]; ok && e.At.Sub(at) < kindGap {
		return nil
	}
	s.lastLetter[e.BoardID], s.lastKind[key] = e.At, e.At
	if spent != nil {
		spent()
	}
	id := fmt.Sprintf("%s-%d-%d", kind, e.BoardID, e.At.UnixNano())
	title, text := story(kind, id, e, n)
	return &EventLetter{ID: id, Kind: kind, BoardID: e.BoardID, Title: title, Text: text, At: e.At}
}

// within keeps the times no older than d before now.
func within(times []time.Time, now time.Time, d time.Duration) []time.Time {
	return slices.DeleteFunc(times, func(t time.Time) bool { return now.Sub(t) > d })
}

// story is a letter's title and text: one of a few, picked by the letter's
// id so it does not change when shown again. Written for The Bakery.
func story(kind, id string, e StoryEvent, n int) (string, string) {
	board := e.BoardName
	if board == "" {
		board = "the board"
	}
	var title string
	var texts []string
	switch kind {
	case StoryRaid:
		title = "Raid!"
		texts = []string{
			fmt.Sprintf("%d bugs came over the hill onto %s in a few minutes. They look organised.", n, board),
			fmt.Sprintf("A pack of %d bugs has broken into %s. Nobody saw them coming, though everybody should have.", n, board),
			fmt.Sprintf("%s is under attack: %d bugs at once. Someone may want to put the kettle on.", board, n),
		}
	case StoryInspiration:
		title = "Inspiration"
		texts = []string{
			fmt.Sprintf("%d tasks reached Done on %s in the last two hours. The colony hums.", n, board),
			fmt.Sprintf("A good stretch on %s: %d tasks done in quick succession. Nobody quite knows why.", board, n),
			fmt.Sprintf("Something is working on %s. %d tasks done, and the day is not over.", board, n),
		}
	case StoryColdSnap:
		title = "Cold snap"
		texts = []string{
			fmt.Sprintf("Nothing has moved in %s on %s for a week. %d tasks sit there, frosting over.", e.Column, board, n),
			fmt.Sprintf("%s on %s has gone quiet: %d tasks, not one moved in seven days.", e.Column, board, n),
			fmt.Sprintf("A chill in %s. %d tasks have waited a week for anyone to touch them.", e.Column, n),
		}
	case StoryWanderer:
		title = "A wanderer joins"
		texts = []string{
			fmt.Sprintf("%s wandered onto %s for the first time. They look around, then stay.", e.Name, board),
			fmt.Sprintf("A newcomer: %s has opened %s. Show them where the good tasks are.", e.Name, board),
			fmt.Sprintf("%s arrived at %s with little more than good intentions.", e.Name, board),
		}
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return title, texts[int(h.Sum32())%len(texts)]
}
