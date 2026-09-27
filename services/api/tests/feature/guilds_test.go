package feature

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/tests"
)

type GuildsTestSuite struct{ featureSuite }

func TestGuildsTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(GuildsTestSuite))
}

func (s *GuildsTestSuite) TestFoundListAndAddMember() {
	_, founder := s.register()
	otherEmail, other := s.register()

	id := s.foundGuild(founder, "Test Colony")
	guilds := s.jsonOf(s.get(founder, "/api/guilds"))["guilds"].([]any)
	s.Len(guilds, 1)
	s.Equal("Test Colony", guilds[0].(map[string]any)["name"])

	uri := fmt.Sprintf("/api/guilds/%d/members", id)
	s.post(founder, uri, fmt.Sprintf(`{"email":%q}`, otherEmail)).AssertCreated()
	s.post(founder, uri, fmt.Sprintf(`{"email":%q}`, otherEmail)).AssertConflict()
	s.post(founder, uri, `{"email":"nobody@bakery.test"}`).AssertUnprocessableEntity()

	guilds = s.jsonOf(s.get(other, "/api/guilds"))["guilds"].([]any)
	s.Len(guilds, 1, "added member sees the guild")
}

func (s *GuildsTestSuite) TestFoundRejectsBadName() {
	_, token := s.register()
	s.post(token, "/api/guilds", `{"name":"   "}`).AssertUnprocessableEntity()
}

func (s *GuildsTestSuite) TestNonMember() {
	email, token := s.register()
	res := s.get(token, "/api/guilds")
	res.AssertOk()
	s.Empty(s.jsonOf(res)["guilds"])

	uri := fmt.Sprintf("/api/guilds/%d/members", s.seededGuildID())
	s.post(token, uri, fmt.Sprintf(`{"email":%q}`, email)).AssertForbidden()
	s.post(token, "/api/guilds/999999999/members", fmt.Sprintf(`{"email":%q}`, email)).AssertForbidden()
}

func (s *GuildsTestSuite) TestRequiresToken() {
	s.get("", "/api/guilds").AssertUnauthorized()
	s.post("", "/api/guilds", `{"name":"Nope"}`).AssertUnauthorized()
}

func (s *GuildsTestSuite) TestSeededMemberSeesFirstColony() {
	s.seededGuildID()
	token := s.login("ada@bakery.test", "password")
	names := []string{}
	for _, g := range s.jsonOf(s.get(token, "/api/guilds"))["guilds"].([]any) {
		names = append(names, g.(map[string]any)["name"].(string))
	}
	s.Contains(names, "First Colony")
}
