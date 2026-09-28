package infra

import (
	"context"
	"strings"
	"time"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

type signalRecord struct {
	ID        uint64 `gorm:"primaryKey"`
	Kind      string
	LimitName string
	MemberID  *uint64
	IP        string `gorm:"column:ip"`
	At        time.Time
}

func (signalRecord) TableName() string { return "signals" }

type Signals struct{}

func (Signals) Add(ctx context.Context, s domain.Signal) error {
	rec := signalRecord{Kind: s.Kind, LimitName: s.Limit, IP: s.IP, At: s.At}
	if s.MemberID != 0 {
		rec.MemberID = &s.MemberID
	}
	return query(ctx).Create(&rec)
}

type countRow struct {
	IP       string    `db:"ip"`
	MemberID *uint64   `db:"member_id"`
	N        int       `db:"n"`
	Limits   *string   `db:"limits"`
	Last     time.Time `db:"last"`
}

func (Signals) ByIP(ctx context.Context, kind string, since time.Time, limit int) ([]app.Count, error) {
	return counts(ctx, "ip", "ip <> ''", kind, since, limit)
}

func (Signals) ByMember(ctx context.Context, kind string, since time.Time, limit int) ([]app.Count, error) {
	return counts(ctx, "member_id", "member_id IS NOT NULL", kind, since, limit)
}

// counts groups signals by one column; group and present are fixed SQL,
// never input. The placeholders are Postgres' own ($1): Goravel's ?
// rewriting miscounts them next to the ” literals.
func counts(ctx context.Context, group, present, kind string, since time.Time, limit int) ([]app.Count, error) {
	var rows []countRow
	err := facades.DB().WithContext(ctx).Select(&rows,
		`SELECT `+group+`, COUNT(*) AS n,
			STRING_AGG(DISTINCT limit_name, ',') FILTER (WHERE limit_name <> '') AS limits, MAX(at) AS last
		FROM signals WHERE kind = $1 AND at >= $2 AND `+present+`
		GROUP BY `+group+` ORDER BY n DESC, last DESC LIMIT $3`,
		kind, since, limit)
	if err != nil {
		return nil, err
	}
	out := make([]app.Count, len(rows))
	for i, r := range rows {
		out[i] = app.Count{IP: r.IP, Count: r.N, Last: r.Last}
		if r.MemberID != nil {
			out[i].MemberID = *r.MemberID
		}
		if r.Limits != nil && *r.Limits != "" {
			out[i].Limits = strings.Split(*r.Limits, ",")
		}
	}
	return out, nil
}
