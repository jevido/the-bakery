package domain

import "fmt"

// Recruit copies a shared agent into the recruiter's roster: same profile
// and skillset, the recruiter as owner, revision 1, and the origin kept so
// updates can be pulled later. Work priorities are copied as a start; the
// recruiter tunes them after.
func Recruit(source Agent, recruiterID uint64, slug string) (Agent, error) {
	if source.DeletedAt != nil {
		return Agent{}, ErrDeleted
	}
	p := source.Profile
	p.Traits = append([]string{}, p.Traits...)
	p.AllowedTools = append([]string{}, p.AllowedTools...)
	p.Files = append([]File{}, p.Files...)
	priorities := make(map[string]int, len(p.WorkPriorities))
	for k, v := range p.WorkPriorities {
		priorities[k] = v
	}
	p.WorkPriorities = priorities
	a, err := NewAgent(recruiterID, slug, p)
	if err != nil {
		return Agent{}, err
	}
	originID, originRevision := source.ID, source.Revision
	a.OriginAgentID, a.OriginRevision = &originID, &originRevision
	return a, nil
}

// PullOrigin takes the origin's profile and skillset into a recruited copy,
// keeping the copy's own work priorities, as a revision based on basedOn.
func (a *Agent) PullOrigin(basedOn int, origin Agent) error {
	if a.OriginAgentID == nil || *a.OriginAgentID != origin.ID {
		return fmt.Errorf("agent %d was not recruited from agent %d", a.ID, origin.ID)
	}
	if origin.DeletedAt != nil {
		return ErrDeleted
	}
	p := origin.Profile
	p.WorkPriorities = a.WorkPriorities
	p.PortraitSeed = a.PortraitSeed
	if err := a.Revise(basedOn, p); err != nil {
		return err
	}
	rev := origin.Revision
	a.OriginRevision = &rev
	return nil
}

// SlugFor returns slug, or slug-2, slug-3, … when taken reports it in use.
func SlugFor(slug string, taken func(string) bool) string {
	if !taken(slug) {
		return slug
	}
	for n := 2; ; n++ {
		suffix := fmt.Sprintf("-%d", n)
		base := slug
		if len(base)+len(suffix) > 40 {
			base = base[:40-len(suffix)]
		}
		if !taken(base + suffix) {
			return base + suffix
		}
	}
}
