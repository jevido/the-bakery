package http

import (
	"errors"
	"strconv"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
	"github.com/jevido/the-bakery/services/api/contexts/moderation/domain"
)

type memberCardJSON struct {
	ID           uint64    `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	PortraitSeed string    `json:"portrait_seed"`
	JoinedAt     time.Time `json:"joined_at"`
}

type guildCardJSON struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Archived    bool      `json:"archived"`
	FoundedAt   time.Time `json:"founded_at"`
	MemberCount int       `json:"member_count"`
}

func memberCardToJSON(c app.MemberCard) memberCardJSON {
	return memberCardJSON{ID: c.ID, Email: c.Email, DisplayName: c.DisplayName, PortraitSeed: c.PortraitSeed, JoinedAt: c.JoinedAt}
}

func guildCardToJSON(c app.GuildCard) guildCardJSON {
	return guildCardJSON{ID: c.ID, Name: c.Name, Archived: c.Archived, FoundedAt: c.FoundedAt, MemberCount: len(c.MemberIDs)}
}

func activeToJSON(s *domain.Sanction) *sanctionJSON {
	if s == nil {
		return nil
	}
	j := sanctionToJSON(*s)
	return &j
}

func sanctionsToJSON(ss []domain.Sanction) []sanctionJSON {
	out := make([]sanctionJSON, len(ss))
	for i, s := range ss {
		out[i] = sanctionToJSON(s)
	}
	return out
}

// ListMembers searches members: ?q= matches email or display name.
func (c *Controller) ListMembers(ctx contractshttp.Context) contractshttp.Response {
	ls, err := c.service.FindMembers(ctx.Context(), ctx.Request().Query("q"))
	if err != nil {
		return serverError(ctx, err)
	}
	type row struct {
		memberCardJSON
		GuildCount int           `json:"guild_count"`
		Sanction   *sanctionJSON `json:"sanction"`
	}
	out := make([]row, len(ls))
	for i, l := range ls {
		out[i] = row{memberCardToJSON(l.MemberCard), l.GuildCount, activeToJSON(l.Active)}
	}
	return ctx.Response().Success().Json(contractshttp.Json{"members": out})
}

// ShowMember is one member's record.
func (c *Controller) ShowMember(ctx contractshttp.Context) contractshttp.Response {
	r, err := c.service.Member(ctx.Context(), uintString(ctx.Request().Route("member")))
	if err != nil {
		return directoryFailure(ctx, err)
	}
	guilds := make([]guildCardJSON, len(r.Guilds))
	for i, g := range r.Guilds {
		guilds[i] = guildCardToJSON(g)
	}
	names, err := c.namesOf(ctx, r.Audit, r.Reports)
	if err != nil {
		return serverError(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{
		"member": memberCardToJSON(r.MemberCard), "guilds": guilds, "sanctions": sanctionsToJSON(r.Sanctions),
		"reports": reportsToJSON(r.Reports), "audit": auditToJSON(r.Audit), "names": names,
	})
}

// ListGuilds searches guilds: ?q= matches the name.
func (c *Controller) ListGuilds(ctx contractshttp.Context) contractshttp.Response {
	ls, err := c.service.FindGuilds(ctx.Context(), ctx.Request().Query("q"))
	if err != nil {
		return serverError(ctx, err)
	}
	type row struct {
		guildCardJSON
		Sanction *sanctionJSON `json:"sanction"`
	}
	out := make([]row, len(ls))
	for i, l := range ls {
		out[i] = row{guildCardToJSON(l.GuildCard), activeToJSON(l.Active)}
	}
	return ctx.Response().Success().Json(contractshttp.Json{"guilds": out})
}

// ShowGuild is one guild's record.
func (c *Controller) ShowGuild(ctx contractshttp.Context) contractshttp.Response {
	r, err := c.service.Guild(ctx.Context(), uintString(ctx.Request().Route("guild")))
	if err != nil {
		return directoryFailure(ctx, err)
	}
	members := make([]memberCardJSON, len(r.Members))
	for i, m := range r.Members {
		members[i] = memberCardToJSON(m)
	}
	names, err := c.namesOf(ctx, r.Audit, r.Reports)
	if err != nil {
		return serverError(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{
		"guild": guildCardToJSON(r.GuildCard), "members": members, "board_count": r.BoardCount,
		"sanctions": sanctionsToJSON(r.Sanctions), "reports": reportsToJSON(r.Reports), "audit": auditToJSON(r.Audit), "names": names,
	})
}

// namesOf names every member and guild the entries and reports mention,
// keyed by id as a string (JSON object keys).
func (c *Controller) namesOf(ctx contractshttp.Context, entries []domain.AuditEntry, reports []domain.Report) (contractshttp.Json, error) {
	var members, guilds []uint64
	add := func(kind string, id uint64) {
		switch kind {
		case domain.TargetMember: // also domain.ActorMember
			members = append(members, id)
		case domain.TargetGuild:
			guilds = append(guilds, id)
		}
	}
	for _, e := range entries {
		add(e.ActorKind, e.ActorID)
		add(e.TargetKind, e.TargetID)
	}
	for _, r := range reports {
		add(domain.TargetMember, r.ByMemberID)
		add(r.TargetKind, r.TargetID)
	}
	m, g, err := c.service.Names(ctx.Context(), members, guilds)
	if err != nil {
		return nil, err
	}
	return contractshttp.Json{"members": keyed(m), "guilds": keyed(g)}, nil
}

func keyed(m map[uint64]string) map[string]string {
	out := make(map[string]string, len(m))
	for id, name := range m {
		out[strconv.FormatUint(id, 10)] = name
	}
	return out
}

func directoryFailure(ctx contractshttp.Context, err error) contractshttp.Response {
	if errors.Is(err, app.ErrMemberNotFound) || errors.Is(err, app.ErrGuildNotFound) {
		return ctx.Response().Json(contractshttp.StatusNotFound, contractshttp.Json{"error": err.Error()})
	}
	return serverError(ctx, err)
}
