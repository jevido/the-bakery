// Package infra stores members with the Goravel ORM and hashes passwords
// with Goravel's hasher.
package infra

import (
	"context"
	"errors"
	"strings"

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
