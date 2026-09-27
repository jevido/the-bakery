package feature

import (
	"fmt"
	nethttp "net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/tests"
)

type GuildAdminTestSuite struct{ featureSuite }

func TestGuildAdminTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(GuildAdminTestSuite))
}

// webLogin signs in through the website endpoint and returns the session
// cookie.
func (s *GuildAdminTestSuite) webLogin(email string) *nethttp.Cookie {
	res, err := s.Http(s.T()).WithHeader("Content-Type", "application/json").WithHeader("X-Bakery-Web", "1").
		Post("/api/web/login", strings.NewReader(fmt.Sprintf(`{"email":%q,"password":"firstlanding"}`, email)))
	s.Require().NoError(err)
	res.AssertOk()
	got := s.jsonOf(res)
	s.Nil(got["token"], "the web login keeps the token out of the body")
	c := res.Cookie("bakery_session")
	s.Require().NotNil(c)
	s.True(c.HttpOnly)
	return c
}

func (s *GuildAdminTestSuite) withCookie(c *nethttp.Cookie, web bool, method, uri, body string) response {
	req := s.Http(s.T()).WithHeader("Content-Type", "application/json").WithCookie(c)
	if web {
		req = req.WithHeader("X-Bakery-Web", "1")
	}
	var (
		res response
		err error
	)
	switch method {
	case "GET":
		res, err = req.Get(uri)
	case "POST":
		res, err = req.Post(uri, strings.NewReader(body))
	case "PATCH":
		res, err = req.Patch(uri, strings.NewReader(body))
	case "DELETE":
		res, err = req.Delete(uri, strings.NewReader(body))
	}
	s.Require().NoError(err)
	return res
}

func (s *GuildAdminTestSuite) TestWebSession() {
	email, _ := s.register()
	c := s.webLogin(email)

	res := s.withCookie(c, false, "GET", "/api/me", "")
	res.AssertOk()
	s.Equal(email, s.jsonOf(res)["member"].(map[string]any)["email"])

	// A change with the cookie but without the header is refused.
	s.withCookie(c, false, "POST", "/api/guilds", `{"name":"Cookie Colony"}`).AssertForbidden()
	res = s.withCookie(c, true, "POST", "/api/guilds", `{"name":"Cookie Colony"}`)
	res.AssertCreated()
	s.guildIDs = append(s.guildIDs, uint64(s.jsonOf(res)["guild"].(map[string]any)["id"].(float64)))

	// The web endpoints themselves need the header too.
	res, err := s.Http(s.T()).WithHeader("Content-Type", "application/json").
		Post("/api/web/login", strings.NewReader(fmt.Sprintf(`{"email":%q,"password":"firstlanding"}`, email)))
	s.Require().NoError(err)
	res.AssertForbidden()

	res = s.withCookie(c, true, "POST", "/api/web/logout", "")
	res.AssertNoContent()
	s.Less(res.Cookie("bakery_session").MaxAge, 0, "logout expires the cookie")
}

func (s *GuildAdminTestSuite) TestRenameArchiveRestore() {
	_, token := s.register()
	id := s.foundGuild(token, "Test Colony")
	uri := fmt.Sprintf("/api/guilds/%d", id)

	res := s.send("PATCH", token, uri, `{"name":"Renamed Colony"}`)
	res.AssertOk()
	s.Equal("Renamed Colony", s.jsonOf(res)["guild"].(map[string]any)["name"])
	s.send("PATCH", token, uri, `{"name":""}`).AssertUnprocessableEntity()

	board := s.post(token, uri+"/boards", `{"name":"Stockpile"}`)
	board.AssertCreated()
	boardID := uint64(s.jsonOf(board)["board"].(map[string]any)["id"].(float64))

	s.post(token, uri+"/archive", "").AssertOk()
	s.post(token, uri+"/archive", "").AssertConflict()
	s.Empty(s.jsonOf(s.get(token, "/api/guilds"))["guilds"], "archived guilds are hidden")
	s.Len(s.jsonOf(s.get(token, "/api/guilds?archived=1"))["guilds"], 1)
	s.True(s.jsonOf(s.get(token, uri))["guild"].(map[string]any)["archived"].(bool))

	// Read-only: boards can be read but not changed.
	s.get(token, fmt.Sprintf("/api/boards/%d", boardID)).AssertOk()
	s.post(token, fmt.Sprintf("/api/boards/%d/tasks", boardID), `{"title":"x"}`).AssertConflict()
	s.post(token, uri+"/boards", `{"name":"Another"}`).AssertConflict()
	s.send("PATCH", token, uri, `{"name":"Nope"}`).AssertConflict()

	s.post(token, uri+"/restore", "").AssertOk()
	s.Len(s.jsonOf(s.get(token, "/api/guilds"))["guilds"], 1)
	s.post(token, fmt.Sprintf("/api/boards/%d/tasks", boardID), `{"title":"x"}`).AssertCreated()
}

func (s *GuildAdminTestSuite) TestMembersRemoveAndLeave() {
	_, founder := s.register()
	otherEmail, other := s.register()
	id := s.foundGuild(founder, "Test Colony")
	uri := fmt.Sprintf("/api/guilds/%d", id)
	s.post(founder, uri+"/members", fmt.Sprintf(`{"email":%q}`, otherEmail)).AssertCreated()

	members := s.jsonOf(s.get(founder, uri+"/members"))["members"].([]any)
	s.Require().Len(members, 2)
	s.Equal("Tester", members[0].(map[string]any)["display_name"])
	s.NotEmpty(members[0].(map[string]any)["joined_at"])
	otherID := uint64(members[1].(map[string]any)["id"].(float64))

	_, stranger := s.register()
	s.get(stranger, uri+"/members").AssertForbidden()
	s.send("DELETE", stranger, fmt.Sprintf("%s/members/%d", uri, otherID), "").AssertForbidden()

	s.send("DELETE", founder, fmt.Sprintf("%s/members/%d", uri, otherID), "").AssertNoContent()
	s.Empty(s.jsonOf(s.get(other, "/api/guilds"))["guilds"], "a removed member no longer sees the guild")
	s.send("DELETE", founder, fmt.Sprintf("%s/members/%d", uri, otherID), "").AssertNotFound()

	// The last member cannot leave.
	res := s.post(founder, uri+"/leave", "")
	res.AssertUnprocessableEntity()
	s.Contains(s.jsonOf(res)["error"], "archive the guild instead")

	s.post(founder, uri+"/members", fmt.Sprintf(`{"email":%q}`, otherEmail)).AssertCreated()
	s.post(other, uri+"/leave", "").AssertNoContent()
	s.Empty(s.jsonOf(s.get(other, "/api/guilds"))["guilds"])
}
