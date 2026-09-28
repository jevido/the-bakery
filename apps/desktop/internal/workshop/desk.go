package workshop

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The desk is a small MCP server inside the app, on loopback only, whose
// one tool, approve, every run gets as Claude's --permission-prompt-tool. A
// permission request or a question from a run becomes a Letter, and the
// tool call waits until a person answers it.

// DeskServer and DeskTool name the desk in a run's MCP config and its
// --permission-prompt-tool.
const (
	DeskServer = "bakery_desk"
	DeskTool   = "mcp__bakery_desk__approve"
)

// LetterTimeout is how long a letter waits for an answer.
const LetterTimeout = 30 * time.Minute

// Letter is a run asking a person: may it use a tool, or which answer to a
// question.
type Letter struct {
	ID    string `json:"id"`
	RunID string `json:"run_id"`
	// Kind is "permission" or "question".
	Kind      string          `json:"kind"`
	ToolName  string          `json:"tool_name"`
	Input     json.RawMessage `json:"input"`
	Questions []Question      `json:"questions,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// Question is one question of an AskUserQuestion call.
type Question struct {
	Question    string   `json:"question"`
	Header      string   `json:"header"`
	MultiSelect bool     `json:"multiSelect"`
	Options     []Option `json:"options"`
}

type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Answer is a person's answer to a letter.
type Answer struct {
	// Behavior is "allow" or "deny".
	Behavior string `json:"behavior"`
	// Message says why, on a deny.
	Message string `json:"message"`
	// Answers answer a question letter: question text to chosen label(s),
	// several joined with ", ".
	Answers map[string]string `json:"answers"`
}

type openLetter struct {
	Letter
	reply chan Answer
}

// Desk holds the open letters and serves the approve tool.
type Desk struct {
	// OnOpen and OnClose hear a letter arrive and go (answered, timed out or
	// cancelled). Set them before Start.
	OnOpen  func(Letter)
	OnClose func(Letter)
	// Timeout is LetterTimeout unless a test sets it.
	Timeout time.Duration

	secret string
	url    string
	srv    *http.Server

	mu      sync.Mutex
	letters map[string]*openLetter
	servers map[string]*mcp.Server
}

func NewDesk() *Desk {
	return &Desk{Timeout: LetterTimeout, letters: map[string]*openLetter{}, servers: map[string]*mcp.Server{}, secret: randomHex(24)}
}

// Start listens on a random loopback port.
func (d *Desk) Start() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	d.url = "http://" + ln.Addr().String()
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return d.serverFor(strings.TrimPrefix(r.URL.Path, "/mcp/"))
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	d.srv = &http.Server{Handler: d.guard(handler), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = d.srv.Serve(ln) }()
	return nil
}

// Stop closes the server and denies every open letter.
func (d *Desk) Stop() {
	if d.srv != nil {
		_ = d.srv.Close()
	}
	d.mu.Lock()
	var all []*openLetter
	for _, l := range d.letters {
		all = append(all, l)
	}
	d.mu.Unlock()
	for _, l := range all {
		d.close(l, Answer{Behavior: "deny", Message: "The app closed."})
	}
}

// ServerConfig is the desk's entry for a run's --mcp-config.
func (d *Desk) ServerConfig(runID string) map[string]any {
	return map[string]any{
		"type":    "http",
		"url":     d.url + "/mcp/" + runID,
		"headers": map[string]string{"Authorization": "Bearer " + d.secret},
	}
}

// guard lets in only requests with the desk's secret, for a run id path.
func (d *Desk) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(d.secret)) != 1 || !strings.HasPrefix(r.URL.Path, "/mcp/") {
			http.Error(w, "not yours", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type approveIn struct {
	ToolName  string         `json:"tool_name"`
	Input     map[string]any `json:"input"`
	ToolUseID string         `json:"tool_use_id"`
}

// serverFor is the MCP server one run talks to; its approve tool files
// letters for that run.
func (d *Desk) serverFor(runID string) *mcp.Server {
	d.mu.Lock()
	defer d.mu.Unlock()
	if s, ok := d.servers[runID]; ok {
		return s
	}
	s := mcp.NewServer(&mcp.Implementation{Name: DeskServer, Title: "The Bakery desk", Version: "1"}, nil)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "approve",
		Description: "Asks the member at the desk whether a tool may run, or for answers to questions.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in approveIn) (*mcp.CallToolResult, any, error) {
		input, err := json.Marshal(in.Input)
		if err != nil {
			return nil, nil, err
		}
		reply, err := d.Request(ctx, runID, in.ToolName, input)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(reply)}}}, nil, nil
	})
	d.servers[runID] = s
	return s
}

// Request files a letter for a run and waits for its answer, the timeout,
// or ctx. It returns the reply Claude expects: allow with the (answered)
// input, or deny with a message.
func (d *Desk) Request(ctx context.Context, runID, toolName string, input json.RawMessage) ([]byte, error) {
	l := &openLetter{
		Letter: Letter{ID: randomHex(8), RunID: runID, Kind: "permission", ToolName: toolName, Input: input, CreatedAt: time.Now()},
		reply:  make(chan Answer, 1),
	}
	if toolName == "AskUserQuestion" {
		var q struct {
			Questions []Question `json:"questions"`
		}
		if json.Unmarshal(input, &q) == nil && len(q.Questions) > 0 {
			l.Kind, l.Questions = "question", q.Questions
		}
	}
	d.mu.Lock()
	d.letters[l.ID] = l
	d.mu.Unlock()
	if d.OnOpen != nil {
		d.OnOpen(l.Letter)
	}

	var a Answer
	timer := time.NewTimer(d.Timeout)
	defer timer.Stop()
	select {
	case a = <-l.reply:
	case <-timer.C:
		d.close(l, Answer{Behavior: "deny", Message: "Nobody answered in time."})
		a = <-l.reply
	case <-ctx.Done():
		d.close(l, Answer{Behavior: "deny", Message: "The run ended."})
		a = <-l.reply
	}
	return replyFor(l.Letter, a)
}

// Answer answers an open letter.
func (d *Desk) Answer(id string, a Answer) error {
	d.mu.Lock()
	l := d.letters[id]
	d.mu.Unlock()
	if l == nil {
		return errors.New("this letter was already answered or has expired")
	}
	if a.Behavior != "allow" && a.Behavior != "deny" {
		return fmt.Errorf("an answer allows or denies, not %q", a.Behavior)
	}
	d.close(l, a)
	return nil
}

// CancelRun denies the open letters of a run that is stopping.
func (d *Desk) CancelRun(runID string) {
	d.mu.Lock()
	var mine []*openLetter
	for _, l := range d.letters {
		if l.RunID == runID {
			mine = append(mine, l)
		}
	}
	delete(d.servers, runID)
	d.mu.Unlock()
	for _, l := range mine {
		d.close(l, Answer{Behavior: "deny", Message: "The run was stopped."})
	}
}

// Open lists the letters waiting for an answer, oldest first.
func (d *Desk) Open() []Letter {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]Letter, 0, len(d.letters))
	for _, l := range d.letters {
		out = append(out, l.Letter)
	}
	slices.SortFunc(out, func(a, b Letter) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out
}

// Letter returns an open letter.
func (d *Desk) Get(id string) (Letter, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	l, ok := d.letters[id]
	if !ok {
		return Letter{}, false
	}
	return l.Letter, true
}

// close removes a letter and hands it its answer, once.
func (d *Desk) close(l *openLetter, a Answer) {
	d.mu.Lock()
	_, open := d.letters[l.ID]
	delete(d.letters, l.ID)
	d.mu.Unlock()
	if !open {
		return
	}
	l.reply <- a
	if d.OnClose != nil {
		d.OnClose(l.Letter)
	}
}

// replyFor is the JSON Claude reads back from the permission prompt tool.
func replyFor(l Letter, a Answer) ([]byte, error) {
	if a.Behavior != "allow" {
		msg := strings.TrimSpace(a.Message)
		if msg == "" {
			msg = "The member said no."
		}
		return json.Marshal(map[string]any{"behavior": "deny", "message": msg})
	}
	input := map[string]any{}
	if len(l.Input) > 0 {
		if err := json.Unmarshal(l.Input, &input); err != nil {
			return nil, err
		}
	}
	if l.Kind == "question" {
		answers := map[string]any{}
		for q, label := range a.Answers {
			answers[q] = label
		}
		input["answers"] = answers
	}
	return json.Marshal(map[string]any{"behavior": "allow", "updatedInput": input})
}

// RuleFor is the allowed-tools rule "always allow" adds for a letter: a
// shell command by its first word (Bash(curl:*)), any other tool by name.
func RuleFor(l Letter) string {
	if l.ToolName == "Bash" {
		var in struct {
			Command string `json:"command"`
		}
		if json.Unmarshal(l.Input, &in) == nil {
			if fields := strings.Fields(in.Command); len(fields) > 0 {
				return "Bash(" + fields[0] + ":*)"
			}
		}
	}
	return l.ToolName
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
