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
)

// Guild is the aggregate root: it owns its memberships, so it alone decides
// who may be added.
type Guild struct {
	ID        uint64
	Name      string
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
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > nameMax {
		return Guild{}, GuildFounded{}, ErrInvalidName
	}
	g := Guild{Name: name, memberIDs: []uint64{founderID}}
	return g, GuildFounded{Name: name, FounderID: founderID}, nil
}

// Rehydrate rebuilds a stored guild. Only persistence calls it.
func Rehydrate(id uint64, name string, memberIDs []uint64) Guild {
	return Guild{ID: id, Name: name, memberIDs: slices.Clone(memberIDs)}
}

func (g *Guild) AddMember(memberID uint64) (MemberJoined, error) {
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
