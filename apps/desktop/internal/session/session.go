package session

import (
	"context"
	"errors"
	"sync"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
)

// Session is the signed-in member, if any, and their token. It is safe for
// concurrent use.
type Session struct {
	client *api.Client
	store  Store

	mu     sync.Mutex
	token  string
	member *api.Member
}

func New(client *api.Client, store Store) *Session {
	return &Session{client: client, store: store}
}

// Restore loads a stored token and checks it with the API. A token the API
// refuses is dropped; an unreachable API keeps it for the next try.
func (s *Session) Restore(ctx context.Context) error {
	token, err := s.store.Load()
	if err != nil || token == "" {
		return err
	}
	m, err := s.client.Me(ctx, token)
	if errors.Is(err, api.ErrUnauthorized) {
		return s.store.Delete()
	}
	if err != nil {
		return err
	}
	s.set(token, &m)
	return nil
}

func (s *Session) Register(ctx context.Context, email, displayName, password string) (api.Member, error) {
	token, m, err := s.client.Register(ctx, email, displayName, password)
	if err != nil {
		return api.Member{}, err
	}
	return m, s.keep(token, m)
}

func (s *Session) Login(ctx context.Context, email, password string) (api.Member, error) {
	token, m, err := s.client.Login(ctx, email, password)
	if err != nil {
		return api.Member{}, err
	}
	return m, s.keep(token, m)
}

// Logout forgets the token on this machine.
func (s *Session) Logout() error {
	s.set("", nil)
	return s.store.Delete()
}

// Member is the signed-in member, or nil.
func (s *Session) Member() *api.Member {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.member == nil {
		return nil
	}
	m := *s.member
	return &m
}

// Token is the signed-in member's token, or "".
func (s *Session) Token() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

func (s *Session) keep(token string, m api.Member) error {
	s.set(token, &m)
	return s.store.Save(token)
}

func (s *Session) set(token string, m *api.Member) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token, s.member = token, m
}
