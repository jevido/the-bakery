// Package app holds the guilds use cases.
package app

import (
	"context"
	"errors"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/guilds/domain"
)

var (
	ErrNotMember    = errors.New("not a member of this guild")
	ErrUnknownEmail = errors.New("no member with this email")
)

// Guilds stores guilds with their memberships.
type Guilds interface {
	// Add stores a new guild and its founding membership, returning it with its ID.
	Add(ctx context.Context, g domain.Guild) (domain.Guild, error)
	ByID(ctx context.Context, id uint64) (domain.Guild, bool, error)
	// OfMember returns the member's guilds; archived ones only when asked.
	OfMember(ctx context.Context, memberID uint64, includeArchived bool) ([]domain.Guild, error)
	// Save stores the guild's name and archived flag.
	Save(ctx context.Context, g domain.Guild) error
	// RemoveMembership ends one membership.
	RemoveMembership(ctx context.Context, guildID, memberID uint64) error
	// Memberships lists a guild's memberships, oldest first.
	Memberships(ctx context.Context, guildID uint64) ([]Membership, error)
	// AddMembership stores a membership the aggregate accepted. It returns
	// domain.ErrAlreadyMember if one already exists.
	AddMembership(ctx context.Context, ev domain.MemberJoined) error
}

// Membership is one member of a guild and when they joined.
type Membership struct {
	MemberID uint64
	JoinedAt time.Time
}

// Member is a guild member as the members list shows them.
type Member struct {
	MemberID    uint64
	DisplayName string
	JoinedAt    time.Time
}

// MemberLookup is what guilds asks identity: a member id by email, and
// display names by id. Guilds keeps only ids.
type MemberLookup interface {
	MemberIDByEmail(ctx context.Context, email string) (uint64, bool, error)
	DisplayNames(ctx context.Context, ids []uint64) (map[uint64]string, error)
}

// Events receives the domain events guilds announce.
type Events interface {
	GuildFounded(ctx context.Context, ev domain.GuildFounded)
	MemberJoined(ctx context.Context, ev domain.MemberJoined)
	// Other announces the smaller events (renamed, archived, restored, left,
	// removed); nothing subscribes to them yet.
	Other(ctx context.Context, ev any)
}

type Service struct {
	guilds  Guilds
	members MemberLookup
	events  Events
}

func NewService(guilds Guilds, members MemberLookup, events Events) *Service {
	return &Service{guilds: guilds, members: members, events: events}
}

func (s *Service) FoundGuild(ctx context.Context, name string, founderID uint64) (domain.Guild, error) {
	g, ev, err := domain.Found(name, founderID)
	if err != nil {
		return domain.Guild{}, err
	}
	g, err = s.guilds.Add(ctx, g)
	if err != nil {
		return domain.Guild{}, err
	}
	ev.GuildID = g.ID
	s.events.GuildFounded(ctx, ev)
	return g, nil
}

func (s *Service) ListGuildsOf(ctx context.Context, memberID uint64, includeArchived bool) ([]domain.Guild, error) {
	return s.guilds.OfMember(ctx, memberID, includeArchived)
}

// Guild returns one guild the member is in, archived or not.
func (s *Service) Guild(ctx context.Context, guildID, memberID uint64) (domain.Guild, error) {
	return s.guildOf(ctx, guildID, memberID)
}

func (s *Service) RenameGuild(ctx context.Context, guildID, by uint64, name string) (domain.Guild, error) {
	return s.change(ctx, guildID, by, func(g *domain.Guild) (any, error) { return g.Rename(name) })
}

func (s *Service) ArchiveGuild(ctx context.Context, guildID, by uint64) (domain.Guild, error) {
	return s.change(ctx, guildID, by, func(g *domain.Guild) (any, error) { return g.Archive(by) })
}

func (s *Service) RestoreGuild(ctx context.Context, guildID, by uint64) (domain.Guild, error) {
	return s.change(ctx, guildID, by, func(g *domain.Guild) (any, error) { return g.Restore(by) })
}

// change applies f to the guild and stores the name and archived flag.
func (s *Service) change(ctx context.Context, guildID, by uint64, f func(*domain.Guild) (any, error)) (domain.Guild, error) {
	g, err := s.guildOf(ctx, guildID, by)
	if err != nil {
		return domain.Guild{}, err
	}
	ev, err := f(&g)
	if err != nil {
		return domain.Guild{}, err
	}
	if err := s.guilds.Save(ctx, g); err != nil {
		return domain.Guild{}, err
	}
	s.events.Other(ctx, ev)
	return g, nil
}

// Members lists the guild's members with their display names, oldest first.
func (s *Service) Members(ctx context.Context, guildID, by uint64) ([]Member, error) {
	if _, err := s.guildOf(ctx, guildID, by); err != nil {
		return nil, err
	}
	ms, err := s.guilds.Memberships(ctx, guildID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, len(ms))
	for i, m := range ms {
		ids[i] = m.MemberID
	}
	names, err := s.members.DisplayNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Member, len(ms))
	for i, m := range ms {
		out[i] = Member{MemberID: m.MemberID, DisplayName: names[m.MemberID], JoinedAt: m.JoinedAt}
	}
	return out, nil
}

// RemoveMember ends memberID's membership; by must be a member too. Removing
// yourself is leaving.
func (s *Service) RemoveMember(ctx context.Context, guildID, by, memberID uint64) error {
	if memberID == by {
		return s.Leave(ctx, guildID, by)
	}
	g, err := s.guildOf(ctx, guildID, by)
	if err != nil {
		return err
	}
	ev, err := g.RemoveMember(memberID, by)
	if err != nil {
		return err
	}
	if err := s.guilds.RemoveMembership(ctx, guildID, memberID); err != nil {
		return err
	}
	s.events.Other(ctx, ev)
	return nil
}

func (s *Service) Leave(ctx context.Context, guildID, memberID uint64) error {
	g, err := s.guildOf(ctx, guildID, memberID)
	if err != nil {
		return err
	}
	ev, err := g.Leave(memberID)
	if err != nil {
		return err
	}
	if err := s.guilds.RemoveMembership(ctx, guildID, memberID); err != nil {
		return err
	}
	s.events.Other(ctx, ev)
	return nil
}

// AddMember adds the member with the given email to a guild. Only a member
// of that guild may do so; a guild that does not exist looks the same as
// one the caller is not in.
func (s *Service) AddMember(ctx context.Context, guildID, byMemberID uint64, email string) (uint64, error) {
	g, err := s.guildOf(ctx, guildID, byMemberID)
	if err != nil {
		return 0, err
	}
	memberID, found, err := s.members.MemberIDByEmail(ctx, email)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, ErrUnknownEmail
	}
	ev, err := g.AddMember(memberID)
	if err != nil {
		return 0, err
	}
	if err := s.guilds.AddMembership(ctx, ev); err != nil {
		return 0, err
	}
	s.events.MemberJoined(ctx, ev)
	return memberID, nil
}

func (s *Service) guildOf(ctx context.Context, guildID, memberID uint64) (domain.Guild, error) {
	g, found, err := s.guilds.ByID(ctx, guildID)
	if err != nil {
		return domain.Guild{}, err
	}
	if !found || !g.HasMember(memberID) {
		return domain.Guild{}, ErrNotMember
	}
	return g, nil
}
