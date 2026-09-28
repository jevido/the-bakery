package workshop

import (
	"testing"
	"time"
)

var alertNow = time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)

func kinds(as []Alert) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.Kind
	}
	return out
}

func TestAlertKinds(t *testing.T) {
	board := func(mod func(*AlertBoard)) AlertBoard {
		b := AlertBoard{ID: 1, Name: "Colony A", WorkTypes: map[string]string{"coding": "Coding"},
			Agents: []AlertAgent{{Slug: "moss", Name: "Moss", Priorities: map[string]int{"coding": 1}}}}
		mod(&b)
		return b
	}
	tests := []struct {
		name string
		st   AlertState
		want []string
	}{
		{"nothing wrong", AlertState{Boards: []AlertBoard{board(func(*AlertBoard) {})}}, nil},
		{"idle long enough", AlertState{Boards: []AlertBoard{board(func(b *AlertBoard) { b.Agents[0].IdleSince = alertNow.Add(-6 * time.Minute) })}}, []string{AlertIdleAgent}},
		{"idle, but not five minutes yet", AlertState{Boards: []AlertBoard{board(func(b *AlertBoard) { b.Agents[0].IdleSince = alertNow.Add(-4 * time.Minute) })}}, nil},
		{"idle on a paused board is no alert", AlertState{Boards: []AlertBoard{board(func(b *AlertBoard) {
			b.Paused = true
			b.Agents[0].IdleSince = alertNow.Add(-time.Hour)
		})}}, nil},
		{"review waits over an hour", AlertState{Boards: []AlertBoard{board(func(b *AlertBoard) {
			b.Review = []AlertTask{{ID: 7, Title: "Carve sign", Since: alertNow.Add(-61 * time.Minute)}, {ID: 8, Since: alertNow.Add(-10 * time.Minute)}}
		})}}, []string{AlertAwaitingReview}},
		{"coding tasks and nobody codes", AlertState{Boards: []AlertBoard{board(func(b *AlertBoard) {
			b.Agents[0].Priorities = map[string]int{}
			b.Ready = []AlertTask{{ID: 1, WorkType: "coding"}, {ID: 2, WorkType: "coding"}, {ID: 3}}
		})}}, []string{AlertUncoveredWork}},
		{"a priority covers it", AlertState{Boards: []AlertBoard{board(func(b *AlertBoard) { b.Ready = []AlertTask{{ID: 1, WorkType: "coding"}} })}}, nil},
		{"a failed run not yet opened", AlertState{Runs: []AlertRun{
			{ID: "r1", Status: "failed", EndedAt: alertNow.Add(-time.Hour)},
			{ID: "r2", Status: "failed", EndedAt: alertNow.Add(-time.Hour), Seen: true},
			{ID: "r3", Status: "failed", EndedAt: alertNow.Add(-25 * time.Hour)},
			{ID: "r4", Status: "succeeded", EndedAt: alertNow.Add(-time.Hour)},
			{ID: "r5", Status: "running"},
		}}, []string{AlertRunFailed}},
		{"a question waits, first and high", AlertState{
			Runs:    []AlertRun{{ID: "r1", Status: "failed", EndedAt: alertNow.Add(-time.Minute)}},
			Letters: []AlertLetter{{ID: "l1", RunID: "r2", AgentName: "Moss", Kind: "question"}},
		}, []string{AlertQuestionWaiting, AlertRunFailed}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.st.Now = alertNow
			got := kinds(Alerts(tt.st))
			if len(got) != len(tt.want) {
				t.Fatalf("kinds = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("kinds = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestAlertIDsAreStable(t *testing.T) {
	st := AlertState{Now: alertNow, Letters: []AlertLetter{{ID: "l1", Kind: "question"}}}
	a, b := Alerts(st), Alerts(AlertState{Now: alertNow.Add(time.Minute), Letters: st.Letters})
	if a[0].ID != b[0].ID || a[0].Severity != "high" {
		t.Fatalf("ids %q and %q", a[0].ID, b[0].ID)
	}
}
