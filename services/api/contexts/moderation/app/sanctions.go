package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

var (
	ErrSanctionNotFound = errors.New("sanction not found")
	ErrTargetNotFound   = errors.New("no such member or guild")
)

// Sanctions stores sanctions. Add ends a finished suspension of the target
// first, and fails with domain.ErrAlreadySanctioned while another one holds.
type Sanctions interface {
	Add(ctx context.Context, s domain.Sanction) (domain.Sanction, error)
	Save(ctx context.Context, s domain.Sanction) error
	ByID(ctx context.Context, id uint64) (domain.Sanction, bool, error)
	// Active is the target's sanction holding at now, if any.
	Active(ctx context.Context, targetKind string, targetID uint64, now time.Time) (*domain.Sanction, error)
	// OfTarget is the target's sanctions, newest first.
	OfTarget(ctx context.Context, targetKind string, targetID uint64) ([]domain.Sanction, error)
}

// Targets asks identity and guilds whether a member or a guild exists.
type Targets interface {
	MemberExists(ctx context.Context, id uint64) (bool, error)
	GuildExists(ctx context.Context, id uint64) (bool, error)
}

// Effects are what a sanction does elsewhere at once: the target's live
// board streams end.
type Effects interface {
	Sanctioned(ctx context.Context, targetKind string, targetID uint64) error
}

// activeTTL is how long an answer to "is this target sanctioned?" is kept.
// A sanction takes effect within this, a lift too.
const activeTTL = 5 * time.Second

type cached struct {
	s  *domain.Sanction
	at time.Time
}

// activeCache keeps ActiveFor's answers for activeTTL.
type activeCache struct {
	mu sync.Mutex
	m  map[string]cached
}

func cacheKey(kind string, id uint64) string { return kind + ":" + itoa(id) }

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for ; n > 0; n /= 10 {
		i--
		b[i] = byte('0' + n%10)
	}
	return string(b[i:])
}

// ActiveFor is the target's sanction holding now, or nil. Other contexts
// ask it on every request, so answers are kept a few seconds.
func (s *Service) ActiveFor(ctx context.Context, targetKind string, targetID uint64) (*domain.Sanction, error) {
	key := cacheKey(targetKind, targetID)
	now := s.now()
	s.cache.mu.Lock()
	c, ok := s.cache.m[key]
	s.cache.mu.Unlock()
	if ok && now.Sub(c.at) < activeTTL {
		if c.s != nil && !c.s.ActiveAt(now) {
			return nil, nil
		}
		return c.s, nil
	}
	active, err := s.sanctions.Active(ctx, targetKind, targetID, now)
	if err != nil {
		return nil, err
	}
	s.cache.mu.Lock()
	s.cache.m[key] = cached{active, now}
	s.cache.mu.Unlock()
	return active, nil
}

func (s *Service) forget(targetKind string, targetID uint64) {
	s.cache.mu.Lock()
	delete(s.cache.m, cacheKey(targetKind, targetID))
	s.cache.mu.Unlock()
}

// Sanction suspends or bans a member or a guild, and records it.
func (s *Service) Sanction(ctx context.Context, operatorID uint64, targetKind string, targetID uint64, kind, reason string, until *time.Time) (domain.Sanction, error) {
	sanction, err := domain.NewSanction(targetKind, targetID, kind, reason, until, operatorID, s.now())
	if err != nil {
		return domain.Sanction{}, err
	}
	var exists bool
	if targetKind == domain.TargetMember {
		exists, err = s.targets.MemberExists(ctx, targetID)
	} else {
		exists, err = s.targets.GuildExists(ctx, targetID)
	}
	if err != nil {
		return domain.Sanction{}, err
	}
	if !exists {
		return domain.Sanction{}, ErrTargetNotFound
	}
	if sanction, err = s.sanctions.Add(ctx, sanction); err != nil {
		return domain.Sanction{}, err
	}
	s.forget(targetKind, targetID)
	meta := map[string]any{"sanction_id": sanction.ID, "kind": kind}
	if until != nil {
		meta["until"] = until.UTC().Format(time.RFC3339)
	}
	_ = s.Record(ctx, domain.AuditEntry{ActorKind: domain.ActorOperator, ActorID: operatorID, Action: "sanction." + kind,
		TargetKind: targetKind, TargetID: targetID, Reason: sanction.Reason, Meta: meta})
	if s.effects != nil {
		_ = s.effects.Sanctioned(ctx, targetKind, targetID)
	}
	return sanction, nil
}

// LiftSanction ends a sanction early, and records it.
func (s *Service) LiftSanction(ctx context.Context, operatorID, sanctionID uint64, reason string) (domain.Sanction, error) {
	sanction, found, err := s.sanctions.ByID(ctx, sanctionID)
	if err != nil {
		return domain.Sanction{}, err
	}
	if !found {
		return domain.Sanction{}, ErrSanctionNotFound
	}
	if err := sanction.Lift(s.now()); err != nil {
		return domain.Sanction{}, err
	}
	if err := s.sanctions.Save(ctx, sanction); err != nil {
		return domain.Sanction{}, err
	}
	s.forget(sanction.TargetKind, sanction.TargetID)
	_ = s.Record(ctx, domain.AuditEntry{ActorKind: domain.ActorOperator, ActorID: operatorID, Action: "sanction.lifted",
		TargetKind: sanction.TargetKind, TargetID: sanction.TargetID, Reason: reason, Meta: map[string]any{"sanction_id": sanction.ID}})
	return sanction, nil
}

// SanctionsOf lists a target's sanctions, newest first.
func (s *Service) SanctionsOf(ctx context.Context, targetKind string, targetID uint64) ([]domain.Sanction, error) {
	return s.sanctions.OfTarget(ctx, targetKind, targetID)
}
