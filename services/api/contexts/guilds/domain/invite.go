package domain

import (
	"errors"
	"time"
)

const inviteMaxLifetime = 30 * 24 * time.Hour

var (
	ErrInviteRevoked     = errors.New("this invite was withdrawn")
	ErrInviteExpired     = errors.New("this invite has expired")
	ErrInviteUsedUp      = errors.New("this invite has been used up")
	ErrInvalidInviteRule = errors.New("an invite lasts from 1 hour to 30 days, and allows at least 1 use")
)

// Invite is a standing offer to join one guild. It is its own aggregate:
// accepting one changes the guild (a new member) and the invite (a use).
type Invite struct {
	ID        uint64
	GuildID   uint64
	CreatedBy uint64
	Code      string
	ExpiresAt *time.Time
	MaxUses   *int
	Uses      int
	RevokedAt *time.Time
}

// InviteCreated and InviteRevoked are announced when an invite is made or
// withdrawn.
type InviteCreated struct {
	InviteID, GuildID uint64
	ExpiresAt         *time.Time
	MaxUses           *int
}

type InviteRevoked struct{ InviteID, GuildID uint64 }

// NewInvite makes an invite to guildID by a member. expiresIn and maxUses
// are optional (nil: no end, no limit).
func NewInvite(guildID, by uint64, code string, now time.Time, expiresIn *time.Duration, maxUses *int) (Invite, InviteCreated, error) {
	inv := Invite{GuildID: guildID, CreatedBy: by, Code: code, MaxUses: maxUses}
	if expiresIn != nil {
		if *expiresIn < time.Hour || *expiresIn > inviteMaxLifetime {
			return Invite{}, InviteCreated{}, ErrInvalidInviteRule
		}
		at := now.Add(*expiresIn)
		inv.ExpiresAt = &at
	}
	if maxUses != nil && *maxUses < 1 {
		return Invite{}, InviteCreated{}, ErrInvalidInviteRule
	}
	return inv, InviteCreated{GuildID: guildID, ExpiresAt: inv.ExpiresAt, MaxUses: maxUses}, nil
}

// Usable reports why the invite cannot be accepted at now, or nil.
func (i Invite) Usable(now time.Time) error {
	switch {
	case i.RevokedAt != nil:
		return ErrInviteRevoked
	case i.ExpiresAt != nil && !now.Before(*i.ExpiresAt):
		return ErrInviteExpired
	case i.MaxUses != nil && i.Uses >= *i.MaxUses:
		return ErrInviteUsedUp
	}
	return nil
}

// Use counts one acceptance.
func (i *Invite) Use(now time.Time) error {
	if err := i.Usable(now); err != nil {
		return err
	}
	i.Uses++
	return nil
}

// Revoke withdraws the invite; revoking twice changes nothing.
func (i *Invite) Revoke(now time.Time) InviteRevoked {
	if i.RevokedAt == nil {
		i.RevokedAt = &now
	}
	return InviteRevoked{InviteID: i.ID, GuildID: i.GuildID}
}
