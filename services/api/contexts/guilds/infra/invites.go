package infra

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/guilds/app"
	"github.com/jevido/the-bakery/services/api/contexts/guilds/domain"
)

type inviteRecord struct {
	ID        uint64 `gorm:"primaryKey"`
	GuildID   uint64
	Code      string
	CreatedBy uint64
	ExpiresAt *time.Time
	MaxUses   *int
	Uses      int
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (inviteRecord) TableName() string { return "guild_invites" }

func (r inviteRecord) toDomain() domain.Invite {
	return domain.Invite{
		ID: r.ID, GuildID: r.GuildID, CreatedBy: r.CreatedBy, Code: r.Code,
		ExpiresAt: r.ExpiresAt, MaxUses: r.MaxUses, Uses: r.Uses, RevokedAt: r.RevokedAt,
	}
}

type Invites struct{}

func (Invites) Add(ctx context.Context, inv domain.Invite) (domain.Invite, error) {
	rec := inviteRecord{GuildID: inv.GuildID, Code: inv.Code, CreatedBy: inv.CreatedBy, ExpiresAt: inv.ExpiresAt, MaxUses: inv.MaxUses}
	if err := query(ctx).Create(&rec); err != nil {
		if isUniqueViolation(err) {
			return domain.Invite{}, app.ErrCodeTaken
		}
		return domain.Invite{}, err
	}
	return rec.toDomain(), nil
}

func (Invites) ByID(ctx context.Context, id uint64) (domain.Invite, bool, error) {
	return firstInvite(ctx, "id", id)
}

func (Invites) ByCode(ctx context.Context, code string) (domain.Invite, bool, error) {
	return firstInvite(ctx, "code", code)
}

func firstInvite(ctx context.Context, column string, value any) (domain.Invite, bool, error) {
	var rec inviteRecord
	if err := query(ctx).Where(column, value).FirstOrFail(&rec); err != nil {
		if isNotFound(err) {
			return domain.Invite{}, false, nil
		}
		return domain.Invite{}, false, err
	}
	return rec.toDomain(), true, nil
}

func (Invites) OfGuild(ctx context.Context, guildID uint64) ([]domain.Invite, error) {
	var recs []inviteRecord
	if err := query(ctx).Where("guild_id", guildID).OrderByDesc("id").Find(&recs); err != nil {
		return nil, err
	}
	out := make([]domain.Invite, len(recs))
	for i, r := range recs {
		out[i] = r.toDomain()
	}
	return out, nil
}

func (Invites) Revoke(ctx context.Context, inv domain.Invite) error {
	_, err := query(ctx).Model(&inviteRecord{}).Where("id", inv.ID).Update("revoked_at", inv.RevokedAt)
	return err
}

func (Invites) CountUse(ctx context.Context, id uint64) error {
	_, err := query(ctx).Exec("UPDATE guild_invites SET uses = uses + 1, updated_at = NOW() WHERE id = ?", id)
	return err
}

// codeAlphabet leaves out letters and digits that are easy to misread.
const codeAlphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"

// Codes makes 10-character invite codes from crypto/rand.
type Codes struct{}

func (Codes) NewCode() (string, error) {
	b := make([]byte, 10)
	n := big.NewInt(int64(len(codeAlphabet)))
	for i := range b {
		v, err := rand.Int(rand.Reader, n) // uniform, no modulo bias
		if err != nil {
			return "", err
		}
		b[i] = codeAlphabet[v.Int64()]
	}
	return string(b), nil
}

func isNotFound(err error) bool {
	return errors.Is(err, errRecordNotFound)
}
