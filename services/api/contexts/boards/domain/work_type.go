package domain

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

const workTypeNameMax = 40

var (
	ErrInvalidWorkTypeKey  = errors.New("a work type key is a lowercase letter, then up to 31 lowercase letters, digits or hyphens")
	ErrInvalidWorkTypeName = errors.New("work type name must be 1 to 40 characters")
	ErrDuplicateWorkType   = errors.New("this guild already has a work type with that key")
	ErrUnknownWorkType     = errors.New("unknown work type")
	ErrWorkTypeInUse       = errors.New("tasks still have this work type; change them first")
)

var workTypeKey = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

// WorkType is a kind of work a task needs, one of a guild's list. Other
// contexts know it by its key only.
type WorkType struct {
	ID       uint64
	GuildID  uint64
	Key      string
	Name     string
	Position string
}

// DefaultWorkTypes are the work types every guild starts with, in order.
var DefaultWorkTypes = []struct{ Key, Name string }{
	{"coding", "Coding"}, {"research", "Research"}, {"writing", "Writing"}, {"testing", "Testing"},
	{"design", "Design"}, {"review", "Review"}, {"ops", "Ops"},
}

// NewWorkType makes a work type at the end of the guild's list (after the
// position last, "" when the list is empty).
func NewWorkType(guildID uint64, key, name, last string) (WorkType, error) {
	if !workTypeKey.MatchString(key) {
		return WorkType{}, ErrInvalidWorkTypeKey
	}
	name, err := cleanWorkTypeName(name)
	if err != nil {
		return WorkType{}, err
	}
	pos, err := KeyBetween(last, "")
	if err != nil {
		return WorkType{}, err
	}
	return WorkType{GuildID: guildID, Key: key, Name: name, Position: pos}, nil
}

// DefaultsFor are a guild's starting work types, in order.
func DefaultsFor(guildID uint64) []WorkType {
	out := make([]WorkType, 0, len(DefaultWorkTypes))
	last := ""
	for _, d := range DefaultWorkTypes {
		wt, _ := NewWorkType(guildID, d.Key, d.Name, last)
		out = append(out, wt)
		last = wt.Position
	}
	return out
}

func (w *WorkType) Rename(name string) error {
	name, err := cleanWorkTypeName(name)
	if err != nil {
		return err
	}
	w.Name = name
	return nil
}

// Reposition puts the work type between the ones at above and below.
func (w *WorkType) Reposition(above, below string) error {
	pos, err := KeyBetween(above, below)
	if err != nil {
		return err
	}
	w.Position = pos
	return nil
}

func cleanWorkTypeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > workTypeNameMax {
		return "", ErrInvalidWorkTypeName
	}
	return name, nil
}
