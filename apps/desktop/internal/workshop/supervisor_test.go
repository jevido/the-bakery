package workshop

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update-supervisor", false, "rewrite testdata/supervisor-*.golden")

func supervisorSnap() SupervisorSnapshot {
	return SupervisorSnapshot{
		Board: "Getting settled", ReadyColumn: "To do", WorkTypes: []string{"coding", "research"},
		Columns: []SupervisorColumn{
			{Name: "Backlog", Tasks: []SupervisorTask{{ID: 1, Title: "Build the landing page", WorkType: "coding"}}},
			{Name: "To do", Tasks: []SupervisorTask{
				{ID: 2, Title: "Chart the river", WorkType: "coding"},
				{ID: 3, Title: "Count the moons", WorkType: "research"},
				{ID: 4, Title: "Hold the gate", WorkType: "coding", Forbidden: true},
			}},
		},
		Agents: []SupervisorAgent{
			{Slug: "moss", Name: "Moss", WorkPriorities: map[string]int{"coding": 1}},
			{Slug: "pip", Name: "Pip", WorkPriorities: map[string]int{"coding": 2, "research": 1}, Busy: true},
		},
	}
}

func TestSupervisorSpecGolden(t *testing.T) {
	for _, resume := range []bool{false, true} {
		spec, err := BuildSupervisorSpec(SupervisorInput{Snapshot: supervisorSnap(), Message: "What's next?", Model: "sonnet",
			Session: "0f6c1a2e-6d3b-4c8a-9a51-1f0d2b3c4e5f", Resume: resume, Dir: "/boards/1"})
		if err != nil {
			t.Fatal(err)
		}
		got := strings.Join(spec.Args, "\n") + "\n--- dir: " + spec.Dir + "\n--- stdin:\n" + spec.Stdin
		name := filepath.Join("testdata", map[bool]string{false: "supervisor-first.golden", true: "supervisor-resume.golden"}[resume])
		if *updateGolden {
			if err := os.WriteFile(name, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Errorf("%s differs:\n%s", name, got)
		}
		if !slices.Contains(spec.Args, "--tools") || spec.Args[slices.Index(spec.Args, "--tools")+1] != "" {
			t.Error("the supervisor has tools")
		}
	}
	if _, err := BuildSupervisorSpec(SupervisorInput{Session: "s", Dir: "d", Model: "m", Message: "  "}); err == nil {
		t.Error("an empty message was accepted")
	}
}

func TestParseRecordedSupervisor(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "supervisor.json"))
	if err != nil {
		t.Fatal(err)
	}
	ans, err := ParseSupervisorOutput(raw, supervisorSnap())
	if err != nil {
		t.Fatal(err)
	}
	if ans.Error != "" || ans.Reply == "" || ans.CostUSD <= 0 || len(ans.Proposals) == 0 {
		t.Fatalf("answer = %+v", ans)
	}
}

func TestSupervisorKeepsOnlyFittingProposals(t *testing.T) {
	out := `{"type":"result","subtype":"success","is_error":false,"total_cost_usd":0.02,"structured_output":{"reply":" Moss next. ","proposals":[
{"kind":"assign","task_id":2,"agent":"moss","why":" codes first "},
{"kind":"assign","task_id":4,"agent":"moss"},
{"kind":"assign","task_id":3,"agent":"moss"},
{"kind":"assign","task_id":1,"agent":"moss"},
{"kind":"assign","task_id":2,"agent":"nobody"},
{"kind":"split","task_id":1,"subtasks":[{"title":"Design","work_type":"design"},{"title":"  Build ","work_type":"coding"},{"title":" "}]},
{"kind":"split","task_id":1,"subtasks":[{"title":"Only one"}]},
{"kind":"split","task_id":99,"subtasks":[{"title":"a"},{"title":"b"}]},
{"kind":"reorder","task_ids":[3,2,4]},
{"kind":"reorder","task_ids":[2,3,4]},
{"kind":"reorder","task_ids":[3,2]},
{"kind":"burn"}]}}`
	ans, err := ParseSupervisorOutput([]byte(strings.ReplaceAll(out, "\n", "")), supervisorSnap())
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, p := range ans.Proposals {
		kinds = append(kinds, p.Kind)
	}
	if got := strings.Join(kinds, ","); got != "assign,split,reorder" {
		t.Fatalf("kinds = %s (%+v)", got, ans.Proposals)
	}
	if ans.Reply != "Moss next." || ans.Proposals[0].Why != "codes first" {
		t.Errorf("answer = %+v", ans)
	}
	split := ans.Proposals[1]
	want := []ProposedSub{{Title: "Design", WorkType: "coding"}, {Title: "Build", WorkType: "coding"}}
	if !slices.Equal(split.Subtasks, want) {
		t.Errorf("split = %+v", split.Subtasks)
	}
	if d := ans.Proposals[0].Describe(supervisorSnap()); d != "Moss takes “Chart the river”" {
		t.Errorf("describe = %q", d)
	}
}

func TestSupervisorError(t *testing.T) {
	ans, err := ParseSupervisorOutput([]byte(`{"type":"result","subtype":"error_during_execution","is_error":true,"result":"overloaded"}`), supervisorSnap())
	if err != nil || ans.Error != "overloaded" {
		t.Fatalf("answer = %+v, %v", ans, err)
	}
	if _, err := ParseSupervisorOutput([]byte("not json"), supervisorSnap()); err == nil {
		t.Fatal("no error for no answer")
	}
}
