package workshop

import (
	"testing"
	"time"
)

var storyNow = time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)

func bug(board uint64, at time.Time) StoryEvent {
	return StoryEvent{Kind: "task_created", BoardID: board, BoardName: "Colony A", At: at, BugLike: true}
}

func TestRaid(t *testing.T) {
	var s Storyteller
	var letters []*EventLetter
	for i := range 10 {
		if l := s.Observe(bug(1, storyNow.Add(time.Duration(i)*time.Second))); l != nil {
			letters = append(letters, l)
		}
	}
	if len(letters) != 1 || letters[0].Kind != StoryRaid || letters[0].Title != "Raid!" {
		t.Fatalf("letters = %+v", letters)
	}
	// Ten more within half an hour: quiet.
	for i := range 10 {
		if l := s.Observe(bug(1, storyNow.Add(time.Minute+time.Duration(i)*time.Second))); l != nil {
			t.Fatalf("a second raid letter within 30 minutes: %+v", l)
		}
	}
	// Another board is its own story.
	var other *EventLetter
	for i := range 10 {
		if l := s.Observe(bug(2, storyNow.Add(time.Duration(i)*time.Second))); l != nil {
			other = l
		}
	}
	if other == nil {
		t.Fatal("no raid on board 2")
	}
}

func TestRaidNeedsBugsCloseTogether(t *testing.T) {
	var s Storyteller
	for i := range 12 {
		// One every two minutes: never ten within ten minutes.
		if l := s.Observe(bug(1, storyNow.Add(time.Duration(i)*2*time.Minute))); l != nil {
			t.Fatalf("raid from spread-out bugs: %+v", l)
		}
	}
	plain := StoryEvent{Kind: "task_created", BoardID: 3, At: storyNow}
	for range 20 {
		if s.Observe(plain) != nil {
			t.Fatal("raid from tasks that are not bugs")
		}
	}
}

func TestInspiration(t *testing.T) {
	var s Storyteller
	var got *EventLetter
	for i := range 5 {
		got = s.Observe(StoryEvent{Kind: "task_done", BoardID: 1, At: storyNow.Add(time.Duration(i) * 20 * time.Minute)})
	}
	if got == nil || got.Kind != StoryInspiration {
		t.Fatalf("got %+v", got)
	}
	var s2 Storyteller
	for i := range 5 {
		// Spread over five hours: not inspiring.
		if l := s2.Observe(StoryEvent{Kind: "task_done", BoardID: 1, At: storyNow.Add(time.Duration(i) * time.Hour)}); l != nil {
			t.Fatalf("inspiration from a slow day: %+v", l)
		}
	}
}

func TestColdSnapAndWanderer(t *testing.T) {
	var s Storyteller
	if s.Observe(StoryEvent{Kind: "cold_column", BoardID: 1, Column: "Doing", Count: 4, At: storyNow}) != nil {
		t.Fatal("cold snap from four tasks")
	}
	l := s.Observe(StoryEvent{Kind: "cold_column", BoardID: 1, BoardName: "Colony A", Column: "Doing", Count: 6, At: storyNow})
	if l == nil || l.Kind != StoryColdSnap {
		t.Fatalf("got %+v", l)
	}
	// Half an hour later a newcomer is news; the next day's cold snap too,
	// but not a second one the same day.
	w := s.Observe(StoryEvent{Kind: "newcomer", BoardID: 1, Name: "Bram", At: storyNow.Add(31 * time.Minute)})
	if w == nil || w.Kind != StoryWanderer {
		t.Fatalf("wanderer = %+v", w)
	}
	if s.Observe(StoryEvent{Kind: "cold_column", BoardID: 1, Column: "Doing", Count: 6, At: storyNow.Add(2 * time.Hour)}) != nil {
		t.Fatal("two cold snaps in a day")
	}
	if s.Observe(StoryEvent{Kind: "cold_column", BoardID: 1, Column: "Doing", Count: 6, At: storyNow.Add(25 * time.Hour)}) == nil {
		t.Fatal("no cold snap the next day")
	}
}

func TestLetterGapAcrossKinds(t *testing.T) {
	var s Storyteller
	if s.Observe(StoryEvent{Kind: "newcomer", BoardID: 1, Name: "Bram", At: storyNow}) == nil {
		t.Fatal("no wanderer")
	}
	if s.Observe(StoryEvent{Kind: "cold_column", BoardID: 1, Count: 9, At: storyNow.Add(10 * time.Minute)}) != nil {
		t.Fatal("a second letter within half an hour")
	}
}

func TestBugLike(t *testing.T) {
	for _, tt := range []struct {
		title, workType string
		want            bool
	}{
		{"Fix the login", "", true},
		{"Bug: sidebar overlaps", "", true},
		{"Hotfix for the build", "", true},
		{"Debug toolbar", "", false},
		{"Prefix routes", "", false},
		{"Anything", "bug", true},
		{"Write the docs", "writing", false},
	} {
		if got := BugLike(tt.title, tt.workType); got != tt.want {
			t.Errorf("BugLike(%q, %q) = %v", tt.title, tt.workType, got)
		}
	}
}

func TestStoryIsStable(t *testing.T) {
	e := StoryEvent{BoardName: "Colony A", Name: "Bram"}
	a, _ := story(StoryWanderer, "x-1", e, 0)
	_, t1 := story(StoryWanderer, "x-1", e, 0)
	_, t2 := story(StoryWanderer, "x-1", e, 0)
	if a == "" || t1 != t2 {
		t.Fatal("story text changes between calls")
	}
}
