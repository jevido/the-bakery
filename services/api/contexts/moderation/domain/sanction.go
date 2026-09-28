package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Sanction kinds.
const (
	Suspension = "suspension"
	Ban        = "ban"
)

const reasonMax = 1000

var (
	ErrInvalidSanctionTarget = errors.New("a sanction is on a member or a guild")
	ErrInvalidSanctionKind   = errors.New("a sanction is a suspension or a ban")
	ErrSuspensionNeedsUntil  = errors.New("a suspension needs an end in the future")
	ErrBanHasNoUntil         = errors.New("a ban has no end; lift it instead")
	ErrInvalidReason         = errors.New("a reason is needed, at most 1000 characters")
	ErrAlreadySanctioned     = errors.New("this already has an active sanction; lift it first")
	ErrSanctionLifted        = errors.New("this sanction has already ended")
)

// Sanction is a suspension (until a date) or a ban (until lifted) on one
// member or one guild. One is active per target at a time.
type Sanction struct {
	ID           uint64
	TargetKind   string
	TargetID     uint64
	Kind         string
	Reason       string
	Until        *time.Time
	ByOperatorID uint64
	LiftedAt     *time.Time
	CreatedAt    time.Time
}

// NewSanction makes a sanction by an operator.
func NewSanction(targetKind string, targetID uint64, kind, reason string, until *time.Time, by uint64, now time.Time) (Sanction, error) {
	if (targetKind != TargetMember && targetKind != TargetGuild) || targetID == 0 {
		return Sanction{}, ErrInvalidSanctionTarget
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > reasonMax {
		return Sanction{}, ErrInvalidReason
	}
	switch kind {
	case Suspension:
		if until == nil || !until.After(now) {
			return Sanction{}, ErrSuspensionNeedsUntil
		}
	case Ban:
		if until != nil {
			return Sanction{}, ErrBanHasNoUntil
		}
	default:
		return Sanction{}, ErrInvalidSanctionKind
	}
	return Sanction{TargetKind: targetKind, TargetID: targetID, Kind: kind, Reason: reason, Until: until, ByOperatorID: by, CreatedAt: now}, nil
}

// ActiveAt reports whether the sanction holds at now: not lifted, and a
// suspension not yet over.
func (s Sanction) ActiveAt(now time.Time) bool {
	if s.LiftedAt != nil {
		return false
	}
	return s.Kind == Ban || (s.Until != nil && now.Before(*s.Until))
}

// Lift ends the sanction, once.
func (s *Sanction) Lift(now time.Time) error {
	if !s.ActiveAt(now) {
		return ErrSanctionLifted
	}
	s.LiftedAt = &now
	return nil
}
