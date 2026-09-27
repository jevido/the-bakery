package infra

import (
	"context"

	"github.com/goravel/framework/database/orm"

	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

type workTypeRecord struct {
	ID       uint64 `gorm:"primaryKey"`
	GuildID  uint64
	Key      string
	Name     string
	Position string
	orm.Timestamps
}

func (workTypeRecord) TableName() string { return "work_types" }

func (r workTypeRecord) toDomain() domain.WorkType {
	return domain.WorkType{ID: r.ID, GuildID: r.GuildID, Key: r.Key, Name: r.Name, Position: r.Position}
}

type WorkTypes struct{}

func (WorkTypes) OfGuild(ctx context.Context, guildID uint64) ([]domain.WorkType, error) {
	var recs []workTypeRecord
	if err := query(ctx).Where("guild_id", guildID).OrderBy("position").Find(&recs); err != nil {
		return nil, err
	}
	out := make([]domain.WorkType, len(recs))
	for i, r := range recs {
		out[i] = r.toDomain()
	}
	return out, nil
}

// AddDefaults inserts the defaults, skipping keys that exist, so two
// requests seeing an empty list at once both end with the same seven.
func (WorkTypes) AddDefaults(ctx context.Context, wts []domain.WorkType) error {
	for _, wt := range wts {
		if _, err := query(ctx).Exec(
			`INSERT INTO work_types (guild_id, key, name, position, created_at, updated_at)
			 VALUES (?, ?, ?, ?, now(), now()) ON CONFLICT (guild_id, key) DO NOTHING`,
			wt.GuildID, wt.Key, wt.Name, wt.Position,
		); err != nil {
			return err
		}
	}
	return nil
}

func (WorkTypes) Add(ctx context.Context, wt domain.WorkType) (domain.WorkType, error) {
	rec := workTypeRecord{GuildID: wt.GuildID, Key: wt.Key, Name: wt.Name, Position: wt.Position}
	if err := query(ctx).Create(&rec); err != nil {
		if isUniqueViolation(err) {
			return domain.WorkType{}, domain.ErrDuplicateWorkType
		}
		return domain.WorkType{}, err
	}
	return rec.toDomain(), nil
}

func (WorkTypes) Save(ctx context.Context, wt domain.WorkType) error {
	_, err := query(ctx).Model(&workTypeRecord{}).Where("id", wt.ID).Update(map[string]any{"name": wt.Name, "position": wt.Position})
	return err
}

func (WorkTypes) Delete(ctx context.Context, guildID uint64, key string) error {
	_, err := query(ctx).Where("guild_id", guildID).Where("key", key).Delete(&workTypeRecord{})
	return err
}

func (WorkTypes) InUse(ctx context.Context, guildID uint64, key string) (bool, error) {
	var n int64
	err := query(ctx).Raw(
		`SELECT count(*) FROM tasks t JOIN boards b ON b.id = t.board_id WHERE b.guild_id = ? AND t.work_type = ?`, guildID, key,
	).Scan(&n)
	return n > 0, err
}

var _ app.WorkTypes = WorkTypes{}
