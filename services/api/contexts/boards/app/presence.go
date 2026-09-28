package app

import (
	"context"
	"time"
)

// presenceTimeout is how long a stream may go without a ping before its
// member no longer counts as present. Streams ping every 20 seconds.
const presenceTimeout = 60 * time.Second

// Presence is one open stream on a board: who, and when it last pinged.
type Presence struct {
	ConnID   string
	BoardID  uint64
	MemberID uint64
	SeenAt   time.Time
}

// PresenceLog stores who has which board open. It is short-lived state, not
// history: rows go when their stream closes or goes quiet.
type PresenceLog interface {
	Enter(ctx context.Context, p Presence) error
	Seen(ctx context.Context, connID string, at time.Time) error
	// Leave removes the stream and reports whether it was still there.
	Leave(ctx context.Context, connID string) (bool, error)
	OnBoard(ctx context.Context, boardID uint64) ([]Presence, error)
	// Prune removes streams not seen since before and returns them.
	Prune(ctx context.Context, before time.Time) ([]Presence, error)
}

// PresenceChanged is announced when a stream opens on a board or leaves it.
// It is not a domain event: presence is not part of the model's history,
// only of the board event stream.
type PresenceChanged struct {
	BoardID      uint64
	ActorID      uint64
	DisplayName  string
	PortraitSeed string
	ConnID       string
	// State is "joined" or "left".
	State string
}

// PresentMember is one open stream with its member's display name.
type PresentMember struct {
	ConnID       string
	MemberID     uint64
	DisplayName  string
	PortraitSeed string
}

// EnterBoard records that the member opened the board's stream connID, and
// returns everyone present, including them.
func (s *Service) EnterBoard(ctx context.Context, boardID, memberID uint64, connID string) ([]PresentMember, error) {
	if err := s.WatchBoard(ctx, boardID, memberID); err != nil {
		return nil, err
	}
	s.pruneStale(ctx)
	if err := s.presence.Enter(ctx, Presence{ConnID: connID, BoardID: boardID, MemberID: memberID, SeenAt: s.now()}); err != nil {
		return nil, err
	}
	names, err := s.names.DisplayNames(ctx, []uint64{memberID})
	if err != nil {
		return nil, err
	}
	seeds, err := s.names.PortraitSeeds(ctx, []uint64{memberID})
	if err != nil {
		return nil, err
	}
	s.events.Publish(ctx, PresenceChanged{BoardID: boardID, ActorID: memberID, DisplayName: names[memberID], PortraitSeed: seeds[memberID], ConnID: connID, State: "joined"})
	return s.presentOn(ctx, boardID)
}

// StillOnBoard is a stream's ping: it keeps the member present, and drops
// streams nobody has heard from in a minute.
func (s *Service) StillOnBoard(ctx context.Context, connID string) error {
	if err := s.presence.Seen(ctx, connID, s.now()); err != nil {
		return err
	}
	s.pruneStale(ctx)
	return nil
}

// LeaveBoard records that the stream closed.
func (s *Service) LeaveBoard(ctx context.Context, boardID, memberID uint64, connID string) error {
	gone, err := s.presence.Leave(ctx, connID)
	if err != nil || !gone {
		return err
	}
	s.events.Publish(ctx, PresenceChanged{BoardID: boardID, ActorID: memberID, ConnID: connID, State: "left"})
	return nil
}

func (s *Service) pruneStale(ctx context.Context) {
	stale, err := s.presence.Prune(ctx, s.now().Add(-presenceTimeout))
	if err != nil {
		return
	}
	for _, p := range stale {
		s.events.Publish(ctx, PresenceChanged{BoardID: p.BoardID, ActorID: p.MemberID, ConnID: p.ConnID, State: "left"})
	}
}

func (s *Service) presentOn(ctx context.Context, boardID uint64) ([]PresentMember, error) {
	ps, err := s.presence.OnBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, len(ps))
	for i, p := range ps {
		ids[i] = p.MemberID
	}
	names, err := s.names.DisplayNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	seeds, err := s.names.PortraitSeeds(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]PresentMember, len(ps))
	for i, p := range ps {
		out[i] = PresentMember{ConnID: p.ConnID, MemberID: p.MemberID, DisplayName: names[p.MemberID], PortraitSeed: seeds[p.MemberID]}
	}
	return out, nil
}
