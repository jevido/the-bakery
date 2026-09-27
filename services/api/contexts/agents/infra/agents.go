// Package infra stores agents with the Goravel ORM.
package infra

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	contractsorm "github.com/goravel/framework/contracts/database/orm"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/agents/app"
	"github.com/jevido/the-bakery/services/api/contexts/agents/domain"
)

func query(ctx context.Context) contractsorm.Query {
	return facades.Orm().WithContext(ctx).Query()
}

// agentRecord stores an agent; the list and map fields are jsonb.
type agentRecord struct {
	ID             uint64 `gorm:"primaryKey"`
	OwnerMemberID  uint64
	Slug           string
	Name           string
	Title          string
	Backstory      string
	Traits         string `gorm:"type:jsonb"`
	Model          string
	PermissionMode string
	AllowedTools   string `gorm:"type:jsonb"`
	PortraitSeed   string
	WorkPriorities string `gorm:"type:jsonb"`
	Revision       int
	OriginAgentID  *uint64
	OriginRevision *int
	DeletedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (agentRecord) TableName() string { return "agents" }

type fileRecord struct {
	AgentID uint64
	Path    string
	Content string
}

func (fileRecord) TableName() string { return "agent_files" }

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func recordOf(a domain.Agent) agentRecord {
	return agentRecord{
		ID: a.ID, OwnerMemberID: a.OwnerID, Slug: a.Slug, Name: a.Name, Title: a.Title, Backstory: a.Backstory,
		Traits: toJSON(a.Traits), Model: a.Model, PermissionMode: a.PermissionMode, AllowedTools: toJSON(a.AllowedTools),
		PortraitSeed: a.PortraitSeed, WorkPriorities: toJSON(a.WorkPriorities), Revision: a.Revision,
		OriginAgentID: a.OriginAgentID, OriginRevision: a.OriginRevision, DeletedAt: a.DeletedAt,
	}
}

func (r agentRecord) toDomain() domain.Agent {
	a := domain.Agent{
		ID: r.ID, OwnerID: r.OwnerMemberID, Slug: r.Slug, Revision: r.Revision,
		OriginAgentID: r.OriginAgentID, OriginRevision: r.OriginRevision, DeletedAt: r.DeletedAt, UpdatedAt: r.UpdatedAt,
	}
	a.Name, a.Title, a.Backstory, a.Model, a.PermissionMode, a.PortraitSeed = r.Name, r.Title, r.Backstory, r.Model, r.PermissionMode, r.PortraitSeed
	_ = json.Unmarshal([]byte(r.Traits), &a.Traits)
	_ = json.Unmarshal([]byte(r.AllowedTools), &a.AllowedTools)
	_ = json.Unmarshal([]byte(r.WorkPriorities), &a.WorkPriorities)
	return a
}

type Agents struct{}

func (Agents) Add(ctx context.Context, a domain.Agent) (domain.Agent, error) {
	rec := recordOf(a)
	rec.ID = 0
	err := facades.Orm().WithContext(ctx).Transaction(func(tx contractsorm.Query) error {
		if err := tx.Create(&rec); err != nil {
			return err
		}
		return writeFiles(tx, rec.ID, a.Files)
	})
	if err != nil {
		if strings.Contains(err.Error(), "agents_owner_slug_unique") {
			return domain.Agent{}, app.ErrSlugTaken
		}
		return domain.Agent{}, err
	}
	out := rec.toDomain()
	out.Files = a.Files
	return out, nil
}

func (Agents) Replace(ctx context.Context, a domain.Agent, basedOn int) error {
	rec := recordOf(a)
	return facades.Orm().WithContext(ctx).Transaction(func(tx contractsorm.Query) error {
		res, err := tx.Model(&agentRecord{}).Where("id", a.ID).Where("revision", basedOn).Update(map[string]any{
			"slug": rec.Slug, "name": rec.Name, "title": rec.Title, "backstory": rec.Backstory, "traits": rec.Traits,
			"model": rec.Model, "permission_mode": rec.PermissionMode, "allowed_tools": rec.AllowedTools,
			"portrait_seed": rec.PortraitSeed, "work_priorities": rec.WorkPriorities, "revision": rec.Revision,
			"origin_agent_id": rec.OriginAgentID, "origin_revision": rec.OriginRevision, "deleted_at": rec.DeletedAt,
			"updated_at": time.Now(),
		})
		if err != nil {
			return err
		}
		if res.RowsAffected == 0 {
			return domain.ErrStale
		}
		if _, err := tx.Where("agent_id", a.ID).Delete(&fileRecord{}); err != nil {
			return err
		}
		if a.DeletedAt != nil {
			return nil // a tombstone keeps no files
		}
		return writeFiles(tx, a.ID, a.Files)
	})
}

func writeFiles(tx contractsorm.Query, agentID uint64, files []domain.File) error {
	for _, f := range files {
		if err := tx.Create(&fileRecord{AgentID: agentID, Path: f.Path, Content: f.Content}); err != nil {
			return err
		}
	}
	return nil
}

func (Agents) ByID(ctx context.Context, id uint64) (domain.Agent, bool, error) {
	var recs []agentRecord
	if err := query(ctx).Where("id", id).Find(&recs); err != nil {
		return domain.Agent{}, false, err
	}
	if len(recs) == 0 {
		return domain.Agent{}, false, nil
	}
	a := recs[0].toDomain()
	var files []fileRecord
	if err := query(ctx).Where("agent_id", id).OrderBy("path").Find(&files); err != nil {
		return domain.Agent{}, false, err
	}
	a.Files = make([]domain.File, len(files))
	for i, f := range files {
		a.Files[i] = domain.File{Path: f.Path, Content: f.Content}
	}
	return a, true, nil
}

func (Agents) OfOwner(ctx context.Context, ownerID uint64, since *time.Time) ([]domain.Agent, error) {
	q := query(ctx).Where("owner_member_id", ownerID)
	if since != nil {
		q = q.Where("updated_at > ?", *since).OrderBy("updated_at")
	} else {
		q = q.WhereNull("deleted_at").OrderBy("name").OrderBy("id")
	}
	var recs []agentRecord
	if err := q.Find(&recs); err != nil {
		return nil, err
	}
	out := make([]domain.Agent, len(recs))
	for i, r := range recs {
		out[i] = r.toDomain()
	}
	return out, nil
}

// LogEvents writes the agents context's events to the log.
type LogEvents struct{}

func (LogEvents) Handle(ctx context.Context, event any) {
	facades.Log().WithContext(ctx).Infof("%T %+v", event, event)
}

var _ app.Agents = Agents{}
