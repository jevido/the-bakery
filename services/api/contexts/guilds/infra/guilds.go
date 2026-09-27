// Package infra stores guilds and memberships with the Goravel ORM.
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
	"github.com/jevido/the-bakery/services/api/contexts/guilds/domain"
)

type guildRecord struct {
	ID   uint64 `gorm:"primaryKey"`
	Name string
	orm.Timestamps
}

func (guildRecord) TableName() string { return "guilds" }

type membershipRecord struct {
	GuildID  uint64 `gorm:"primaryKey;autoIncrement:false"`
	MemberID uint64 `gorm:"primaryKey;autoIncrement:false"`
	JoinedAt time.Time
}

func (membershipRecord) TableName() string { return "guild_memberships" }

func query(ctx context.Context) contractsorm.Query {
	return facades.Orm().WithContext(ctx).Query()
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key")
}

type Guilds struct{}

func (Guilds) Add(ctx context.Context, g domain.Guild) (domain.Guild, error) {
	rec := guildRecord{Name: g.Name}
	err := facades.Orm().WithContext(ctx).Transaction(func(tx contractsorm.Query) error {
		if err := tx.Create(&rec); err != nil {
			return err
		}
		now := time.Now()
		for _, memberID := range g.MemberIDs() {
			if err := tx.Create(&membershipRecord{GuildID: rec.ID, MemberID: memberID, JoinedAt: now}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.Guild{}, err
	}
	return domain.Rehydrate(rec.ID, rec.Name, g.MemberIDs()), nil
}

func (Guilds) ByID(ctx context.Context, id uint64) (domain.Guild, bool, error) {
	var rec guildRecord
	if err := query(ctx).Where("id", id).FirstOrFail(&rec); err != nil {
		if errors.Is(err, frameworkerrors.OrmRecordNotFound) {
			return domain.Guild{}, false, nil
		}
		return domain.Guild{}, false, err
	}
	var memberIDs []uint64
	if err := query(ctx).Model(&membershipRecord{}).Where("guild_id", id).Pluck("member_id", &memberIDs); err != nil {
		return domain.Guild{}, false, err
	}
	return domain.Rehydrate(rec.ID, rec.Name, memberIDs), true, nil
}

func (Guilds) OfMember(ctx context.Context, memberID uint64) ([]domain.Guild, error) {
	var recs []guildRecord
	err := query(ctx).Model(&guildRecord{}).
		Select("guilds.*").
		Join("JOIN guild_memberships ON guild_memberships.guild_id = guilds.id").
		Where("guild_memberships.member_id", memberID).
		OrderBy("guilds.name").
		Find(&recs)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return []domain.Guild{}, nil
	}
	ids := make([]any, len(recs))
	for i, r := range recs {
		ids[i] = r.ID
	}
	var mems []membershipRecord
	if err := query(ctx).WhereIn("guild_id", ids).Find(&mems); err != nil {
		return nil, err
	}
	byGuild := map[uint64][]uint64{}
	for _, m := range mems {
		byGuild[m.GuildID] = append(byGuild[m.GuildID], m.MemberID)
	}
	guilds := make([]domain.Guild, len(recs))
	for i, r := range recs {
		guilds[i] = domain.Rehydrate(r.ID, r.Name, byGuild[r.ID])
	}
	return guilds, nil
}

func (Guilds) AddMembership(ctx context.Context, ev domain.MemberJoined) error {
	err := query(ctx).Create(&membershipRecord{GuildID: ev.GuildID, MemberID: ev.MemberID, JoinedAt: time.Now()})
	if err != nil && isUniqueViolation(err) {
		// Two requests adding the same member raced past the aggregate.
		return domain.ErrAlreadyMember
	}
	return err
}

// ByName finds a guild by its name, for the dev seeder only: names are not
// unique.
func (g Guilds) ByName(ctx context.Context, name string) (domain.Guild, bool, error) {
	var rec guildRecord
	if err := query(ctx).Where("name", name).OrderBy("id").FirstOrFail(&rec); err != nil {
		if errors.Is(err, frameworkerrors.OrmRecordNotFound) {
			return domain.Guild{}, false, nil
		}
		return domain.Guild{}, false, err
	}
	return g.ByID(ctx, rec.ID)
}

// Memberships answers IsMember with one query instead of loading the guild.
type Memberships struct{}

func (Memberships) IsMember(ctx context.Context, guildID, memberID uint64) (bool, error) {
	return query(ctx).Model(&membershipRecord{}).Where("guild_id", guildID).Where("member_id", memberID).Exists()
}

// LogEvents writes guilds' domain events to the log; nothing subscribes to
// them yet.
type LogEvents struct{}

func (LogEvents) GuildFounded(ctx context.Context, ev domain.GuildFounded) {
	facades.Log().WithContext(ctx).Infof("GuildFounded guild=%d name=%q founder=%d", ev.GuildID, ev.Name, ev.FounderID)
}

func (LogEvents) MemberJoined(ctx context.Context, ev domain.MemberJoined) {
	facades.Log().WithContext(ctx).Infof("MemberJoined guild=%d member=%d", ev.GuildID, ev.MemberID)
}
