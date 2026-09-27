package infra

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	frameworkerrors "github.com/goravel/framework/errors"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/identity/domain"
)

type tokenRecord struct {
	ID         uint64 `gorm:"primaryKey"`
	MemberID   uint64
	Name       string
	TokenHash  string
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (tokenRecord) TableName() string { return "personal_tokens" }

func (r tokenRecord) toDomain() domain.PersonalToken {
	return domain.PersonalToken{ID: r.ID, MemberID: r.MemberID, Name: r.Name, Hash: r.TokenHash,
		CreatedAt: r.CreatedAt, LastUsedAt: r.LastUsedAt, RevokedAt: r.RevokedAt}
}

type Tokens struct{}

func (Tokens) Add(ctx context.Context, t domain.PersonalToken) (domain.PersonalToken, error) {
	rec := tokenRecord{MemberID: t.MemberID, Name: t.Name, TokenHash: t.Hash}
	if err := facades.Orm().WithContext(ctx).Query().Create(&rec); err != nil {
		return domain.PersonalToken{}, err
	}
	return rec.toDomain(), nil
}

// OfMember lists the member's tokens that still work, newest first.
func (Tokens) OfMember(ctx context.Context, memberID uint64) ([]domain.PersonalToken, error) {
	var recs []tokenRecord
	err := facades.Orm().WithContext(ctx).Query().Where("member_id", memberID).WhereNull("revoked_at").OrderByDesc("id").Find(&recs)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PersonalToken, len(recs))
	for i, r := range recs {
		out[i] = r.toDomain()
	}
	return out, nil
}

func (Tokens) ByID(ctx context.Context, id uint64) (domain.PersonalToken, bool, error) {
	return firstToken(ctx, "id", id)
}

func (Tokens) ByHash(ctx context.Context, hash string) (domain.PersonalToken, bool, error) {
	return firstToken(ctx, "token_hash", hash)
}

func firstToken(ctx context.Context, column string, value any) (domain.PersonalToken, bool, error) {
	var rec tokenRecord
	if err := facades.Orm().WithContext(ctx).Query().Where(column, value).FirstOrFail(&rec); err != nil {
		if errors.Is(err, frameworkerrors.OrmRecordNotFound) {
			return domain.PersonalToken{}, false, nil
		}
		return domain.PersonalToken{}, false, err
	}
	return rec.toDomain(), true, nil
}

func (Tokens) Revoke(ctx context.Context, t domain.PersonalToken) error {
	_, err := facades.Orm().WithContext(ctx).Query().Model(&tokenRecord{}).Where("id", t.ID).Update("revoked_at", t.RevokedAt)
	return err
}

func (Tokens) Touch(ctx context.Context, id uint64, at time.Time) error {
	_, err := facades.Orm().WithContext(ctx).Query().Model(&tokenRecord{}).Where("id", id).Update("last_used_at", at)
	return err
}

// Secrets makes `bky_` + 32 URL-safe random characters (24 random bytes).
type Secrets struct{}

func (Secrets) NewSecret() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return domain.PersonalTokenPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}
