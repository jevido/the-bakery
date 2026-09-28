package workshop

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// stopGrace is how long a stopped run gets to end by itself before it is
// killed.
const stopGrace = 5 * time.Second

// maxLine is the longest stream-json line read; a tool result can be large.
const maxLine = 10 << 20

// Process is one running Claude CLI.
type Process struct {
	cmd  *exec.Cmd
	done chan struct{}

	mu       sync.Mutex
	stopped  bool
	result   *RunEvent
	lastText string
	outcome  Outcome
}

// Outcome is how a process ended.
type Outcome struct {
	// Result is Claude's final result event, nil when it never sent one.
	Result *RunEvent
	// Stopped is true when the member stopped the run.
	Stopped bool
	// ExitCode is the process's exit code, -1 when it was killed.
	ExitCode int
	// LastText is the last text Claude wrote, for a run without a result.
	LastText string
	// Stderr is the end of what Claude wrote to stderr.
	Stderr string
}

// StartProcess starts claude (the binary to run: "claude", or a stand-in in
// tests) with the spec, in its own process group. Every line of its output
// goes to runDir/stream.jsonl and, decoded, to onEvent; stderr goes to
// runDir/stderr.log. onEvent is called from one goroutine, in order.
func StartProcess(ctx context.Context, claude string, spec RunSpec, runDir string, onEvent func(RunEvent)) (*Process, error) {
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return nil, err
	}
	streamLog, err := os.OpenFile(filepath.Join(runDir, "stream.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	stderrLog, err := os.OpenFile(filepath.Join(runDir, "stderr.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		streamLog.Close()
		return nil, err
	}
	cmd := exec.Command(claude, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Stdin = strings.NewReader(spec.Stdin)
	ownProcessGroup(cmd)
	var stderrTail tailBuffer
	cmd.Stderr = io.MultiWriter(stderrLog, &stderrTail)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		streamLog.Close()
		stderrLog.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		streamLog.Close()
		stderrLog.Close()
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("claude is not installed or not on the PATH: %w", err)
		}
		return nil, err
	}
	p := &Process{cmd: cmd, done: make(chan struct{})}
	go func() {
		defer close(p.done)
		defer streamLog.Close()
		defer stderrLog.Close()
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 64<<10), maxLine)
		for sc.Scan() {
			line := sc.Bytes()
			_, _ = streamLog.Write(append(bytes.Clone(line), '\n'))
			events, _ := DecodeLine(line)
			for _, ev := range events {
				p.mu.Lock()
				switch ev.Kind {
				case "result":
					r := ev
					p.result = &r
				case "text":
					p.lastText = ev.Text
				}
				p.mu.Unlock()
				onEvent(ev)
			}
		}
		// A line past maxLine ends the scan; drain the rest so claude is
		// not blocked writing.
		_, _ = io.Copy(io.Discard, stdout)
		err := cmd.Wait()
		code := 0
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		} else if err != nil {
			code = -1
		}
		p.mu.Lock()
		p.outcome = Outcome{Result: p.result, Stopped: p.stopped, ExitCode: code, LastText: p.lastText, Stderr: stderrTail.String()}
		p.mu.Unlock()
	}()
	// The context ends the run too, as a stop.
	go func() {
		select {
		case <-ctx.Done():
			p.Stop()
		case <-p.done:
		}
	}()
	return p, nil
}

// Stop asks the run to end (an interrupt to its process group) and kills it
// when it has not ended within stopGrace.
func (p *Process) Stop() {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	p.stopped = true
	p.mu.Unlock()
	interrupt(p.cmd)
	go func() {
		select {
		case <-p.done:
		case <-time.After(stopGrace):
			kill(p.cmd)
		}
	}()
}

// PID is the process's id, which is also its process group's.
func (p *Process) PID() int { return p.cmd.Process.Pid }

// Done is closed when the process has ended.
func (p *Process) Done() <-chan struct{} { return p.done }

// Wait blocks until the process ends and says how.
func (p *Process) Wait() Outcome {
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.outcome
}

// tailBuffer keeps the last 4 KB written to it.
type tailBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (t *tailBuffer) Write(b []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, b...)
	if len(t.buf) > 4096 {
		t.buf = t.buf[len(t.buf)-4096:]
	}
	return len(b), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.TrimSpace(string(t.buf))
}
