package app

import (
	"context"
	"errors"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/guilds/domain"
)

var (
	ErrInviteNotFound = errors.New("this invite link does not exist")
	// ErrCodeTaken is returned by Invites.Add when the code is already in use.
	ErrCodeTaken = errors.New("invite code taken")
)

// Invites stores invites.
type Invites interface {
	Add(ctx context.Context, inv domain.Invite) (domain.Invite, error)
	ByID(ctx context.Context, id uint64) (domain.Invite, bool, error)
	ByCode(ctx context.Context, code string) (domain.Invite, bool, error)
	OfGuild(ctx context.Context, guildID uint64) ([]domain.Invite, error)
	Revoke(ctx context.Context, inv domain.Invite) error
	// CountUse adds one use in place (uses = uses + 1).
	CountUse(ctx context.Context, id uint64) error
}

// Codes makes invite codes.
type Codes interface {
	NewCode() (string, error)
}

// InviteInfo is what anyone holding a link may see before joining.
type InviteInfo struct {
	GuildName   string
	MemberCount int
	// Problem is why the invite cannot be used, or nil.
	Problem error
}

func (s *Service) CreateInvite(ctx context.Context, guildID, by uint64, expiresIn *time.Duration, maxUses *int) (domain.Invite, error) {
	g, err := s.guildOf(ctx, guildID, by)
	if err != nil {
		return domain.Invite{}, err
	}
	if g.Archived {
		return domain.Invite{}, domain.ErrArchived
	}
	for range 3 {
		code, err := s.codes.NewCode()
		if err != nil {
			return domain.Invite{}, err
		}
		inv, ev, err := domain.NewInvite(guildID, by, code, time.Now(), expiresIn, maxUses)
		if err != nil {
			return domain.Invite{}, err
		}
		inv, err = s.invites.Add(ctx, inv)
		if errors.Is(err, ErrCodeTaken) {
			continue
		}
		if err != nil {
			return domain.Invite{}, err
		}
		ev.InviteID = inv.ID
		s.events.Other(ctx, ev)
		return inv, nil
	}
	return domain.Invite{}, ErrCodeTaken
}

func (s *Service) ListInvites(ctx context.Context, guildID, by uint64) ([]domain.Invite, error) {
	if _, err := s.guildOf(ctx, guildID, by); err != nil {
		return nil, err
	}
	return s.invites.OfGuild(ctx, guildID)
}

func (s *Service) RevokeInvite(ctx context.Context, guildID, inviteID, by uint64) error {
	if _, err := s.guildOf(ctx, guildID, by); err != nil {
		return err
	}
	inv, found, err := s.invites.ByID(ctx, inviteID)
	if err != nil {
		return err
	}
	if !found || inv.GuildID != guildID {
		return ErrInviteNotFound
	}
	ev := inv.Revoke(time.Now())
	if err := s.invites.Revoke(ctx, inv); err != nil {
		return err
	}
	s.events.Other(ctx, ev)
	return nil
}

// Invite describes an invite by its code, for anyone holding the link.
func (s *Service) Invite(ctx context.Context, code string) (InviteInfo, error) {
	inv, found, err := s.invites.ByCode(ctx, code)
	if err != nil {
		return InviteInfo{}, err
	}
	if !found {
		return InviteInfo{}, ErrInviteNotFound
	}
	g, found, err := s.guilds.ByID(ctx, inv.GuildID)
	if err != nil {
		return InviteInfo{}, err
	}
	if !found {
		return InviteInfo{}, ErrInviteNotFound
	}
	info := InviteInfo{GuildName: g.Name, MemberCount: len(g.MemberIDs()), Problem: inv.Usable(time.Now())}
	if info.Problem == nil && g.Archived {
		info.Problem = domain.ErrArchived
	}
	return info, nil
}

// AcceptInvite makes memberID a member of the invite's guild and returns
// the guild. Someone already in the guild is simply let in, without using
// the invite up.
//
// Guild and invite are two aggregates, so this is two steps: the member is
// added, then the use is counted. Two people accepting at the same moment
// can go one use over the limit.
func (s *Service) AcceptInvite(ctx context.Context, code string, memberID uint64) (domain.Guild, error) {
	inv, found, err := s.invites.ByCode(ctx, code)
	if err != nil {
		return domain.Guild{}, err
	}
	if !found {
		return domain.Guild{}, ErrInviteNotFound
	}
	g, found, err := s.guilds.ByID(ctx, inv.GuildID)
	if err != nil {
		return domain.Guild{}, err
	}
	if !found {
		return domain.Guild{}, ErrInviteNotFound
	}
	if g.HasMember(memberID) {
		return g, nil
	}
	if err := inv.Usable(time.Now()); err != nil {
		return domain.Guild{}, err
	}
	ev, err := g.AddMember(memberID)
	if err != nil {
		return domain.Guild{}, err
	}
	if err := s.guilds.AddMembership(ctx, ev); err != nil && !errors.Is(err, domain.ErrAlreadyMember) {
		return domain.Guild{}, err
	}
	s.events.MemberJoined(ctx, ev)
	if err := s.invites.CountUse(ctx, inv.ID); err != nil {
		return domain.Guild{}, err
	}
	return g, nil
}
