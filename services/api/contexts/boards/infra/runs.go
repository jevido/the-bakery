package infra

import (
	"context"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

type runRecord struct {
	ID           uint64 `gorm:"primaryKey"`
	TaskID       uint64
	MemberID     uint64
	AgentID      uint64
	AgentName    string
	Kind         string
	Machine      string
	Branch       string
	Status       string
	StartedAt    time.Time
	EndedAt      *time.Time
	CostUSD      float64 `gorm:"column:cost_usd"`
	Turns        int
	Summary      string
	FilesChanged int
	Additions    int
	Deletions    int
}

func (runRecord) TableName() string { return "task_runs" }

func runRecordOf(r domain.Run) runRecord {
	return runRecord{
		ID: r.ID, TaskID: r.TaskID, MemberID: r.MemberID, AgentID: r.AgentID, AgentName: r.AgentName, Kind: r.Kind,
		Machine: r.Machine, Branch: r.Branch, Status: string(r.Status), StartedAt: r.StartedAt, EndedAt: r.EndedAt,
		CostUSD: r.CostUSD, Turns: r.Turns, Summary: r.Summary,
		FilesChanged: r.FilesChanged, Additions: r.Additions, Deletions: r.Deletions,
	}
}

func (r runRecord) toDomain() domain.Run {
	return domain.Run{
		ID: r.ID, TaskID: r.TaskID, MemberID: r.MemberID, AgentID: r.AgentID, AgentName: r.AgentName, Kind: r.Kind,
		Machine: r.Machine, Branch: r.Branch, Status: domain.RunStatus(r.Status), StartedAt: r.StartedAt, EndedAt: r.EndedAt,
		RunStats: domain.RunStats{
			CostUSD: r.CostUSD, Turns: r.Turns, Summary: r.Summary,
			FilesChanged: r.FilesChanged, Additions: r.Additions, Deletions: r.Deletions,
		},
	}
}

type Runs struct{}

func (Runs) Add(ctx context.Context, r domain.Run) (domain.Run, error) {
	rec := runRecordOf(r)
	if err := query(ctx).Create(&rec); err != nil {
		return domain.Run{}, err
	}
	return rec.toDomain(), nil
}

func (Runs) Save(ctx context.Context, r domain.Run) error {
	_, err := query(ctx).Model(&runRecord{}).Where("id", r.ID).Update(map[string]any{
		"status":        string(r.Status),
		"ended_at":      r.EndedAt,
		"cost_usd":      r.CostUSD,
		"turns":         r.Turns,
		"summary":       r.Summary,
		"files_changed": r.FilesChanged,
		"additions":     r.Additions,
		"deletions":     r.Deletions,
	})
	return err
}

func (Runs) ByID(ctx context.Context, id uint64) (domain.Run, bool, error) {
	var rec runRecord
	if err := query(ctx).Where("id", id).FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return domain.Run{}, false, nil
		}
		return domain.Run{}, false, err
	}
	return rec.toDomain(), true, nil
}

func (Runs) OfTask(ctx context.Context, taskID uint64, limit int) ([]domain.Run, error) {
	q := query(ctx).Where("task_id", taskID).OrderByDesc("started_at").OrderByDesc("id")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var recs []runRecord
	if err := q.Find(&recs); err != nil {
		return nil, err
	}
	out := make([]domain.Run, len(recs))
	for i, r := range recs {
		out[i] = r.toDomain()
	}
	return out, nil
}
