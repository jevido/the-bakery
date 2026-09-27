// Package app holds the guilds use cases.
package app

import (
	"context"
	"errors"

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
	OfMember(ctx context.Context, memberID uint64) ([]domain.Guild, error)
	// AddMembership stores a membership the aggregate accepted. It returns
	// domain.ErrAlreadyMember if one already exists.
	AddMembership(ctx context.Context, ev domain.MemberJoined) error
}

// MemberLookup finds a member id by email. It is identity's lookup, seen
// from here as nothing more than an id.
type MemberLookup interface {
	MemberIDByEmail(ctx context.Context, email string) (uint64, bool, error)
}

// Events receives the domain events guilds announce.
type Events interface {
	GuildFounded(ctx context.Context, ev domain.GuildFounded)
	MemberJoined(ctx context.Context, ev domain.MemberJoined)
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

func (s *Service) ListGuildsOf(ctx context.Context, memberID uint64) ([]domain.Guild, error) {
	return s.guilds.OfMember(ctx, memberID)
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
