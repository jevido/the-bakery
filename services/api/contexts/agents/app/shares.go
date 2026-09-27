package app

import (
	"context"
	"errors"

	"github.com/jevido/the-bakery/services/api/contexts/agents/domain"
)

var (
	ErrNotMember = errors.New("not a member of this guild")
	ErrNotShared = errors.New("that agent is not shared with this guild")
)

// Memberships is guilds' answer to "is this member in this guild?".
type Memberships interface {
	IsMember(ctx context.Context, guildID, memberID uint64) (bool, error)
}

// MemberNames is identity's answer to "what are these members called?".
type MemberNames interface {
	DisplayNames(ctx context.Context, ids []uint64) (map[uint64]string, error)
}

// Shares stores which agent is shared with which guild.
type Shares interface {
	Add(ctx context.Context, agentID, guildID uint64) error
	Remove(ctx context.Context, agentID, guildID uint64) error
	// GuildsOf lists the guilds an agent is shared with.
	GuildsOf(ctx context.Context, agentID uint64) ([]uint64, error)
	// AgentsIn lists the ids of the agents shared with a guild.
	AgentsIn(ctx context.Context, guildID uint64) ([]uint64, error)
}

type (
	AgentShared    struct{ AgentID, GuildID uint64 }
	AgentUnshared  struct{ AgentID, GuildID uint64 }
	AgentRecruited struct {
		AgentID, RecruiterID, OriginAgentID, GuildID uint64
		OriginRevision                               int
	}
)

func (s *Service) requireMember(ctx context.Context, guildID, memberID uint64) error {
	ok, err := s.memberships.IsMember(ctx, guildID, memberID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotMember
	}
	return nil
}

// ShareAgent makes the owner's agent available to a guild they are in.
func (s *Service) ShareAgent(ctx context.Context, id, ownerID, guildID uint64) error {
	a, err := s.GetAgent(ctx, id, ownerID)
	if err != nil {
		return err
	}
	if err := s.requireMember(ctx, guildID, ownerID); err != nil {
		return err
	}
	if err := s.shares.Add(ctx, a.ID, guildID); err != nil {
		return err
	}
	s.events.Publish(ctx, AgentShared{AgentID: a.ID, GuildID: guildID})
	return nil
}

// UnshareAgent stops new recruits from the guild; copies already made stay.
func (s *Service) UnshareAgent(ctx context.Context, id, ownerID, guildID uint64) error {
	if _, err := s.own(ctx, id, ownerID); err != nil {
		return err
	}
	if err := s.shares.Remove(ctx, id, guildID); err != nil {
		return err
	}
	s.events.Publish(ctx, AgentUnshared{AgentID: id, GuildID: guildID})
	return nil
}

// SharedWith lists the guilds one of the member's agents is shared with.
func (s *Service) SharedWith(ctx context.Context, id, ownerID uint64) ([]uint64, error) {
	if _, err := s.own(ctx, id, ownerID); err != nil {
		return nil, err
	}
	return s.shares.GuildsOf(ctx, id)
}

// SharedAgent is an agent shared with a guild, with its owner's name.
type SharedAgent struct {
	domain.Agent
	OwnerName string
}

// ListGuildAgents lists the live agents shared with a guild, for its members.
func (s *Service) ListGuildAgents(ctx context.Context, guildID, memberID uint64) ([]SharedAgent, error) {
	if err := s.requireMember(ctx, guildID, memberID); err != nil {
		return nil, err
	}
	ids, err := s.shares.AgentsIn(ctx, guildID)
	if err != nil {
		return nil, err
	}
	var agents []domain.Agent
	owners := []uint64{}
	for _, id := range ids {
		a, found, err := s.agents.ByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if found && a.DeletedAt == nil {
			agents = append(agents, a)
			owners = append(owners, a.OwnerID)
		}
	}
	names, err := s.names.DisplayNames(ctx, owners)
	if err != nil {
		return nil, err
	}
	out := make([]SharedAgent, len(agents))
	for i, a := range agents {
		out[i] = SharedAgent{Agent: a, OwnerName: names[a.OwnerID]}
	}
	return out, nil
}

