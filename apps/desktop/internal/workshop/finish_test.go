package workshop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseShortstat(t *testing.T) {
	tests := map[string]DiffStats{
		" 3 files changed, 10 insertions(+), 2 deletions(-)": {3, 10, 2},
		" 1 file changed, 1 insertion(+)":                    {1, 1, 0},
		" 1 file changed, 4 deletions(-)":                    {1, 0, 4},
		"":                                                   {},
	}
	for in, want := range tests {
		if got := ParseShortstat(in); got != want {
			t.Errorf("ParseShortstat(%q) = %+v, want %+v", in, got, want)
		}
	}
}

func TestEndStatus(t *testing.T) {
	ok := &RunEvent{Kind: "result", Subtype: "success"}
	tests := []struct {
		name string
		out  Outcome
		want string
	}{
		{"success", Outcome{Result: ok}, "succeeded"},
		{"stopped wins", Outcome{Result: ok, Stopped: true}, "stopped"},
		{"error result", Outcome{Result: &RunEvent{Subtype: "success", IsError: true}}, "failed"},
		{"over budget", Outcome{Result: &RunEvent{Subtype: "error_max_budget_usd"}}, "failed"},
		{"no result", Outcome{ExitCode: 1}, "failed"},
	}
	for _, tt := range tests {
		if got := EndStatus(tt.out); got != tt.want {
			t.Errorf("%s: EndStatus() = %s, want %s", tt.name, got, tt.want)
		}
	}
}

func TestSummary(t *testing.T) {
	if got := Summary(Outcome{LastText: "last words", Result: &RunEvent{Text: "  the result  "}}); got != "the result" {
		t.Errorf("Summary() = %q", got)
	}
	if got := Summary(Outcome{LastText: "last words"}); got != "last words" {
		t.Errorf("Summary() without a result = %q", got)
	}
	if got := Summary(Outcome{LastText: strings.Repeat("a", 3000)}); len([]rune(got)) != summaryMax+1 {
		t.Errorf("Summary() is %d characters", len([]rune(got)))
	}
}

func TestFinishCommentGolden(t *testing.T) {
	tests := map[string]RunReport{
		"comment-succeeded": {
			AgentName: "Vera", Branch: "bakery/12-wall-in-the-freezer", Status: "succeeded",
			Diff: WorktreeDiff{Committed: DiffStats{6, 120, 14}}, CostUSD: 0.4234,
			Summary: "Walls up; the cooler is placed.\n\nLeft: power.",
		},
		"comment-uncommitted": {
			AgentName: "Pip", Branch: "bakery/3-plant-rice", Status: "succeeded",
			Diff: WorktreeDiff{Committed: DiffStats{}, Uncommitted: DiffStats{Files: 1, Additions: 3}}, CostUSD: 0.02,
			Summary: "Planted.",
		},
		"comment-stopped": {
			AgentName: "Vera", Branch: "bakery/12-wall-in-the-freezer", Status: "stopped",
			Diff: WorktreeDiff{Committed: DiffStats{1, 2, 0}}, CostUSD: 0.1, Problem: "Stopped by Ada.",
		},
	}
	for name, r := range tests {
		t.Run(name, func(t *testing.T) {
			got := FinishComment(r)
			path := filepath.Join("testdata", name+".golden")
			if *update {
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run with -update to write it)", err)
			}
			if got != string(want) {
				t.Errorf("comment differs from %s:\n%s", path, got)
			}
		})
	}
}

func TestDiff(t *testing.T) {
	cfg := linked(t)
	wt, err := PrepareWorktree(t.Context(), cfg, 8, "Diff me")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt.Path, "a.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOut(t, wt.Path, "add", "a.txt")
	gitOut(t, wt.Path, "-c", "user.name=T", "-c", "user.email=t@bakery.test", "commit", "-qm", "a")
	if err := os.WriteFile(filepath.Join(wt.Path, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt.Path, "new.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Diff(t.Context(), wt)
	if err != nil {
		t.Fatal(err)
	}
	if d.Committed != (DiffStats{1, 2, 0}) {
		t.Errorf("committed = %+v", d.Committed)
	}
	if d.Uncommitted != (DiffStats{2, 0, 1}) {
		t.Errorf("uncommitted = %+v", d.Uncommitted)
	}
}

func TestStateAfter(t *testing.T) {
	tests := []struct {
		ev   RunEvent
		want string
	}{
		{RunEvent{Kind: "init"}, "thinking"},
		{RunEvent{Kind: "text"}, "thinking"},
		{RunEvent{Kind: "tool_call", Tool: "Edit"}, "editing"},
		{RunEvent{Kind: "tool_call", Tool: "Write"}, "editing"},
		{RunEvent{Kind: "tool_call", Tool: "Bash"}, "running"},
		{RunEvent{Kind: "tool_call", Tool: "Read"}, "thinking"},
		{RunEvent{Kind: "tool_result"}, "thinking"},
		{RunEvent{Kind: "note"}, ""},
		{RunEvent{Kind: "result"}, ""},
	}
	for _, tt := range tests {
		if got := StateAfter("editing", tt.ev); got != tt.want {
			t.Errorf("StateAfter(%s %s) = %q, want %q", tt.ev.Kind, tt.ev.Tool, got, tt.want)
		}
	}
}
