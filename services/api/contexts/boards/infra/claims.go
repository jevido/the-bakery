package infra

import (
	"context"
	"time"

	contractsorm "github.com/goravel/framework/contracts/database/orm"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

type claimRecord struct {
	ID         uint64 `gorm:"primaryKey"`
	TaskID     uint64
	AgentID    uint64
	MemberID   uint64
	MachineID  string
	ClaimedAt  time.Time
	ExpiresAt  time.Time
	ReleasedAt *time.Time
}

func (claimRecord) TableName() string { return "task_claims" }

func (r claimRecord) toDomain() domain.Claim {
	return domain.Claim{
		ID: r.ID, TaskID: r.TaskID, AgentID: r.AgentID, MemberID: r.MemberID, MachineID: r.MachineID,
		ClaimedAt: r.ClaimedAt, ExpiresAt: r.ExpiresAt, ReleasedAt: r.ReleasedAt,
	}
}

type Claims struct{}

// Current is the task's unreleased claim, expired or not.
func (Claims) Current(ctx context.Context, taskID uint64) (*domain.Claim, error) {
	var rec claimRecord
	if err := query(ctx).Where("task_id", taskID).Where("released_at IS NULL").FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return nil, nil
		}
		return nil, err
	}
	c := rec.toDomain()
	return &c, nil
}

// Add stores a claim. In the same transaction it first releases the
// task's expired claim, so the one-unreleased-claim index lets the new one
// in; a claim that is still active makes the insert fail as
// app.ErrTaskClaimed, also when two machines claim at once.
func (Claims) Add(ctx context.Context, c domain.Claim) (domain.Claim, error) {
	rec := claimRecord{TaskID: c.TaskID, AgentID: c.AgentID, MemberID: c.MemberID, MachineID: c.MachineID, ClaimedAt: c.ClaimedAt, ExpiresAt: c.ExpiresAt}
	err := facades.Orm().WithContext(ctx).Transaction(func(tx contractsorm.Query) error {
		if _, err := tx.Exec(`UPDATE task_claims SET released_at = expires_at WHERE task_id = ? AND released_at IS NULL AND expires_at <= ?`, c.TaskID, c.ClaimedAt); err != nil {
			return err
		}
		return tx.Create(&rec)
	})
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Claim{}, domain.ErrTaskClaimed
		}
		return domain.Claim{}, err
	}
	return rec.toDomain(), nil
}

func (Claims) Save(ctx context.Context, c domain.Claim) error {
	_, err := query(ctx).Model(&claimRecord{}).Where("id", c.ID).Update(map[string]any{
		"expires_at":  c.ExpiresAt,
		"released_at": c.ReleasedAt,
	})
	return err
}

func (Claims) ByID(ctx context.Context, id uint64) (domain.Claim, bool, error) {
	var rec claimRecord
	if err := query(ctx).Where("id", id).FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return domain.Claim{}, false, nil
		}
		return domain.Claim{}, false, err
	}
	return rec.toDomain(), true, nil
}

// ActiveOf returns the claims holding any of the tasks at now, by task.
func (Claims) ActiveOf(ctx context.Context, taskIDs []uint64, now time.Time) (map[uint64]domain.Claim, error) {
	out := map[uint64]domain.Claim{}
	if len(taskIDs) == 0 {
		return out, nil
	}
	var recs []claimRecord
	if err := query(ctx).Raw(`SELECT * FROM task_claims WHERE task_id IN ? AND released_at IS NULL AND expires_at > ?`, taskIDs, now).Scan(&recs); err != nil {
		return nil, err
	}
	for _, r := range recs {
		out[r.TaskID] = r.toDomain()
	}
	return out, nil
}

var _ app.Claims = Claims{}
