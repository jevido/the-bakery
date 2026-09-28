package app

import "context"

// AuditLog records a sensitive action in the platform's audit log (the
// moderation context's, set when the contexts are wired). Recording never
// fails the action.
type AuditLog interface {
	Record(ctx context.Context, e AuditRecord)
}

// AuditRecord is one sensitive action by a member.
type AuditRecord struct {
	ActorID    uint64
	Action     string
	TargetKind string
	TargetID   uint64
	Meta       map[string]any
}

type noAudit struct{}

func (noAudit) Record(context.Context, AuditRecord) {}

// SetSanctionCheck sets the moderation context's guild check: nil, or an
// error (a *refusal.Refusal) when the guild is sanctioned.
func (s *Service) SetSanctionCheck(f func(ctx context.Context, guildID uint64) error) { s.sanction = f }

func (s *Service) checkSanction(ctx context.Context, guildID uint64) error {
	if s.sanction == nil {
		return nil
	}
	return s.sanction(ctx, guildID)
}

// SetAuditLog sets where sensitive actions are recorded.
func (s *Service) SetAuditLog(a AuditLog) { s.audit = a }

func (s *Service) auditLog() AuditLog {
	if s.audit == nil {
		return noAudit{}
	}
	return s.audit
}
