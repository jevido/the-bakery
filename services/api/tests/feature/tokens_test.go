package feature

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/tests"
)

type TokensTestSuite struct{ featureSuite }

func TestTokensTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(TokensTestSuite))
}

func (s *TokensTestSuite) TearDownTest() {
	for _, email := range s.emails {
		var ids []uint64
		s.NoError(facades.Orm().Query().Table("members").Where("email", email).Pluck("id", &ids))
		for _, id := range ids {
			_, err := facades.DB().Table("personal_tokens").Where("member_id", id).Delete()
			s.NoError(err)
		}
	}
	s.featureSuite.TearDownTest()
}

func (s *TokensTestSuite) TestCreateUseRevoke() {
	email, session := s.register()

	res := s.post(session, "/api/tokens", `{"name":"laptop claude"}`)
	res.AssertCreated()
	got := s.jsonOf(res)
	secret := got["token"].(string)
	s.True(strings.HasPrefix(secret, "bky_"))
	s.Len(secret, 36)
	tokenID := int(got["personal_token"].(map[string]any)["id"].(float64))
	s.post(session, "/api/tokens", `{"name":""}`).AssertUnprocessableEntity()

	// The token signs in like the member.
	res = s.get(secret, "/api/me")
	res.AssertOk()
	s.Equal(email, s.jsonOf(res)["member"].(map[string]any)["email"])
	guildID := s.foundGuild(secret, "Token Colony")
	s.get(secret, fmt.Sprintf("/api/guilds/%d/boards", guildID)).AssertOk()

	// The list shows it with its last use, never the secret.
	list := s.jsonOf(s.get(session, "/api/tokens"))["tokens"].([]any)
	s.Require().Len(list, 1)
	s.Equal("laptop claude", list[0].(map[string]any)["name"])
	s.NotNil(list[0].(map[string]any)["last_used_at"])
	s.NotContains(fmt.Sprint(list), secret)

	// A token cannot manage tokens.
	s.post(secret, "/api/tokens", `{"name":"more"}`).AssertForbidden()
	s.get(secret, "/api/tokens").AssertForbidden()

	// Only the owner can revoke; for others it does not exist.
	_, other := s.register()
	s.send("DELETE", other, fmt.Sprintf("/api/tokens/%d", tokenID), "").AssertNotFound()

	s.send("DELETE", session, fmt.Sprintf("/api/tokens/%d", tokenID), "").AssertNoContent()
	s.get(secret, "/api/me").AssertUnauthorized()
	s.Empty(s.jsonOf(s.get(session, "/api/tokens"))["tokens"])
	s.get("bky_"+strings.Repeat("x", 32), "/api/me").AssertUnauthorized()
}
