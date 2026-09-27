// Package infra stores members with the Goravel ORM and hashes passwords
// with Goravel's hasher.
package infra

import (
	"context"
	"errors"
	"strings"
	"time"

	contractsorm "github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/database/orm"
	frameworkerrors "github.com/goravel/framework/errors"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/identity/app"
	"github.com/jevido/the-bakery/services/api/contexts/identity/domain"
)

type memberRecord struct {
	ID           uint64 `gorm:"primaryKey"`
	Email        string
	DisplayName  string
	PasswordHash string
	orm.Timestamps
}

func (memberRecord) TableName() string { return "members" }

func (r memberRecord) toDomain() domain.Member {
	return domain.Member{ID: r.ID, Email: r.Email, DisplayName: r.DisplayName, PasswordHash: r.PasswordHash}
}

type Members struct{}

func (Members) query(ctx context.Context) contractsorm.Query {
	return facades.Orm().WithContext(ctx).Query()
}

func (m Members) Add(ctx context.Context, member domain.Member) (domain.Member, error) {
	rec := memberRecord{Email: member.Email, DisplayName: member.DisplayName, PasswordHash: member.PasswordHash}
	if err := m.query(ctx).Create(&rec); err != nil {
		// The unique index on email catches two registrations racing past
		// the service's own check.
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key") {
			return domain.Member{}, app.ErrEmailTaken
		}
		return domain.Member{}, err
	}
	return rec.toDomain(), nil
}

func (m Members) ByEmail(ctx context.Context, email string) (domain.Member, bool, error) {
	return m.first(m.query(ctx).Where("email", email))
}

func (m Members) ByID(ctx context.Context, id uint64) (domain.Member, bool, error) {
	return m.first(m.query(ctx).Where("id", id))
}

func (m Members) ByIDs(ctx context.Context, ids []uint64) ([]domain.Member, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	in := make([]any, len(ids))
	for i, id := range ids {
		in[i] = id
	}
	var recs []memberRecord
	if err := m.query(ctx).WhereIn("id", in).Find(&recs); err != nil {
		return nil, err
	}
	out := make([]domain.Member, len(recs))
	for i, r := range recs {
		out[i] = r.toDomain()
	}
	return out, nil
}

func (Members) first(q contractsorm.Query) (domain.Member, bool, error) {
	var rec memberRecord
	if err := q.FirstOrFail(&rec); err != nil {
		if errors.Is(err, frameworkerrors.OrmRecordNotFound) {
			return domain.Member{}, false, nil
		}
		return domain.Member{}, false, err
	}
	return rec.toDomain(), true, nil
}

type Hasher struct{}

func (Hasher) Hash(password string) (string, error) { return facades.Hash().Make(password) }
func (Hasher) Check(password, hash string) bool     { return facades.Hash().Check(password, hash) }

type handoffRecord struct {
	ID        uint64 `gorm:"primaryKey"`
	CodeHash  string
	MemberID  uint64
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (handoffRecord) TableName() string { return "web_handoffs" }

type Handoffs struct{}

func (Handoffs) Add(ctx context.Context, codeHash string, memberID uint64, expiresAt time.Time) error {
	return facades.Orm().WithContext(ctx).Query().Create(&handoffRecord{CodeHash: codeHash, MemberID: memberID, ExpiresAt: expiresAt, CreatedAt: time.Now()})
}

// Take marks the code used in one conditional update, so two redeems of the
// same code cannot both succeed.
func (Handoffs) Take(ctx context.Context, codeHash string, now time.Time) (uint64, bool, error) {
	q := facades.Orm().WithContext(ctx).Query()
	res, err := q.Model(&handoffRecord{}).
		Where("code_hash", codeHash).WhereNull("used_at").Where("expires_at > ?", now).
		Update("used_at", now)
	if err != nil || res.RowsAffected != 1 {
		return 0, false, err
	}
	var rec handoffRecord
	if err := facades.Orm().WithContext(ctx).Query().Where("code_hash", codeHash).FirstOrFail(&rec); err != nil {
		return 0, false, err
	}
	// Old codes are of no use to anyone; clear them out while we are here.
	_, _ = facades.Orm().WithContext(ctx).Query().Where("expires_at < ?", now.Add(-time.Hour)).Delete(&handoffRecord{})
	return rec.MemberID, true, nil
}
