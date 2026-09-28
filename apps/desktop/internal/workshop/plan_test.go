package workshop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestParseRecordedPlan(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := ParsePlanOutput(raw, []string{"coding", "design", "writing"}, "coding")
	if err != nil {
		t.Fatal(err)
	}
	p := res.Proposal
	if res.Error != "" || p.Plan == "" || len(p.Subtasks) != 3 || p.CostUSD <= 0 {
		t.Fatalf("result = %+v", res)
	}
	for _, s := range p.Subtasks {
		if s.Title == "" || !slices.Contains([]string{"coding", "design", "writing"}, s.WorkType) {
			t.Errorf("subtask = %+v", s)
		}
	}
}

func TestParsePlanCleansUp(t *testing.T) {
	out := `banner line
{"type":"result","subtype":"success","is_error":false,"total_cost_usd":0.1,"num_turns":2,"structured_output":{"plan":" Two steps. ","subtasks":[{"title":"  Dig  ","work_type":"digging"},{"title":"   ","work_type":"coding"},{"title":"Wall","work_type":"coding","description":" walls "}]}}`
	res, err := ParsePlanOutput([]byte(out), []string{"coding", "ops"}, "ops")
	if err != nil {
		t.Fatal(err)
	}
	want := []ProposedSub{{Title: "Dig", WorkType: "ops"}, {Title: "Wall", WorkType: "coding", Description: "walls"}}
	if res.Proposal.Plan != "Two steps." || !slices.Equal(res.Proposal.Subtasks, want) {
		t.Fatalf("proposal = %+v", res.Proposal)
	}
}

func TestParsePlanFailure(t *testing.T) {
	res, err := ParsePlanOutput([]byte(`{"type":"result","subtype":"error_max_budget_usd","is_error":true,"total_cost_usd":2}`), nil, "")
	if err != nil || res.Error == "" || res.Proposal.CostUSD != 2 {
		t.Fatalf("res = %+v, %v", res, err)
	}
	if _, err := ParsePlanOutput([]byte("nothing useful"), nil, ""); err == nil {
		t.Fatal("parsed nothing")
	}
}

func TestBuildPlanSpec(t *testing.T) {
	in := PlanInput{
		RunID: "r1", RunDir: "/tmp/r1", MCPURL: "http://x/mcp", Token: "bky_x",
		Task:      RunTask{ID: 7, Title: "Landing page", Description: "For the colony.", Subtasks: []RunSubtask{{Title: "Pick fonts"}}},
		Agent:     RunAgent{Name: "Vera", Model: "sonnet"},
		Board:     BoardConfig{Repo: "/home/ada/Projects/colony-site", MaxBudgetUSD: 1, IsolateUserSettings: true},
		WorkTypes: []string{"coding", "design"},
	}
	spec, err := BuildPlanSpec(in)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Join(spec.Args, " ")
	for _, want := range []string{"--permission-mode plan", "--output-format json", "--json-schema {", "--model sonnet", "--strict-mcp-config"} {
		if !strings.Contains(args, want) {
			t.Errorf("args lack %q", want)
		}
	}
	if spec.Dir != in.Board.Repo || !strings.Contains(spec.Stdin, "coding, design") || !strings.Contains(spec.Stdin, "- Pick fonts") {
		t.Errorf("spec = %+v", spec)
	}
	if strings.Contains(args, "Edit") || strings.Contains(args, "Bash") {
		t.Error("a plan may not edit or run commands")
	}
}
