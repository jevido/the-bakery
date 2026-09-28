// Package infra stores the moderation context and talks to the framework.
package infra

import (
	"context"
	"errors"
	"strings"
	"time"

	contractsorm "github.com/goravel/framework/contracts/database/orm"
	frameworkerrors "github.com/goravel/framework/errors"
	"github.com/pquerna/otp/totp"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

func query(ctx context.Context) contractsorm.Query {
	return facades.Orm().WithContext(ctx).Query()
}

func notFound(err error) bool {
	return errors.Is(err, frameworkerrors.OrmRecordNotFound)
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key")
}

type operatorRecord struct {
	ID              uint64 `gorm:"primaryKey"`
	Email           string
	PasswordHash    string
	TOTPSecret      string     `gorm:"column:totp_secret"`
	TOTPConfirmedAt *time.Time `gorm:"column:totp_confirmed_at"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (operatorRecord) TableName() string { return "operators" }

func (r operatorRecord) toDomain() domain.Operator {
	return domain.Operator{ID: r.ID, Email: r.Email, PasswordHash: r.PasswordHash, TOTPSecret: r.TOTPSecret, ConfirmedAt: r.TOTPConfirmedAt}
}

type Operators struct{}

func (Operators) Add(ctx context.Context, o domain.Operator) (domain.Operator, error) {
	rec := operatorRecord{Email: o.Email, PasswordHash: o.PasswordHash, TOTPSecret: o.TOTPSecret}
	if err := query(ctx).Create(&rec); err != nil {
		if isUniqueViolation(err) {
			return domain.Operator{}, app.ErrOperatorExists
		}
		return domain.Operator{}, err
	}
	return rec.toDomain(), nil
}

func (Operators) Save(ctx context.Context, o domain.Operator) error {
	_, err := query(ctx).Model(&operatorRecord{}).Where("id", o.ID).Update(map[string]any{
		"password_hash":     o.PasswordHash,
		"totp_confirmed_at": o.ConfirmedAt,
	})
	return err
}

func (Operators) ByID(ctx context.Context, id uint64) (domain.Operator, bool, error) {
	return operatorBy(ctx, "id", id)
}

func (Operators) ByEmail(ctx context.Context, email string) (domain.Operator, bool, error) {
	return operatorBy(ctx, "email", strings.ToLower(strings.TrimSpace(email)))
}

func operatorBy(ctx context.Context, column string, value any) (domain.Operator, bool, error) {
	var rec operatorRecord
	if err := query(ctx).Where(column, value).FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return domain.Operator{}, false, nil
		}
		return domain.Operator{}, false, err
	}
	return rec.toDomain(), true, nil
}

type Hasher struct{}

func (Hasher) Hash(password string) (string, error) { return facades.Hash().Make(password) }
func (Hasher) Check(password, hash string) bool     { return facades.Hash().Check(password, hash) }

// Secrets encrypts with the app key (APP_KEY).
type Secrets struct{}

func (Secrets) Encrypt(plain string) (string, error) { return facades.Crypt().EncryptString(plain) }
func (Secrets) Decrypt(encrypted string) (string, error) {
	return facades.Crypt().DecryptString(encrypted)
}

// TOTP uses RFC 6238 codes: 6 digits, 30 seconds, one step of clock skew.
type TOTP struct{}

func (TOTP) Generate(account string) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "The Bakery console", AccountName: account})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

func (TOTP) Validate(code, secret string) bool {
	return totp.Validate(strings.TrimSpace(code), secret)
}
