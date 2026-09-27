// Package app holds the agents use cases. Only an agent's owner sees or
// changes it; to anyone else it does not exist.
package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/agents/domain"
)

var (
	ErrAgentNotFound = errors.New("agent not found")
	ErrSlugTaken     = errors.New("you already have an agent with that slug")
)

// Agents stores agents with their skillsets.
type Agents interface {
	// Add stores a new agent and its files in one transaction.
	Add(ctx context.Context, a domain.Agent) (domain.Agent, error)
	// Replace writes the agent (fields and files) in one transaction if it
	// is still at basedOn, and reports ErrStale otherwise.
	Replace(ctx context.Context, a domain.Agent, basedOn int) error
	// ByID returns the agent with its files, deleted or not.
	ByID(ctx context.Context, id uint64) (domain.Agent, bool, error)
	// OfOwner returns the owner's agents without files: live ones, or, when
	// since is set, every one changed since then, deleted ones included.
	OfOwner(ctx context.Context, ownerID uint64, since *time.Time) ([]domain.Agent, error)
}

// Events takes the domain events of a change once it is stored.
type Events interface {
	Publish(ctx context.Context, events ...any)
}

type Service struct {
	agents      Agents
	shares      Shares
	memberships Memberships
	names       MemberNames
	events      Events
	now         func() time.Time
}

func NewService(agents Agents, shares Shares, memberships Memberships, names MemberNames, events Events) *Service {
	return &Service{agents: agents, shares: shares, memberships: memberships, names: names, events: events, now: time.Now}
}

// Events the agents context announces.
type (
	AgentCreated struct{ AgentID, OwnerID uint64 }
	AgentRevised struct {
		AgentID, OwnerID uint64
		Revision         int
	}
	AgentDeleted struct{ AgentID, OwnerID uint64 }
)

// CreateAgent adds an agent to the member's roster. An empty slug is made
// from the name; an empty portrait seed gets a random one.
func (s *Service) CreateAgent(ctx context.Context, ownerID uint64, slug string, p domain.Profile) (domain.Agent, error) {
	if slug == "" {
		slug = domain.Slugify(p.Name)
	}
	if p.PortraitSeed == "" {
		p.PortraitSeed = randomSeed()
	}
	a, err := domain.NewAgent(ownerID, slug, p)
	if err != nil {
		return domain.Agent{}, err
	}
	if a, err = s.agents.Add(ctx, a); err != nil {
		return domain.Agent{}, err
	}
	s.events.Publish(ctx, AgentCreated{AgentID: a.ID, OwnerID: ownerID})
	return a, nil
}

// ReviseAgent replaces the whole agent, fields and files, if nobody changed
// it since basedOn. On a stale revision it returns ErrStale with the
// current agent, so the caller can show both.
func (s *Service) ReviseAgent(ctx context.Context, id, ownerID uint64, basedOn int, p domain.Profile) (domain.Agent, error) {
	a, err := s.own(ctx, id, ownerID)
	if err != nil {
		return domain.Agent{}, err
	}
	if p.PortraitSeed == "" {
		p.PortraitSeed = a.PortraitSeed
	}
	current := a
	if err := a.Revise(basedOn, p); err != nil {
		if errors.Is(err, domain.ErrStale) {
			return current, err
		}
		return domain.Agent{}, err
	}
	if err := s.agents.Replace(ctx, a, basedOn); err != nil {
		if errors.Is(err, domain.ErrStale) {
			// Someone else wrote in between reading and writing.
			fresh, _, _ := s.agents.ByID(ctx, id)
			return fresh, err
		}
		return domain.Agent{}, err
	}
	s.events.Publish(ctx, AgentRevised{AgentID: a.ID, OwnerID: ownerID, Revision: a.Revision})
	return s.reload(ctx, a.ID)
}

// DeleteAgent marks the agent deleted, keeping a tombstone for sync.
func (s *Service) DeleteAgent(ctx context.Context, id, ownerID uint64, basedOn int) (domain.Agent, error) {
	a, err := s.own(ctx, id, ownerID)
	if err != nil {
		return domain.Agent{}, err
	}
	current := a
	if err := a.Delete(basedOn, s.now()); err != nil {
		if errors.Is(err, domain.ErrStale) {
			return current, err
		}
		return domain.Agent{}, err
	}
	if err := s.agents.Replace(ctx, a, basedOn); err != nil {
		return domain.Agent{}, err
	}
	s.events.Publish(ctx, AgentDeleted{AgentID: a.ID, OwnerID: ownerID})
	return a, nil
}

// ListAgents returns the member's agents without files; with since, every
// agent changed since then, tombstones included, for sync.
func (s *Service) ListAgents(ctx context.Context, ownerID uint64, since *time.Time) ([]domain.Agent, error) {
	return s.agents.OfOwner(ctx, ownerID, since)
}

// GetAgent returns one of the member's agents with its files.
func (s *Service) GetAgent(ctx context.Context, id, ownerID uint64) (domain.Agent, error) {
	a, err := s.own(ctx, id, ownerID)
	if err != nil {
		return domain.Agent{}, err
	}
	if a.DeletedAt != nil {
		return domain.Agent{}, ErrAgentNotFound
	}
	return a, nil
}

func (s *Service) own(ctx context.Context, id, ownerID uint64) (domain.Agent, error) {
	a, found, err := s.agents.ByID(ctx, id)
	if err != nil {
		return domain.Agent{}, err
	}
	if !found || a.OwnerID != ownerID {
		return domain.Agent{}, ErrAgentNotFound
	}
	return a, nil
}

func (s *Service) reload(ctx context.Context, id uint64) (domain.Agent, error) {
	a, found, err := s.agents.ByID(ctx, id)
	if err != nil {
		return domain.Agent{}, err
	}
	if !found {
		return domain.Agent{}, fmt.Errorf("agent %d vanished after writing", id)
	}
	return a, nil
}

func randomSeed() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
