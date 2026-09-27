package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
)

// The runner token is a personal token the app makes for itself, so Claude
// can use the Bakery MCP server as the member during a run. A personal token
// cannot make tokens, so it is made with the session's token, kept in its
// own store entry, and revoked on clock out.

type runnerToken struct {
	ID     uint64 `json:"id"`
	Secret string `json:"secret"`
}

// UseRunnerStore sets where the runner token is kept. Without one, runs
// cannot get a token.
func (s *Session) UseRunnerStore(st Store) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runner = st
}

// RunnerToken returns this machine's runner token, making one when there is
// none or the API no longer takes the one it has.
func (s *Session) RunnerToken(ctx context.Context) (string, error) {
	token := s.Token()
	s.mu.Lock()
	st := s.runner
	s.mu.Unlock()
	if token == "" {
		return "", errors.New("signed out")
	}
	if st == nil {
		return "", errors.New("no place to keep the runner token")
	}
	if rt, ok := loadRunner(st); ok {
		_, err := s.client.Me(ctx, rt.Secret)
		if err == nil {
			return rt.Secret, nil
		}
		if !errors.Is(err, api.ErrUnauthorized) {
			return "", err
		}
	}
	pt, secret, err := s.client.CreatePersonalToken(ctx, token, runnerName())
	if err != nil {
		return "", fmt.Errorf("making the runner token: %w", err)
	}
	raw, _ := json.Marshal(runnerToken{ID: pt.ID, Secret: secret})
	if err := st.Save(string(raw)); err != nil {
		return "", err
	}
	return secret, nil
}

// forgetRunner revokes the runner token (best effort: the API may be out of
// reach) and forgets it on this machine.
func (s *Session) forgetRunner(ctx context.Context, sessionToken string) error {
	s.mu.Lock()
	st := s.runner
	s.mu.Unlock()
	if st == nil {
		return nil
	}
	if rt, ok := loadRunner(st); ok && sessionToken != "" {
		_ = s.client.RevokePersonalToken(ctx, sessionToken, rt.ID)
	}
	return st.Delete()
}

func loadRunner(st Store) (runnerToken, bool) {
	raw, err := st.Load()
	if err != nil || raw == "" {
		return runnerToken{}, false
	}
	var rt runnerToken
	if json.Unmarshal([]byte(raw), &rt) != nil || rt.Secret == "" {
		return runnerToken{}, false
	}
	return rt, true
}

// runnerName says which machine a runner token belongs to, within the
// API's 60 characters.
func runnerName() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		host = "this machine"
	}
	if len(host) > 40 {
		host = host[:40]
	}
	return "desktop runner (" + host + ")"
}
