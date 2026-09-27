// Package domain holds the agents model: agents, their skillsets, traits and
// work priorities. It imports nothing outside the standard library.
package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalidName           = errors.New("an agent's name must be 1 to 40 characters")
	ErrInvalidTitle          = errors.New("an agent's title is at most 60 characters")
	ErrInvalidBackstory      = errors.New("a backstory is at most 2000 characters")
	ErrInvalidSlug           = errors.New("a slug is 1 to 40 lowercase letters, digits and hyphens")
	ErrUnknownTrait          = errors.New("unknown trait")
	ErrConflictingTraits     = errors.New("these traits cannot be combined")
	ErrInvalidModel          = errors.New("model must be sonnet, opus, haiku or a claude-… model id")
	ErrInvalidPermissionMode = errors.New("permission mode must be manual, acceptEdits, plan, dontAsk or auto")
	ErrTooManyTools          = errors.New("at most 100 allowed tools")
	ErrInvalidTool           = errors.New("an allowed tool is 1 to 200 characters on one line")
	ErrInvalidPriority       = errors.New("a work priority is 1 to 4 for a work type key")
	ErrStale                 = errors.New("the agent changed since that revision")
	ErrDeleted               = errors.New("the agent is deleted")
)

// PermissionModes are the Claude CLI permission modes an agent may run in.
// bypassPermissions is never one of them.
var PermissionModes = []string{"manual", "acceptEdits", "plan", "dontAsk", "auto"}

var (
	slugPattern     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	modelPattern    = regexp.MustCompile(`^(sonnet|opus|haiku|claude-[a-z0-9.-]+)$`)
	workTypePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
)

// Profile is everything about an agent a member writes: who it is, how it
// runs, its work priorities and its skillset.
type Profile struct {
	Name           string
	Title          string
	Backstory      string
	Traits         []string
	Model          string
	PermissionMode string
	AllowedTools   []string
	PortraitSeed   string
	// WorkPriorities maps a work type key to 1 (first) … 4 (last); a key
	// that is missing is off.
	WorkPriorities map[string]int
	Files          []File
}

// Agent is a configured Claude CLI worker owned by one member.
type Agent struct {
	ID       uint64
	OwnerID  uint64
	Slug     string
	Revision int
	Profile
	// OriginAgentID and OriginRevision are set on a recruited copy.
	OriginAgentID  *uint64
	OriginRevision *int
	DeletedAt      *time.Time
	UpdatedAt      time.Time
}

// NewAgent makes an agent at revision 1.
func NewAgent(ownerID uint64, slug string, p Profile) (Agent, error) {
	if !validSlug(slug) {
		return Agent{}, ErrInvalidSlug
	}
	p, err := clean(p)
	if err != nil {
		return Agent{}, err
	}
	return Agent{OwnerID: ownerID, Slug: slug, Revision: 1, Profile: p}, nil
}

// Revise replaces the profile. basedOn is the revision the change was made
// from; any other revision means someone else changed the agent meanwhile.
func (a *Agent) Revise(basedOn int, p Profile) error {
	if a.DeletedAt != nil {
		return ErrDeleted
	}
	if basedOn != a.Revision {
		return ErrStale
	}
	p, err := clean(p)
	if err != nil {
		return err
	}
	a.Profile = p
	a.Revision++
	return nil
}

// Delete marks the agent deleted; it stays as a tombstone so other devices
// learn about it.
func (a *Agent) Delete(basedOn int, now time.Time) error {
	if a.DeletedAt != nil {
		return ErrDeleted
	}
	if basedOn != a.Revision {
		return ErrStale
	}
	a.DeletedAt = &now
	a.Revision++
	return nil
}

// Skills lists the agent's skills from their SKILL.md files.
func (a Agent) Skills() []Skill {
	skills, _ := CheckSkillset(a.Files)
	return skills
}

// Slugify makes a slug from a name: "Vera the Builder" → "vera-the-builder".
func Slugify(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 40 {
		s = strings.Trim(s[:40], "-")
	}
	if s == "" {
		s = "agent"
	}
	return s
}

func validSlug(s string) bool {
	return len(s) >= 1 && len(s) <= 40 && slugPattern.MatchString(s)
}

// clean checks a profile and returns it tidied (trimmed, defaults filled).
func clean(p Profile) (Profile, error) {
	p.Name = strings.TrimSpace(p.Name)
	if n := utf8.RuneCountInString(p.Name); n < 1 || n > 40 {
		return Profile{}, ErrInvalidName
	}
	p.Title = strings.TrimSpace(p.Title)
	if utf8.RuneCountInString(p.Title) > 60 {
		return Profile{}, ErrInvalidTitle
	}
	if utf8.RuneCountInString(p.Backstory) > 2000 {
		return Profile{}, ErrInvalidBackstory
	}
	seen := map[string]bool{}
	traits := []string{}
	for _, t := range p.Traits {
		if !traitKnown(t) {
			return Profile{}, ErrUnknownTrait
		}
		if seen[t] {
			continue
		}
		for other := range seen {
			if TraitsConflict(t, other) {
				return Profile{}, ErrConflictingTraits
			}
		}
		seen[t] = true
		traits = append(traits, t)
	}
	p.Traits = traits
	if p.Model == "" {
		p.Model = "sonnet"
	}
	if !modelPattern.MatchString(p.Model) {
		return Profile{}, ErrInvalidModel
	}
	if p.PermissionMode == "" {
		p.PermissionMode = "manual"
	}
	known := false
	for _, m := range PermissionModes {
		known = known || m == p.PermissionMode
	}
	if !known {
		return Profile{}, ErrInvalidPermissionMode
	}
	if len(p.AllowedTools) > 100 {
		return Profile{}, ErrTooManyTools
	}
	tools := []string{}
	for _, t := range p.AllowedTools {
		t = strings.TrimSpace(t)
		if t == "" || utf8.RuneCountInString(t) > 200 || strings.ContainsAny(t, "\r\n") {
			return Profile{}, ErrInvalidTool
		}
		tools = append(tools, t)
	}
	p.AllowedTools = tools
	priorities := map[string]int{}
	for key, v := range p.WorkPriorities {
		if !workTypePattern.MatchString(key) || v < 1 || v > 4 {
			return Profile{}, ErrInvalidPriority
		}
		priorities[key] = v
	}
	p.WorkPriorities = priorities
	if p.Files == nil {
		p.Files = []File{}
	}
	if _, err := CheckSkillset(p.Files); err != nil {
		return Profile{}, err
	}
	return p, nil
}
