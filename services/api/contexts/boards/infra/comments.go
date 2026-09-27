package infra

import (
	"context"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

type commentRecord struct {
	ID        uint64 `gorm:"primaryKey"`
	TaskID    uint64
	AuthorID  uint64
	Body      string
	CreatedAt time.Time
	EditedAt  *time.Time
}

func (commentRecord) TableName() string { return "task_comments" }

func (r commentRecord) toDomain() domain.Comment {
	return domain.Comment{ID: r.ID, TaskID: r.TaskID, AuthorID: r.AuthorID, Body: r.Body, CreatedAt: r.CreatedAt, EditedAt: r.EditedAt}
}

type Comments struct{}

func (Comments) Add(ctx context.Context, c domain.Comment) (domain.Comment, error) {
	rec := commentRecord{TaskID: c.TaskID, AuthorID: c.AuthorID, Body: c.Body, CreatedAt: c.CreatedAt}
	if err := query(ctx).Create(&rec); err != nil {
		return domain.Comment{}, err
	}
	return rec.toDomain(), nil
}

func (Comments) Save(ctx context.Context, c domain.Comment) error {
	_, err := query(ctx).Model(&commentRecord{}).Where("id", c.ID).Update(map[string]any{
		"body":      c.Body,
		"edited_at": c.EditedAt,
	})
	return err
}

func (Comments) Delete(ctx context.Context, id uint64) error {
	_, err := query(ctx).Where("id", id).Delete(&commentRecord{})
	return err
}

func (Comments) ByID(ctx context.Context, id uint64) (domain.Comment, bool, error) {
	var rec commentRecord
	if err := query(ctx).Where("id", id).FirstOrFail(&rec); err != nil {
		if notFound(err) {
			return domain.Comment{}, false, nil
		}
		return domain.Comment{}, false, err
	}
	return rec.toDomain(), true, nil
}

func (Comments) OfTask(ctx context.Context, taskID uint64) ([]domain.Comment, error) {
	var recs []commentRecord
	if err := query(ctx).Where("task_id", taskID).OrderBy("created_at").OrderBy("id").Find(&recs); err != nil {
		return nil, err
	}
	out := make([]domain.Comment, len(recs))
	for i, r := range recs {
		out[i] = r.toDomain()
	}
	return out, nil
}
