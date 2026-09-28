package app

import (
	"context"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

// Signals stores abuse signals and counts them.
type Signals interface {
	Add(ctx context.Context, s domain.Signal) error
	// ByIP counts signals of a kind since a time per IP, most first.
	ByIP(ctx context.Context, kind string, since time.Time, limit int) ([]Count, error)
	// ByMember counts signals of a kind since a time per member (those with
	// one), most first.
	ByMember(ctx context.Context, kind string, since time.Time, limit int) ([]Count, error)
}

// Count is how many signals one IP or one member gave, and for limit hits
// which limits.
type Count struct {
	IP       string
	MemberID uint64
	Count    int
	Limits   []string
	Last     time.Time
}

// SignalsSummary is the console's Signals screen.
type SignalsSummary struct {
	Since             time.Time
	SignUpsByIP       []Count
	LimitHitsByMember []Count
	LimitHitsByIP     []Count
	GuildsFoundedBy   []Count
	MemberNames       map[uint64]string
}

// RecordSignal adds a signal. The IP comes from the request's context when
// not given. A failure is the caller's to log.
func (s *Service) RecordSignal(ctx context.Context, kind, limit string, memberID uint64, ip string) error {
	if ip == "" {
		ip, _ = ctx.Value(IPKey).(string)
	}
	sig, err := domain.NewSignal(kind, limit, memberID, ip, s.now())
	if err != nil {
		return err
	}
	return s.signals.Add(ctx, sig)
}

// SignalsSince summarises the signals of the last hours (1 to 720, default
// 24): top IPs by sign-ups, members and IPs by limit hits, members by
// guilds founded; 20 rows each.
func (s *Service) SignalsSince(ctx context.Context, hours int) (SignalsSummary, error) {
	if hours <= 0 || hours > 720 {
		hours = 24
	}
	since := s.now().Add(-time.Duration(hours) * time.Hour)
	out := SignalsSummary{Since: since}
	var err error
	if out.SignUpsByIP, err = s.signals.ByIP(ctx, domain.SignalSignUp, since, 20); err != nil {
		return out, err
	}
	if out.LimitHitsByMember, err = s.signals.ByMember(ctx, domain.SignalLimitHit, since, 20); err != nil {
		return out, err
	}
	if out.LimitHitsByIP, err = s.signals.ByIP(ctx, domain.SignalLimitHit, since, 20); err != nil {
		return out, err
	}
	if out.GuildsFoundedBy, err = s.signals.ByMember(ctx, domain.SignalGuildFounded, since, 20); err != nil {
		return out, err
	}
	var ids []uint64
	for _, c := range append(out.LimitHitsByMember, out.GuildsFoundedBy...) {
		ids = append(ids, c.MemberID)
	}
	if out.MemberNames, _, err = s.Names(ctx, ids, nil); err != nil {
		return out, err
	}
	return out, nil
}
