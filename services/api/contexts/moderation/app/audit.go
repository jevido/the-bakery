package app

import (
	"context"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

// IPKey is where a request's client IP sits in its context, for the audit
// log (set by the http middleware RememberIP).
type ipKey struct{}

var IPKey = ipKey{}

// AuditEntries stores the audit log. There is no update or delete.
type AuditEntries interface {
	Add(ctx context.Context, e domain.AuditEntry) error
	List(ctx context.Context, f AuditFilter) ([]domain.AuditEntry, error)
}

// AuditFilter narrows the log; zero fields match everything. Before pages
// back: only entries older than that id.
type AuditFilter struct {
	ActorKind  string
	ActorID    uint64
	Action     string
	TargetKind string
	TargetID   uint64
	From, To   *time.Time
	Before     uint64
	Limit      int
}

// Record writes an entry, stamped with the time and the request's IP. A
// failure is the caller's to log: the action it records has happened.
func (s *Service) Record(ctx context.Context, e domain.AuditEntry) error {
	e, err := domain.NewAuditEntry(e, s.now())
	if err != nil {
		return err
	}
	if ip, ok := ctx.Value(IPKey).(string); ok && e.IP == "" {
		e.IP = ip
	}
	return s.audit.Add(ctx, e)
}

// Audit lists the log, newest first.
func (s *Service) Audit(ctx context.Context, f AuditFilter) ([]domain.AuditEntry, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	return s.audit.List(ctx, f)
}
