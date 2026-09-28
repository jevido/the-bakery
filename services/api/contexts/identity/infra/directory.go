package infra

import (
	"context"
	"strings"
	"time"
)

// Listed is a member as the platform's directory lists them.
type Listed struct {
	ID          uint64
	Email       string
	DisplayName string
	JoinedAt    time.Time
}

// Directory searches every member, for the operator console.
type Directory struct{}

// Find lists members whose email or display name contains q (any case),
// newest first; an empty q lists the newest.
func (Directory) Find(ctx context.Context, q string, limit int) ([]Listed, error) {
	query := (Members{}).query(ctx)
	if q = strings.TrimSpace(q); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		query = query.Where("(LOWER(email) LIKE ? OR LOWER(display_name) LIKE ?)", like, like)
	}
	var recs []memberRecord
	if err := query.OrderByDesc("id").Limit(limit).Find(&recs); err != nil {
		return nil, err
	}
	return listed(recs), nil
}

// ByIDs lists the given members.
func (Directory) ByIDs(ctx context.Context, ids []uint64) ([]Listed, error) {
	if len(ids) == 0 {
		return []Listed{}, nil
	}
	in := make([]any, len(ids))
	for i, id := range ids {
		in[i] = id
	}
	var recs []memberRecord
	if err := (Members{}).query(ctx).WhereIn("id", in).OrderBy("display_name").Find(&recs); err != nil {
		return nil, err
	}
	return listed(recs), nil
}

func listed(recs []memberRecord) []Listed {
	out := make([]Listed, len(recs))
	for i, r := range recs {
		out[i] = Listed{ID: r.ID, Email: r.Email, DisplayName: r.DisplayName, JoinedAt: joinedAt(r)}
	}
	return out
}

func joinedAt(r memberRecord) time.Time {
	if r.CreatedAt == nil {
		return time.Time{}
	}
	return r.CreatedAt.StdTime()
}
