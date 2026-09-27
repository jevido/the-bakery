package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
)

type memStore struct{ token string }

func (m *memStore) Load() (string, error)   { return m.token, nil }
func (m *memStore) Save(token string) error { m.token = token; return nil }
func (m *memStore) Delete() error           { m.token = ""; return nil }

func fakeAPI(t *testing.T) *api.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/login":
			w.Write([]byte(`{"token":"good","member":{"id":1,"email":"ada@bakery.test","display_name":"Ada"}}`))
		case "/api/me":
			if r.Header.Get("Authorization") != "Bearer good" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(`{"member":{"id":1,"email":"ada@bakery.test","display_name":"Ada"}}`))
		}
	}))
	t.Cleanup(srv.Close)
	return api.New(srv.URL)
}

func TestLoginRestoreLogout(t *testing.T) {
	client := fakeAPI(t)
	store := &memStore{}

	s := New(client, store)
	if _, err := s.Login(t.Context(), "ada@bakery.test", "password"); err != nil {
		t.Fatal(err)
	}
	if store.token != "good" {
		t.Fatalf("stored token = %q", store.token)
	}

	// A new launch restores the member from the stored token.
	s = New(client, store)
	if err := s.Restore(t.Context()); err != nil {
		t.Fatal(err)
	}
	if m := s.Member(); m == nil || m.DisplayName != "Ada" {
		t.Fatalf("restored member = %+v", m)
	}

	if err := s.Logout(t.Context()); err != nil {
		t.Fatal(err)
	}
	if s.Member() != nil || store.token != "" {
		t.Fatal("logout kept the session")
	}
}

func TestRestoreDropsRefusedToken(t *testing.T) {
	store := &memStore{token: "expired"}
	s := New(fakeAPI(t), store)
	if err := s.Restore(t.Context()); err != nil {
		t.Fatal(err)
	}
	if s.Member() != nil || store.token != "" {
		t.Fatal("refused token was kept")
	}
}

func TestFileStore(t *testing.T) {
	f := fileStore{path: t.TempDir() + "/sub/token"}
	if tok, err := f.Load(); err != nil || tok != "" {
		t.Fatalf("empty Load() = %q, %v", tok, err)
	}
	if err := f.Save("abc"); err != nil {
		t.Fatal(err)
	}
	if tok, _ := f.Load(); tok != "abc" {
		t.Fatalf("Load() = %q", tok)
	}
	if err := f.Delete(); err != nil {
		t.Fatal(err)
	}
	if err := f.Delete(); err != nil {
		t.Fatalf("second Delete() = %v", err)
	}
}
