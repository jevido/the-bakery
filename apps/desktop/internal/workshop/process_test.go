//go:build !windows

package workshop

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDecodeRecordedRun(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "stream.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	var init, result RunEvent
	notJSON := 0
	for line := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
		events, ok := DecodeLine([]byte(line))
		if !ok {
			notJSON++
		}
		for _, ev := range events {
			kinds = append(kinds, ev.Kind)
			switch ev.Kind {
			case "init":
				init = ev
			case "result":
				result = ev
			}
		}
	}
	if notJSON != 1 {
		t.Errorf("%d lines were not JSON, want 1 (the version manager's banner)", notJSON)
	}
	if got := strings.Join(kinds, ","); got != "init,text,tool_call,tool_result,text" && got != "init,tool_call,tool_result,text,result" {
		// The exact shape depends on the recording; the ends are what matter.
		if !strings.HasPrefix(got, "init,") || !strings.HasSuffix(got, ",result") || !strings.Contains(got, "tool_call,tool_result") {
			t.Errorf("kinds = %s", got)
		}
	}
	if init.Model == "" || init.PermissionMode == "" || len(init.Skills) == 0 {
		t.Errorf("init = %+v", init)
	}
	if result.Subtype != "success" || result.IsError || result.CostUSD <= 0 || result.Turns == 0 || !strings.Contains(result.Text, "Colony site") {
		t.Errorf("result = %+v", result)
	}
}

func TestDecodeToolCallAndCut(t *testing.T) {
	long := strings.Repeat("é", toolTextMax)
	line := `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":[{"type":"text","text":"` + long + `"}],"is_error":true}]}}`
	events, ok := DecodeLine([]byte(line))
	if !ok || len(events) != 1 {
		t.Fatalf("events = %+v", events)
	}
	ev := events[0]
	if ev.Kind != "tool_result" || ev.ToolID != "t1" || !ev.IsError || len(ev.Text) > toolTextMax+len("…") || !strings.HasSuffix(ev.Text, "…") {
		t.Errorf("tool result = %q…", ev.Text[:20])
	}
	events, _ = DecodeLine([]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t2","name":"Bash","input":{"command":"go test ./..."}}]}}`))
	if len(events) != 1 || events[0].Tool != "Bash" || events[0].Input != `{"command":"go test ./..."}` {
		t.Errorf("tool call = %+v", events)
	}
	for _, unknown := range []string{`{"type":"rate_limit_event"}`, `{"type":"system","subtype":"hook_started"}`, `{"type":"something_new","x":1}`} {
		if events, ok := DecodeLine([]byte(unknown)); !ok || len(events) != 0 {
			t.Errorf("%s gave %+v, %v", unknown, events, ok)
		}
	}
}

// fakeClaude writes a script standing in for claude: it prints the lines,
// then sleeps if asked, then exits with code.
func fakeClaude(t *testing.T, lines []string, sleep string, code int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claude")
	var b strings.Builder
	b.WriteString("#!/bin/sh\ncat > /dev/null\n")
	for _, l := range lines {
		b.WriteString("printf '%s\\n' '" + strings.ReplaceAll(l, "'", `'\''`) + "'\n")
	}
	b.WriteString("echo 'some warning' >&2\n")
	if sleep != "" {
		// In the foreground, so the interrupt reaches the sleep too.
		b.WriteString("trap 'exit 130' INT\ni=0\nwhile [ $i -lt " + sleep + "0 ]; do sleep 0.1; i=$((i+1)); done\n")
	}
	b.WriteString("exit " + string(rune('0'+code)) + "\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

type collected struct {
	mu     sync.Mutex
	events []RunEvent
}

func (c *collected) add(ev RunEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ev)
}

const (
	initLine   = `{"type":"system","subtype":"init","model":"claude-haiku","permissionMode":"default","skills":["bakery-vera-go-tests"],"mcp_servers":[{"name":"bakery","status":"connected"}]}`
	textLine   = `{"type":"assistant","message":{"content":[{"type":"text","text":"Walls up."}]}}`
	resultLine = `{"type":"result","subtype":"success","is_error":false,"total_cost_usd":0.42,"num_turns":3,"duration_ms":1200,"result":"Walls up; cooler placed."}`
)

func TestProcessRunsToTheEnd(t *testing.T) {
	claude := fakeClaude(t, []string{initLine, textLine, resultLine}, "", 0)
	dir := t.TempDir()
	var got collected
	p, err := StartProcess(t.Context(), claude, RunSpec{Dir: dir, Stdin: "the prompt"}, filepath.Join(dir, "run"), got.add)
	if err != nil {
		t.Fatal(err)
	}
	out := p.Wait()
	if out.Result == nil || out.Result.CostUSD != 0.42 || out.Stopped || out.ExitCode != 0 || out.LastText != "Walls up." {
		t.Fatalf("outcome = %+v", out)
	}
	if out.Stderr != "some warning" {
		t.Errorf("stderr = %q", out.Stderr)
	}
	if len(got.events) != 3 || got.events[0].Kind != "init" || got.events[2].Kind != "result" {
		t.Errorf("events = %+v", got.events)
	}
	logged, _ := os.ReadFile(filepath.Join(dir, "run", "stream.jsonl"))
	if strings.Count(string(logged), "\n") != 3 {
		t.Errorf("stream.jsonl:\n%s", logged)
	}
}

func TestProcessStops(t *testing.T) {
	claude := fakeClaude(t, []string{initLine}, "30", 0)
	dir := t.TempDir()
	var got collected
	p, err := StartProcess(t.Context(), claude, RunSpec{Dir: dir}, filepath.Join(dir, "run"), got.add)
	if err != nil {
		t.Fatal(err)
	}
	// Wait for the init line, so the script is sleeping.
	deadline := time.Now().Add(5 * time.Second)
	for {
		got.mu.Lock()
		n := len(got.events)
		got.mu.Unlock()
		if n > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	start := time.Now()
	p.Stop()
	out := p.Wait()
	if !out.Stopped || out.Result != nil {
		t.Fatalf("outcome = %+v", out)
	}
	// An interrupt is enough; the kill after stopGrace is not needed.
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("stopping took %s", took)
	}
}

func TestProcessFailsWithoutResult(t *testing.T) {
	claude := fakeClaude(t, []string{initLine}, "", 3)
	dir := t.TempDir()
	p, err := StartProcess(t.Context(), claude, RunSpec{Dir: dir}, filepath.Join(dir, "run"), func(RunEvent) {})
	if err != nil {
		t.Fatal(err)
	}
	if out := p.Wait(); out.ExitCode != 3 || out.Result != nil || out.Stopped {
		t.Fatalf("outcome = %+v", out)
	}
}

func TestProcessWithoutClaude(t *testing.T) {
	dir := t.TempDir()
	_, err := StartProcess(t.Context(), filepath.Join(dir, "no-such-claude"), RunSpec{Dir: dir}, filepath.Join(dir, "run"), func(RunEvent) {})
	if err == nil {
		t.Fatal("started a claude that does not exist")
	}
}
