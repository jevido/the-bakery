package workshop

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// opened collects the letters a desk opens.
type opened struct {
	mu   sync.Mutex
	list []Letter
	seen chan Letter
}

func watch(d *Desk) *opened {
	o := &opened{seen: make(chan Letter, 8)}
	d.OnOpen = func(l Letter) {
		o.mu.Lock()
		o.list = append(o.list, l)
		o.mu.Unlock()
		o.seen <- l
	}
	return o
}

func (o *opened) next(t *testing.T) Letter {
	t.Helper()
	select {
	case l := <-o.seen:
		return l
	case <-time.After(5 * time.Second):
		t.Fatal("no letter")
		return Letter{}
	}
}

type result struct {
	reply []byte
	err   error
}

func ask(d *Desk, ctx context.Context, run, tool, input string) chan result {
	out := make(chan result, 1)
	go func() {
		r, err := d.Request(ctx, run, tool, json.RawMessage(input))
		out <- result{r, err}
	}()
	return out
}

func reply(t *testing.T, ch chan result) map[string]any {
	t.Helper()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatal(r.err)
		}
		var m map[string]any
		if err := json.Unmarshal(r.reply, &m); err != nil {
			t.Fatal(err)
		}
		return m
	case <-time.After(5 * time.Second):
		t.Fatal("no reply")
		return nil
	}
}

func TestDeskAllowAndDeny(t *testing.T) {
	d := NewDesk()
	o := watch(d)
	ch := ask(d, t.Context(), "run-1", "Bash", `{"command":"curl -sI https://example.com"}`)
	l := o.next(t)
	if l.Kind != "permission" || l.ToolName != "Bash" || l.RunID != "run-1" {
		t.Fatalf("letter = %+v", l)
	}
	if len(d.Open()) != 1 {
		t.Fatal("the letter is not open")
	}
	if err := d.Answer(l.ID, Answer{Behavior: "allow"}); err != nil {
		t.Fatal(err)
	}
	m := reply(t, ch)
	if m["behavior"] != "allow" || m["updatedInput"].(map[string]any)["command"] != "curl -sI https://example.com" {
		t.Fatalf("reply = %v", m)
	}
	if len(d.Open()) != 0 {
		t.Fatal("an answered letter is still open")
	}
	if err := d.Answer(l.ID, Answer{Behavior: "allow"}); err == nil {
		t.Fatal("answered twice")
	}

	ch = ask(d, t.Context(), "run-1", "Bash", `{"command":"rm -rf /"}`)
	l = o.next(t)
	_ = d.Answer(l.ID, Answer{Behavior: "deny", Message: "Not that."})
	if m := reply(t, ch); m["behavior"] != "deny" || m["message"] != "Not that." {
		t.Fatalf("reply = %v", m)
	}
}

func TestDeskQuestion(t *testing.T) {
	d := NewDesk()
	o := watch(d)
	ch := ask(d, t.Context(), "run-2", "AskUserQuestion",
		`{"questions":[{"question":"Which colour?","header":"Colour","multiSelect":false,"options":[{"label":"Red","description":""},{"label":"Blue","description":""}]}]}`)
	l := o.next(t)
	if l.Kind != "question" || len(l.Questions) != 1 || len(l.Questions[0].Options) != 2 {
		t.Fatalf("letter = %+v", l)
	}
	_ = d.Answer(l.ID, Answer{Behavior: "allow", Answers: map[string]string{"Which colour?": "Blue"}})
	m := reply(t, ch)
	in := m["updatedInput"].(map[string]any)
	if in["answers"].(map[string]any)["Which colour?"] != "Blue" || in["questions"] == nil {
		t.Fatalf("reply = %v", m)
	}
}

func TestDeskTimeoutAndCancel(t *testing.T) {
	d := NewDesk()
	d.Timeout = 50 * time.Millisecond
	o := watch(d)
	closed := make(chan Letter, 4)
	d.OnClose = func(l Letter) { closed <- l }
	ch := ask(d, t.Context(), "run-3", "WebFetch", `{"url":"https://example.com"}`)
	o.next(t)
	if m := reply(t, ch); m["behavior"] != "deny" || !strings.Contains(m["message"].(string), "in time") {
		t.Fatalf("reply = %v", m)
	}
	<-closed

	d.Timeout = time.Minute
	ch = ask(d, t.Context(), "run-4", "Bash", `{"command":"ls"}`)
	other := ask(d, t.Context(), "run-5", "Bash", `{"command":"ls"}`)
	o.next(t)
	o.next(t)
	d.CancelRun("run-4")
	if m := reply(t, ch); m["behavior"] != "deny" || !strings.Contains(m["message"].(string), "stopped") {
		t.Fatalf("reply = %v", m)
	}
	if len(d.Open()) != 1 || d.Open()[0].RunID != "run-5" {
		t.Fatalf("open = %+v", d.Open())
	}
	d.Stop()
	if m := reply(t, other); m["behavior"] != "deny" {
		t.Fatalf("reply after stop = %v", m)
	}
}

func TestRuleFor(t *testing.T) {
	tests := map[string]Letter{
		"Bash(curl:*)": {ToolName: "Bash", Input: json.RawMessage(`{"command":"curl -sI https://example.com"}`)},
		"WebFetch":     {ToolName: "WebFetch", Input: json.RawMessage(`{"url":"x"}`)},
		"Bash":         {ToolName: "Bash", Input: json.RawMessage(`{}`)},
	}
	for want, l := range tests {
		if got := RuleFor(l); got != want {
			t.Errorf("RuleFor(%s) = %s, want %s", l.Input, got, want)
		}
	}
}

// TestDeskOverHTTP calls approve the way Claude does: over streamable HTTP
// with the desk's secret, on the run's path.
func TestDeskOverHTTP(t *testing.T) {
	d := NewDesk()
	o := watch(d)
	if err := d.Start(); err != nil {
		t.Fatal(err)
	}
	defer d.Stop()
	cfg := d.ServerConfig("run-6")
	headers := cfg["headers"].(map[string]string)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: cfg["url"].(string), HTTPClient: &http.Client{Transport: headerTransport(headers)}}
	session, err := client.Connect(t.Context(), transport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	done := make(chan *mcp.CallToolResult, 1)
	go func() {
		r, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "approve", Arguments: map[string]any{
			"tool_name": "Bash", "input": map[string]any{"command": "ls"}, "tool_use_id": "toolu_1",
		}})
		if err != nil {
			t.Error(err)
			return
		}
		done <- r
	}()
	l := o.next(t)
	if l.RunID != "run-6" {
		t.Fatalf("letter for run %q", l.RunID)
	}
	_ = d.Answer(l.ID, Answer{Behavior: "allow"})
	select {
	case r := <-done:
		text := r.Content[0].(*mcp.TextContent).Text
		if !strings.Contains(text, `"behavior":"allow"`) {
			t.Fatalf("reply = %s", text)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no reply over HTTP")
	}

	// Without the secret: refused.
	bad := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	if _, err := bad.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: cfg["url"].(string)}, nil); err == nil {
		t.Fatal("connected without the secret")
	}
}

type headerTransport map[string]string

func (h headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	for k, v := range h {
		r.Header.Set(k, v)
	}
	return http.DefaultTransport.RoundTrip(r)
}
