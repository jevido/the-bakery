package feature

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/tests"
)

type IdentityTestSuite struct{ featureSuite }

func TestIdentityTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(IdentityTestSuite))
}

func (s *IdentityTestSuite) TestRegisterLoginMe() {
	email := s.newEmail()
	body := fmt.Sprintf(`{"email":%q,"display_name":"Ada","password":"firstlanding"}`, strings.ToUpper(email))
	res := s.post("", "/api/register", body)
	res.AssertCreated()
	reg := s.jsonOf(res)
	s.NotEmpty(reg["token"])
	s.Equal(email, reg["member"].(map[string]any)["email"], "email stored lowercased")

	s.post("", "/api/register", body).AssertUnprocessableEntity()

	token := s.login(email, "firstlanding")
	res = s.get(token, "/api/me")
	res.AssertOk()
	s.Equal("Ada", s.jsonOf(res)["member"].(map[string]any)["display_name"])
}

func (s *IdentityTestSuite) TestBadLoginIsGeneric() {
	email, _ := s.register()
	for _, body := range []string{
		fmt.Sprintf(`{"email":%q,"password":"wrong-password"}`, email),
		`{"email":"nobody@bakery.test","password":"firstlanding"}`,
	} {
		res := s.post("", "/api/login", body)
		res.AssertUnauthorized()
		s.Equal("email or password is incorrect", s.jsonOf(res)["error"])
	}
}

func (s *IdentityTestSuite) TestMeWithoutToken() {
	s.get("", "/api/me").AssertUnauthorized()
	s.get("not-a-token", "/api/me").AssertUnauthorized()
}

func (s *IdentityTestSuite) TestPortraitSeedAndReroll() {
	_, ada := s.register()
	me := s.jsonOf(s.get(ada, "/api/me"))["member"].(map[string]any)
	seed := me["portrait_seed"].(string)
	s.NotEmpty(seed)

	res := s.post(ada, "/api/me/portrait", "")
	res.AssertOk()
	rerolled := s.jsonOf(res)["member"].(map[string]any)["portrait_seed"].(string)
	s.NotEmpty(rerolled)
	s.NotEqual(seed, rerolled)
	s.Equal(rerolled, s.jsonOf(s.get(ada, "/api/me"))["member"].(map[string]any)["portrait_seed"])

	// Guild member lists show it too.
	guildID := s.foundGuild(ada, "Portrait Colony")
	members := s.jsonOf(s.get(ada, fmt.Sprintf("/api/guilds/%d/members", guildID)))["members"].([]any)
	s.Equal(rerolled, members[0].(map[string]any)["portrait_seed"])
	s.post("", "/api/me/portrait", "").AssertUnauthorized()
}