// VisibleAgent returns an agent the member owns, or one shared with a guild
// the member is in.
func (s *Service) VisibleAgent(ctx context.Context, id, memberID uint64) (SharedAgent, error) {
	a, found, err := s.agents.ByID(ctx, id)
	if err != nil {
		return SharedAgent{}, err
	}
	if !found || a.DeletedAt != nil {
		return SharedAgent{}, ErrAgentNotFound
	}
	if a.OwnerID != memberID {
		guilds, err := s.shares.GuildsOf(ctx, id)
		if err != nil {
			return SharedAgent{}, err
		}
		visible := false
		for _, g := range guilds {
			if ok, err := s.memberships.IsMember(ctx, g, memberID); err != nil {
				return SharedAgent{}, err
			} else if ok {
				visible = true
				break
			}
		}
		if !visible {
			return SharedAgent{}, ErrAgentNotFound
		}
	}
	names, err := s.names.DisplayNames(ctx, []uint64{a.OwnerID})
	if err != nil {
		return SharedAgent{}, err
	}
	return SharedAgent{Agent: a, OwnerName: names[a.OwnerID]}, nil
}

// RecruitAgent copies an agent shared with the guild into the member's
// roster. A slug the member already uses gets -2, -3, … appended.
func (s *Service) RecruitAgent(ctx context.Context, memberID, sourceID, guildID uint64, slug string) (domain.Agent, error) {
	if err := s.requireMember(ctx, guildID, memberID); err != nil {
		return domain.Agent{}, err
	}
	guilds, err := s.shares.GuildsOf(ctx, sourceID)
	if err != nil {
		return domain.Agent{}, err
	}
	shared := false
	for _, g := range guilds {
		shared = shared || g == guildID
	}
	source, found, err := s.agents.ByID(ctx, sourceID)
	if err != nil {
		return domain.Agent{}, err
	}
	if !shared || !found || source.DeletedAt != nil {
		return domain.Agent{}, ErrNotShared
	}
	mine, err := s.agents.OfOwner(ctx, memberID, nil)
	if err != nil {
		return domain.Agent{}, err
	}
	used := map[string]bool{}
	for _, a := range mine {
		used[a.Slug] = true
	}
	if slug == "" {
		slug = source.Slug
	}
	a, err := domain.Recruit(source, memberID, domain.SlugFor(slug, func(s string) bool { return used[s] }))
	if err != nil {
		return domain.Agent{}, err
	}
	if a, err = s.agents.Add(ctx, a); err != nil {
		return domain.Agent{}, err
	}
	s.events.Publish(ctx, AgentRecruited{AgentID: a.ID, RecruiterID: memberID, OriginAgentID: sourceID, GuildID: guildID, OriginRevision: source.Revision})
	return a, nil
}

// OriginStatus says how a recruited copy stands against its origin.
type OriginStatus struct {
	OriginAgentID  uint64
	OriginRevision int // the revision the copy last took
	Current        int // the origin's revision now
	Newer          bool
	Gone           bool
}

func (s *Service) Origin(ctx context.Context, id, ownerID uint64) (OriginStatus, error) {
	a, err := s.GetAgent(ctx, id, ownerID)
	if err != nil {
		return OriginStatus{}, err
	}
	if a.OriginAgentID == nil {
		return OriginStatus{}, ErrAgentNotFound
	}
	st := OriginStatus{OriginAgentID: *a.OriginAgentID, OriginRevision: *a.OriginRevision}
	origin, found, err := s.agents.ByID(ctx, *a.OriginAgentID)
	if err != nil {
		return OriginStatus{}, err
	}
	if !found || origin.DeletedAt != nil {
		st.Gone = true
		return st, nil
	}
	st.Current = origin.Revision
	st.Newer = origin.Revision > st.OriginRevision
	return st, nil
}

// PullOrigin takes the origin's fields and skillset into the copy, keeping
// its work priorities.
func (s *Service) PullOrigin(ctx context.Context, id, ownerID uint64, basedOn int) (domain.Agent, error) {
	a, err := s.GetAgent(ctx, id, ownerID)
	if err != nil {
		return domain.Agent{}, err
	}
	if a.OriginAgentID == nil {
		return domain.Agent{}, ErrAgentNotFound
	}
	origin, found, err := s.agents.ByID(ctx, *a.OriginAgentID)
	if err != nil {
		return domain.Agent{}, err
	}
	if !found || origin.DeletedAt != nil {
		return domain.Agent{}, ErrAgentNotFound
	}
	current := a
	if err := a.PullOrigin(basedOn, origin); err != nil {
		if errors.Is(err, domain.ErrStale) {
			return current, err
		}
		return domain.Agent{}, err
	}
	if err := s.agents.Replace(ctx, a, basedOn); err != nil {
		if errors.Is(err, domain.ErrStale) {
			fresh, _, _ := s.agents.ByID(ctx, id)
			return fresh, err
		}
		return domain.Agent{}, err
	}
	s.events.Publish(ctx, AgentRevised{AgentID: a.ID, OwnerID: ownerID, Revision: a.Revision})
	return s.reload(ctx, a.ID)
}
