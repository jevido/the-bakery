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

type IdentityTestSuite struct {
	suite.Suite
	tests.TestCase
	email string
}

func TestIdentityTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(IdentityTestSuite))
}

func (s *IdentityTestSuite) SetupTest() {
	// A fresh address per run, so the test can use the dev database.
	s.email = fmt.Sprintf("test-%d@bakery.test", time.Now().UnixNano())
}

func (s *IdentityTestSuite) TearDownTest() {
	_, err := facades.DB().Table("members").Where("email", s.email).Delete()
	s.NoError(err)
}

func (s *IdentityTestSuite) TestRegisterLoginMe() {
	body := fmt.Sprintf(`{"email":%q,"display_name":"Ada","password":"firstlanding"}`, strings.ToUpper(s.email))
	res, err := s.Http(s.T()).WithHeader("Content-Type", "application/json").Post("/api/register", strings.NewReader(body))
	s.Require().NoError(err)
	res.AssertCreated()
	reg, err := res.Json()
	s.Require().NoError(err)
	s.NotEmpty(reg["token"])
	s.Equal(s.email, reg["member"].(map[string]any)["email"], "email stored lowercased")

	res, err = s.Http(s.T()).WithHeader("Content-Type", "application/json").Post("/api/register", strings.NewReader(body))
	s.Require().NoError(err)
	res.AssertUnprocessableEntity()

	login := fmt.Sprintf(`{"email":%q,"password":"firstlanding"}`, s.email)
	res, err = s.Http(s.T()).WithHeader("Content-Type", "application/json").Post("/api/login", strings.NewReader(login))
	s.Require().NoError(err)
	res.AssertOk()
	got, err := res.Json()
	s.Require().NoError(err)
	token, _ := got["token"].(string)
	s.Require().NotEmpty(token)

	res, err = s.Http(s.T()).WithToken(token).Get("/api/me")
	s.Require().NoError(err)
	res.AssertOk()
	me, err := res.Json()
	s.Require().NoError(err)
	s.Equal("Ada", me["member"].(map[string]any)["display_name"])
}

func (s *IdentityTestSuite) TestBadLoginIsGeneric() {
	for _, body := range []string{
		fmt.Sprintf(`{"email":%q,"password":"wrong-password"}`, s.email),
		`{"email":"nobody@bakery.test","password":"firstlanding"}`,
	} {
		res, err := s.Http(s.T()).WithHeader("Content-Type", "application/json").Post("/api/login", strings.NewReader(body))
		s.Require().NoError(err)
		res.AssertUnauthorized()
		got, err := res.Json()
		s.Require().NoError(err)
		s.Equal("email or password is incorrect", got["error"])
	}
}

func (s *IdentityTestSuite) TestMeWithoutToken() {
	res, err := s.Http(s.T()).Get("/api/me")
	s.Require().NoError(err)
	res.AssertUnauthorized()

	res, err = s.Http(s.T()).WithToken("not-a-token").Get("/api/me")
	s.Require().NoError(err)
	res.AssertUnauthorized()
}
