package infra

import (
	"context"
	"strings"
	"time"
)

// Listed is a guild as the platform's directory lists it.
type Listed struct {
	ID        uint64
	Name      string
	Archived  bool
	FoundedAt time.Time
	MemberIDs []uint64
}

// Directory searches every guild, for the operator console. Sanctions do
// not hide a guild here.
type Directory struct{}

// Find lists guilds whose name contains q (any case), newest first.
func (d Directory) Find(ctx context.Context, q string, limit int) ([]Listed, error) {
	qr := query(ctx).Model(&guildRecord{})
	if q = strings.TrimSpace(q); q != "" {
		qr = qr.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(q)+"%")
	}
	var recs []guildRecord
	if err := qr.OrderByDesc("id").Limit(limit).Find(&recs); err != nil {
		return nil, err
	}
	return d.withMembers(ctx, recs)
}

// ByID is one guild, found or not.
func (d Directory) ByID(ctx context.Context, id uint64) (Listed, bool, error) {
	var recs []guildRecord
	if err := query(ctx).Where("id", id).Find(&recs); err != nil || len(recs) == 0 {
		return Listed{}, false, err
	}
	ls, err := d.withMembers(ctx, recs)
	if err != nil {
		return Listed{}, false, err
	}
	return ls[0], true, nil
}

// OfMember lists every guild the member is in, archived ones too.
func (d Directory) OfMember(ctx context.Context, memberID uint64) ([]Listed, error) {
	var recs []guildRecord
	err := query(ctx).Model(&guildRecord{}).Select("guilds.*").
		Join("JOIN guild_memberships ON guild_memberships.guild_id = guilds.id").
		Where("guild_memberships.member_id", memberID).OrderBy("guilds.name").Find(&recs)
	if err != nil {
		return nil, err
	}
	return d.withMembers(ctx, recs)
}

// GuildCounts maps each member to the number of guilds they are in.
func (Directory) GuildCounts(ctx context.Context, memberIDs []uint64) (map[uint64]int, error) {
	out := map[uint64]int{}
	if len(memberIDs) == 0 {
		return out, nil
	}
	in := make([]any, len(memberIDs))
	for i, id := range memberIDs {
		in[i] = id
	}
	var mems []membershipRecord
	if err := query(ctx).WhereIn("member_id", in).Find(&mems); err != nil {
		return nil, err
	}
	for _, m := range mems {
		out[m.MemberID]++
	}
	return out, nil
}

func (Directory) withMembers(ctx context.Context, recs []guildRecord) ([]Listed, error) {
	out := make([]Listed, len(recs))
	if len(recs) == 0 {
		return out, nil
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
	for i, r := range recs {
		var founded time.Time
		if r.CreatedAt != nil {
			founded = r.CreatedAt.StdTime()
		}
		out[i] = Listed{ID: r.ID, Name: r.Name, Archived: r.ArchivedAt != nil, FoundedAt: founded, MemberIDs: byGuild[r.ID]}
	}
	return out, nil
}
