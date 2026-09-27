package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
)

// tokenAPI is a fake API that makes and revokes personal tokens.
type tokenAPI struct {
	mu      sync.Mutex
	made    int
	live    map[string]bool
	revoked []string
}

func (f *tokenAPI) serve(t *testing.T) *api.Client {
	t.Helper()
	f.live = map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		switch {
		case r.URL.Path == "/api/login":
			w.Write([]byte(`{"token":"session","member":{"id":1,"display_name":"Ada"}}`))
		case r.URL.Path == "/api/me":
			if auth != "session" && !f.live[auth] {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(`{"member":{"id":1,"display_name":"Ada"}}`))
		case r.URL.Path == "/api/tokens" && r.Method == http.MethodPost:
			if auth != "session" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			f.made++
			secret := "bky_" + strings.Repeat("x", f.made)
			f.live[secret] = true
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"token":"` + secret + `","personal_token":{"id":7,"name":"desktop runner (x)"}}`))
		case strings.HasPrefix(r.URL.Path, "/api/tokens/") && r.Method == http.MethodDelete:
			f.revoked = append(f.revoked, strings.TrimPrefix(r.URL.Path, "/api/tokens/"))
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return api.New(srv.URL)
}

func TestRunnerToken(t *testing.T) {
	fake := &tokenAPI{}
	runner := &memStore{}
	s := New(fake.serve(t), &memStore{})
	s.UseRunnerStore(runner)
	if _, err := s.RunnerToken(t.Context()); err == nil {
		t.Fatal("got a runner token while signed out")
	}
	if _, err := s.Login(t.Context(), "ada@bakery.test", "password"); err != nil {
		t.Fatal(err)
	}

	first, err := s.RunnerToken(t.Context())
	if err != nil || first != "bky_x" {
		t.Fatalf("RunnerToken() = %q, %v", first, err)
	}
	// Kept: the next run reuses it.
	if again, _ := s.RunnerToken(t.Context()); again != first || fake.made != 1 {
		t.Fatalf("second RunnerToken() = %q after %d made", again, fake.made)
	}
	// Revoked on the website: a new one is made.
	fake.mu.Lock()
	delete(fake.live, first)
	fake.mu.Unlock()
	if next, _ := s.RunnerToken(t.Context()); next == first || fake.made != 2 {
		t.Fatalf("after revoking, RunnerToken() = %q after %d made", next, fake.made)
	}

	// Clock out revokes it and forgets it.
	if err := s.Logout(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(fake.revoked) != 1 || fake.revoked[0] != "7" {
		t.Fatalf("revoked = %v", fake.revoked)
	}
	if runner.token != "" {
		t.Fatal("the runner token is still stored")
	}
}

func TestRunnerName(t *testing.T) {
	if n := runnerName(); !strings.HasPrefix(n, "desktop runner (") || len(n) > 60 {
		t.Fatalf("runnerName() = %q", n)
	}
}
