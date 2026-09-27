package feature

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/tests"
)

func init() {
	guildCleanup = append(guildCleanup, func(guildID uint64) error {
		_, err := facades.DB().Table("guild_invites").Where("guild_id", guildID).Delete()
		return err
	})
}

type InvitesTestSuite struct{ featureSuite }

func TestInvitesTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(InvitesTestSuite))
}

func (s *InvitesTestSuite) invite(token string, guildID uint64, body string) map[string]any {
	res := s.post(token, fmt.Sprintf("/api/guilds/%d/invites", guildID), body)
	res.AssertCreated()
	return s.jsonOf(res)["invite"].(map[string]any)
}

func (s *InvitesTestSuite) TestInviteAndJoin() {
	_, founder := s.register()
	guildID := s.foundGuild(founder, "Test Colony")
	inv := s.invite(founder, guildID, `{"max_uses":2}`)
	code := inv["code"].(string)
	s.Len(code, 10)
	s.Equal("active", inv["state"])

	// Anyone can look it up.
	res := s.get("", "/api/invites/"+code)
	res.AssertOk()
	info := s.jsonOf(res)["invite"].(map[string]any)
	s.Equal("Test Colony", info["guild_name"])
	s.Equal(float64(1), info["member_count"])
	s.Equal(true, info["valid"])
	s.get("", "/api/invites/nope").AssertNotFound()

	// Joining needs an account.
	s.post("", "/api/invites/"+code+"/accept", "").AssertUnauthorized()
	_, joiner := s.register()
	res = s.post(joiner, "/api/invites/"+code+"/accept", "")
	res.AssertOk()
	s.Equal("Test Colony", s.jsonOf(res)["guild"].(map[string]any)["name"])
	s.Len(s.jsonOf(s.get(joiner, "/api/guilds"))["guilds"], 1)

	// Accepting again is fine and does not use the invite up.
	s.post(joiner, "/api/invites/"+code+"/accept", "").AssertOk()
	invites := s.jsonOf(s.get(founder, fmt.Sprintf("/api/guilds/%d/invites", guildID)))["invites"].([]any)
	s.Equal(float64(1), invites[0].(map[string]any)["uses"])

	// The second use is the last.
	_, second := s.register()
	s.post(second, "/api/invites/"+code+"/accept", "").AssertOk()
	_, third := s.register()
	s.post(third, "/api/invites/"+code+"/accept", "").AssertStatus(410)
	s.Equal(false, s.jsonOf(s.get("", "/api/invites/"+code))["invite"].(map[string]any)["valid"])

	// Only members see and manage a guild's invites.
	s.get(third, fmt.Sprintf("/api/guilds/%d/invites", guildID)).AssertForbidden()
}

func (s *InvitesTestSuite) TestRevokedAndExpired() {
	_, founder := s.register()
	guildID := s.foundGuild(founder, "Test Colony")

	revoked := s.invite(founder, guildID, `{}`)
	s.send("DELETE", founder, fmt.Sprintf("/api/guilds/%d/invites/%d", guildID, int(revoked["id"].(float64))), "").AssertNoContent()
	_, joiner := s.register()
	res := s.post(joiner, "/api/invites/"+revoked["code"].(string)+"/accept", "")
	res.AssertStatus(410)
	s.Equal("this invite was withdrawn", s.jsonOf(res)["error"])

	expired := s.invite(founder, guildID, `{"expires_in_hours":1}`)
	_, err := facades.DB().Table("guild_invites").Where("id", int(expired["id"].(float64))).Update("expires_at", time.Now().Add(-time.Minute))
	s.Require().NoError(err)
	s.post(joiner, "/api/invites/"+expired["code"].(string)+"/accept", "").AssertStatus(410)

	s.post(founder, fmt.Sprintf("/api/guilds/%d/invites", guildID), `{"expires_in_hours":0}`).AssertUnprocessableEntity()
	s.post(founder, fmt.Sprintf("/api/guilds/%d/invites", guildID), `{"max_uses":0}`).AssertUnprocessableEntity()

	// An archived guild takes no invites and no new members.
	open := s.invite(founder, guildID, `{}`)
	s.post(founder, fmt.Sprintf("/api/guilds/%d/archive", guildID), "").AssertOk()
	s.post(founder, fmt.Sprintf("/api/guilds/%d/invites", guildID), `{}`).AssertConflict()
	s.post(joiner, "/api/invites/"+open["code"].(string)+"/accept", "").AssertConflict()
}
