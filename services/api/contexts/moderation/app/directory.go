package app

import (
	"context"
	"errors"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

var (
	ErrMemberNotFound = errors.New("member not found")
	ErrGuildNotFound  = errors.New("guild not found")
)

// MemberCard is a member as the console shows them, translated from the
// identity context.
type MemberCard struct {
	ID           uint64
	Email        string
	DisplayName  string
	PortraitSeed string
	JoinedAt     time.Time
}

// GuildCard is a guild as the console shows it, translated from the
// guilds context.
type GuildCard struct {
	ID        uint64
	Name      string
	Archived  bool
	FoundedAt time.Time
	MemberIDs []uint64
}

// Directory is what the console reads about members and guilds. It asks
// the identity, guilds and boards contexts; moderation never reads their
// tables.
type Directory interface {
	FindMembers(ctx context.Context, q string, limit int) ([]MemberCard, error)
	Members(ctx context.Context, ids []uint64) ([]MemberCard, error)
	GuildCounts(ctx context.Context, memberIDs []uint64) (map[uint64]int, error)
	FindGuilds(ctx context.Context, q string, limit int) ([]GuildCard, error)
	Guild(ctx context.Context, id uint64) (GuildCard, bool, error)
	GuildsOfMember(ctx context.Context, memberID uint64) ([]GuildCard, error)
	BoardCount(ctx context.Context, guildID uint64) (int, error)
}

// MemberListing is a row in the console's members list.
type MemberListing struct {
	MemberCard
	GuildCount int
	Active     *domain.Sanction
}

// GuildListing is a row in the console's guilds list.
type GuildListing struct {
	GuildCard
	Active *domain.Sanction
}

// MemberRecord is everything the console shows about one member.
type MemberRecord struct {
	MemberCard
	Guilds    []GuildCard
	Sanctions []domain.Sanction
	Reports   []domain.Report
	Audit     []domain.AuditEntry
}

// GuildRecord is everything the console shows about one guild.
type GuildRecord struct {
	GuildCard
	Members    []MemberCard
	BoardCount int
	Sanctions  []domain.Sanction
	Reports    []domain.Report
	Audit      []domain.AuditEntry
}

const listLimit = 50

// FindMembers lists members matching q, with their guild count and any
// sanction in force.
func (s *Service) FindMembers(ctx context.Context, q string) ([]MemberListing, error) {
	cards, err := s.directory.FindMembers(ctx, q, listLimit)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, len(cards))
	for i, c := range cards {
		ids[i] = c.ID
	}
	counts, err := s.directory.GuildCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]MemberListing, len(cards))
	for i, c := range cards {
		active, err := s.sanctions.Active(ctx, domain.TargetMember, c.ID, s.now())
		if err != nil {
			return nil, err
		}
		out[i] = MemberListing{MemberCard: c, GuildCount: counts[c.ID], Active: active}
	}
	return out, nil
}

// FindGuilds lists guilds matching q, with any sanction in force.
func (s *Service) FindGuilds(ctx context.Context, q string) ([]GuildListing, error) {
	cards, err := s.directory.FindGuilds(ctx, q, listLimit)
	if err != nil {
		return nil, err
	}
	out := make([]GuildListing, len(cards))
	for i, c := range cards {
		active, err := s.sanctions.Active(ctx, domain.TargetGuild, c.ID, s.now())
		if err != nil {
			return nil, err
		}
		out[i] = GuildListing{GuildCard: c, Active: active}
	}
	return out, nil
}

// Member is one member's record: their guilds, sanctions, the reports
// about them, and the audit log of what they did or had done to them.
func (s *Service) Member(ctx context.Context, id uint64) (MemberRecord, error) {
	cards, err := s.directory.Members(ctx, []uint64{id})
	if err != nil {
		return MemberRecord{}, err
	}
	if len(cards) == 0 {
		return MemberRecord{}, ErrMemberNotFound
	}
	r := MemberRecord{MemberCard: cards[0]}
	if r.Guilds, err = s.directory.GuildsOfMember(ctx, id); err != nil {
		return MemberRecord{}, err
	}
	if r.Sanctions, err = s.sanctions.OfTarget(ctx, domain.TargetMember, id); err != nil {
		return MemberRecord{}, err
	}
	if r.Reports, err = s.reports.About(ctx, domain.TargetMember, id); err != nil {
		return MemberRecord{}, err
	}
	about, err := s.audit.List(ctx, AuditFilter{TargetKind: domain.TargetMember, TargetID: id, Limit: 50})
	if err != nil {
		return MemberRecord{}, err
	}
	by, err := s.audit.List(ctx, AuditFilter{ActorKind: domain.ActorMember, ActorID: id, Limit: 50})
	if err != nil {
		return MemberRecord{}, err
	}
	r.Audit = mergeNewestFirst(about, by, 50)
	return r, nil
}

// Guild is one guild's record: its members, how many boards it has, its
// sanctions, the reports about it and its audit log.
func (s *Service) Guild(ctx context.Context, id uint64) (GuildRecord, error) {
	card, found, err := s.directory.Guild(ctx, id)
	if err != nil {
		return GuildRecord{}, err
	}
	if !found {
		return GuildRecord{}, ErrGuildNotFound
	}
	r := GuildRecord{GuildCard: card}
	if r.Members, err = s.directory.Members(ctx, card.MemberIDs); err != nil {
		return GuildRecord{}, err
	}
	if r.BoardCount, err = s.directory.BoardCount(ctx, id); err != nil {
		return GuildRecord{}, err
	}
	if r.Sanctions, err = s.sanctions.OfTarget(ctx, domain.TargetGuild, id); err != nil {
		return GuildRecord{}, err
	}
	if r.Reports, err = s.reports.About(ctx, domain.TargetGuild, id); err != nil {
		return GuildRecord{}, err
	}
	if r.Audit, err = s.audit.List(ctx, AuditFilter{TargetKind: domain.TargetGuild, TargetID: id, Limit: 50}); err != nil {
		return GuildRecord{}, err
	}
	return r, nil
}

// Names maps member and guild ids to what the console calls them, for
// lists that only hold ids (reports, the audit log).
func (s *Service) Names(ctx context.Context, memberIDs, guildIDs []uint64) (members, guilds map[uint64]string, err error) {
	members, guilds = map[uint64]string{}, map[uint64]string{}
	cards, err := s.directory.Members(ctx, memberIDs)
	if err != nil {
		return nil, nil, err
	}
	for _, c := range cards {
		members[c.ID] = c.DisplayName
	}
	for _, id := range guildIDs {
		if _, ok := guilds[id]; ok {
			continue
		}
		g, found, err := s.directory.Guild(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if found {
			guilds[id] = g.Name
		}
	}
	return members, guilds, nil
}

// mergeNewestFirst merges two newest-first lists of entries, dropping
// duplicates, at most limit.
func mergeNewestFirst(a, b []domain.AuditEntry, limit int) []domain.AuditEntry {
	out := make([]domain.AuditEntry, 0, len(a)+len(b))
	seen := map[uint64]bool{}
	i, j := 0, 0
	for len(out) < limit && (i < len(a) || j < len(b)) {
		var e domain.AuditEntry
		if j >= len(b) || (i < len(a) && a[i].ID > b[j].ID) {
			e, i = a[i], i+1
		} else {
			e, j = b[j], j+1
		}
		if !seen[e.ID] {
			seen[e.ID] = true
			out = append(out, e)
		}
	}
	return out
}
