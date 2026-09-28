package app

import (
	"context"
	"errors"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

var (
	ErrClaimNotFound = errors.New("claim not found")
	ErrNotYourAgent  = errors.New("that agent is not one of yours")
)

// Claims stores claims. Add stores a new claim after releasing the task's
// expired one, in one transaction, and fails with domain.ErrTaskClaimed
// when another claim still holds the task.
type Claims interface {
	Current(ctx context.Context, taskID uint64) (*domain.Claim, error)
	Add(ctx context.Context, c domain.Claim) (domain.Claim, error)
	Save(ctx context.Context, c domain.Claim) error
	ByID(ctx context.Context, id uint64) (domain.Claim, bool, error)
	// ActiveOf returns the claims holding any of the tasks at now, by task.
	ActiveOf(ctx context.Context, taskIDs []uint64, now time.Time) (map[uint64]domain.Claim, error)
}

// AgentOwners is the agents context's answer to "is this agent the
// member's?". Boards knows agents only by id.
type AgentOwners interface {
	Owns(ctx context.Context, memberID, agentID uint64) (bool, error)
}

// ClaimTask gives the task to one of the member's agents, on one machine,
// for domain.ClaimTTL unless heartbeated.
func (s *Service) ClaimTask(ctx context.Context, taskID, memberID, agentID uint64, machineID string) (domain.Claim, error) {
	t, err := s.task(ctx, taskID, memberID)
	if err != nil {
		return domain.Claim{}, err
	}
	owns, err := s.owners.Owns(ctx, memberID, agentID)
	if err != nil {
		return domain.Claim{}, err
	}
	if !owns {
		return domain.Claim{}, ErrNotYourAgent
	}
	current, err := s.claims.Current(ctx, t.ID)
	if err != nil {
		return domain.Claim{}, err
	}
	c, err := t.Claim(agentID, memberID, machineID, current, s.now())
	if err != nil {
		return domain.Claim{}, err
	}
	if c, err = s.claims.Add(ctx, c); err != nil {
		return domain.Claim{}, err
	}
	s.events.Publish(ctx, domain.TaskClaimed{TaskID: t.ID, BoardID: t.BoardID, ActorID: memberID, ClaimID: c.ID, AgentID: c.AgentID, ExpiresAt: c.ExpiresAt})
	return c, nil
}

// HeartbeatClaim keeps a claim for another domain.ClaimTTL.
func (s *Service) HeartbeatClaim(ctx context.Context, claimID, memberID uint64) (domain.Claim, error) {
	c, _, err := s.claim(ctx, claimID, memberID)
	if err != nil {
		return domain.Claim{}, err
	}
	if err := c.Heartbeat(memberID, s.now()); err != nil {
		return domain.Claim{}, err
	}
	return c, s.claims.Save(ctx, c)
}

// ReleaseClaim lets the task go. It works in an archived guild too, so a
// run that ends there can still let go.
func (s *Service) ReleaseClaim(ctx context.Context, claimID, memberID uint64) error {
	c, t, err := s.claim(ctx, claimID, memberID)
	if err != nil {
		return err
	}
	changed, err := c.Release(memberID, s.now())
	if err != nil || !changed {
		return err
	}
	if err := s.claims.Save(ctx, c); err != nil {
		return err
	}
	s.events.Publish(ctx, domain.TaskReleased{TaskID: t.ID, BoardID: t.BoardID, ActorID: memberID, ClaimID: c.ID})
	return nil
}

// claim is a claim and its task, for a member of the task's guild.
func (s *Service) claim(ctx context.Context, claimID, memberID uint64) (domain.Claim, domain.Task, error) {
	c, found, err := s.claims.ByID(ctx, claimID)
	if err != nil {
		return domain.Claim{}, domain.Task{}, err
	}
	if !found {
		return domain.Claim{}, domain.Task{}, ErrClaimNotFound
	}
	t, err := s.readableTask(ctx, c.TaskID, memberID)
	if err != nil {
		return domain.Claim{}, domain.Task{}, err
	}
	return c, t, nil
}

// ActiveClaim is the claim holding the task now, if any.
func (s *Service) ActiveClaim(ctx context.Context, taskID uint64) (*domain.Claim, error) {
	m, err := s.claims.ActiveOf(ctx, []uint64{taskID}, s.now())
	if err != nil {
		return nil, err
	}
	if c, ok := m[taskID]; ok {
		return &c, nil
	}
	return nil, nil
}
