package http

import (
	"strconv"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/moderation/app"
)

type countJSON struct {
	IP       string    `json:"ip,omitempty"`
	MemberID uint64    `json:"member_id,omitempty"`
	Name     string    `json:"name,omitempty"`
	Count    int       `json:"count"`
	Limits   []string  `json:"limits,omitempty"`
	Last     time.Time `json:"last"`
}

func countsToJSON(cs []app.Count, names map[uint64]string) []countJSON {
	out := make([]countJSON, len(cs))
	for i, c := range cs {
		out[i] = countJSON{IP: c.IP, MemberID: c.MemberID, Name: names[c.MemberID], Count: c.Count, Limits: c.Limits, Last: c.Last}
	}
	return out
}

// Signals summarises abuse signals of the last ?hours= (default 24): IPs
// by sign-ups, members and IPs by rate-limit refusals, members by guilds
// founded.
func (c *Controller) Signals(ctx contractshttp.Context) contractshttp.Response {
	hours, _ := strconv.Atoi(ctx.Request().Query("hours"))
	s, err := c.service.SignalsSince(ctx.Context(), hours)
	if err != nil {
		return serverError(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{
		"since":                s.Since,
		"sign_ups_by_ip":       countsToJSON(s.SignUpsByIP, nil),
		"limit_hits_by_member": countsToJSON(s.LimitHitsByMember, s.MemberNames),
		"limit_hits_by_ip":     countsToJSON(s.LimitHitsByIP, nil),
		"guilds_founded_by":    countsToJSON(s.GuildsFoundedBy, s.MemberNames),
	})
}
