// Package domain holds the guilds model: a Guild and its memberships. It
// imports nothing outside the standard library.
package domain

import (
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

const nameMax = 60

var (
	ErrInvalidName   = errors.New("guild name must be 1 to 60 characters")
	ErrAlreadyMember = errors.New("already a member of this guild")
	ErrArchived      = errors.New("this guild is archived; restore it first")
	ErrNotArchived   = errors.New("this guild is not archived")
	ErrNotAMember    = errors.New("no such member in this guild")
	ErrLastMember    = errors.New("the last member cannot leave or be removed; archive the guild instead")
)

// Guild is the aggregate root: it owns its memberships, so it alone decides
// who may be added.
type Guild struct {
	ID        uint64
	Name      string
	Archived  bool
	memberIDs []uint64
}

// GuildFounded is announced when a guild is founded.
type GuildFounded struct {
	GuildID   uint64
	Name      string
	FounderID uint64
}

// MemberJoined is announced when a member is added to a guild.
type MemberJoined struct {
	GuildID  uint64
	MemberID uint64
}

// Found creates a guild whose first member is its founder. The guild has no
// ID until it is stored; the event gets it then.
func Found(name string, founderID uint64) (Guild, GuildFounded, error) {
	name, err := cleanName(name)
	if err != nil {
		return Guild{}, GuildFounded{}, err
	}
	g := Guild{Name: name, memberIDs: []uint64{founderID}}
	return g, GuildFounded{Name: name, FounderID: founderID}, nil
}

// GuildRenamed is announced when a guild gets a new name.
type GuildRenamed struct {
	GuildID uint64
	Name    string
}

// GuildArchived and GuildRestored are announced when a guild is archived or
// brought back.
type GuildArchived struct{ GuildID, By uint64 }
type GuildRestored struct{ GuildID, By uint64 }

// MemberLeft is announced when a member ends their own membership.
type MemberLeft struct{ GuildID, MemberID uint64 }

// MemberRemoved is announced when a member ends another's membership.
type MemberRemoved struct{ GuildID, MemberID, By uint64 }

// Rehydrate rebuilds a stored guild. Only persistence calls it.
func Rehydrate(id uint64, name string, archived bool, memberIDs []uint64) Guild {
	return Guild{ID: id, Name: name, Archived: archived, memberIDs: slices.Clone(memberIDs)}
}

func (g *Guild) Rename(name string) (GuildRenamed, error) {
	if g.Archived {
		return GuildRenamed{}, ErrArchived
	}
	name, err := cleanName(name)
	if err != nil {
		return GuildRenamed{}, err
	}
	g.Name = name
	return GuildRenamed{GuildID: g.ID, Name: name}, nil
}

func (g *Guild) Archive(by uint64) (GuildArchived, error) {
	if g.Archived {
		return GuildArchived{}, ErrArchived
	}
	g.Archived = true
	return GuildArchived{GuildID: g.ID, By: by}, nil
}

func (g *Guild) Restore(by uint64) (GuildRestored, error) {
	if !g.Archived {
		return GuildRestored{}, ErrNotArchived
	}
	g.Archived = false
	return GuildRestored{GuildID: g.ID, By: by}, nil
}

// Leave ends memberID's own membership.
func (g *Guild) Leave(memberID uint64) (MemberLeft, error) {
	if err := g.endMembership(memberID); err != nil {
		return MemberLeft{}, err
	}
	return MemberLeft{GuildID: g.ID, MemberID: memberID}, nil
}

// RemoveMember ends another member's membership.
func (g *Guild) RemoveMember(memberID, by uint64) (MemberRemoved, error) {
	if err := g.endMembership(memberID); err != nil {
		return MemberRemoved{}, err
	}
	return MemberRemoved{GuildID: g.ID, MemberID: memberID, By: by}, nil
}

func (g *Guild) endMembership(memberID uint64) error {
	if g.Archived {
		return ErrArchived
	}
	i := slices.Index(g.memberIDs, memberID)
	if i < 0 {
		return ErrNotAMember
	}
	if len(g.memberIDs) == 1 {
		return ErrLastMember
	}
	g.memberIDs = slices.Delete(g.memberIDs, i, i+1)
	return nil
}

func (g *Guild) AddMember(memberID uint64) (MemberJoined, error) {
	if g.Archived {
		return MemberJoined{}, ErrArchived
	}
	if g.HasMember(memberID) {
		return MemberJoined{}, ErrAlreadyMember
	}
	g.memberIDs = append(g.memberIDs, memberID)
	return MemberJoined{GuildID: g.ID, MemberID: memberID}, nil
}

func (g Guild) HasMember(memberID uint64) bool {
	return slices.Contains(g.memberIDs, memberID)
}

func (g Guild) MemberIDs() []uint64 {
	return slices.Clone(g.memberIDs)
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > nameMax {
		return "", ErrInvalidName
	}
	return name, nil
}
