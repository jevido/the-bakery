package feature

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/tests"
)

type HandoffTestSuite struct{ featureSuite }

func TestHandoffTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(HandoffTestSuite))
}

func (s *HandoffTestSuite) redeem(code string, web bool) response {
	req := s.Http(s.T()).WithHeader("Content-Type", "application/json")
	if web {
		req = req.WithHeader("X-Bakery-Web", "1")
	}
	res, err := req.Post("/api/web/handoff/redeem", strings.NewReader(fmt.Sprintf(`{"code":%q}`, code)))
	s.Require().NoError(err)
	return res
}

func (s *HandoffTestSuite) code(token string) string {
	res := s.post(token, "/api/web/handoff", "")
	res.AssertCreated()
	return s.jsonOf(res)["code"].(string)
}

func (s *HandoffTestSuite) TestDesktopToWebsite() {
	email, token := s.register()
	s.post("", "/api/web/handoff", "").AssertUnauthorized()

	code := s.code(token)
	s.redeem(code, false).AssertForbidden()

	res := s.redeem(code, true)
	res.AssertOk()
	s.Equal(email, s.jsonOf(res)["member"].(map[string]any)["email"])
	cookie := res.Cookie("bakery_session")
	s.Require().NotNil(cookie)

	me, err := s.Http(s.T()).WithCookie(cookie).Get("/api/me")
	s.Require().NoError(err)
	me.AssertOk()

	// A code works once.
	s.redeem(code, true).AssertUnauthorized()
	s.redeem("not-a-code", true).AssertUnauthorized()
}

func (s *HandoffTestSuite) TestExpiredCode() {
	_, token := s.register()
	code := s.code(token)
	_, err := facades.DB().Table("web_handoffs").Where("used_at IS NULL").Where("created_at > ?", time.Now().Add(-time.Minute)).
		Update("expires_at", time.Now().Add(-time.Second))
	s.Require().NoError(err)
	s.redeem(code, true).AssertUnauthorized()
}
