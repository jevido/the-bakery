package feature

import (
	"fmt"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/tests"
)

type ConsoleTestSuite struct {
	featureSuite
	operatorEmails []string
}

func TestConsoleTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(ConsoleTestSuite))
}

// SetupTest starts each test with fresh rate limits: they are counted per
// client in the cache, and every test here is the same client.
func (s *ConsoleTestSuite) SetupTest() {
	s.Require().True(facades.Cache().Flush())
}

// tooMany reports a refusal by the rate limiter, which says when to try
// again.
func tooMany(res response) bool {
	return !res.IsSuccessful() && res.Headers().Get("Retry-After") != ""
}

func (s *ConsoleTestSuite) TearDownTest() {
	for _, e := range s.operatorEmails {
		_, err := facades.DB().Table("operators").Where("email", e).Delete()
		s.NoError(err)
	}
	s.operatorEmails = nil
	s.featureSuite.TearDownTest()
}

// operator makes a confirmed operator straight in the database, as
// operator:create and operator:confirm would, and returns its TOTP secret.
func (s *ConsoleTestSuite) operator(password string) (email, secret string) {
	email = fmt.Sprintf("op-%d@bakery.test", time.Now().UnixNano())
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "test", AccountName: email})
	s.Require().NoError(err)
	hash, err := facades.Hash().Make(password)
	s.Require().NoError(err)
	enc, err := facades.Crypt().EncryptString(key.Secret())
	s.Require().NoError(err)
	_, err = facades.DB().Table("operators").Insert(map[string]any{
		"email": email, "password_hash": hash, "totp_secret": enc, "totp_confirmed_at": time.Now(),
	})
	s.Require().NoError(err)
	s.operatorEmails = append(s.operatorEmails, email)
	return email, key.Secret()
}

// signIn goes through both steps and returns the operator's token.
func (s *ConsoleTestSuite) signIn(email, password, secret string) string {
	res := s.post("", "/api/console/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, password))
	res.AssertOk()
	challenge := s.jsonOf(res)["challenge"].(string)
	code, err := totp.GenerateCode(secret, time.Now())
	s.Require().NoError(err)
	res = s.post("", "/api/console/totp/verify", fmt.Sprintf(`{"challenge":%q,"code":%q}`, challenge, code))
	res.AssertOk()
	return s.jsonOf(res)["token"].(string)
}

func (s *ConsoleTestSuite) TestSignInAndGuards() {
	email, secret := s.operator("a long enough secret")
	token := s.signIn(email, "a long enough secret", secret)

	res := s.get(token, "/api/console/me")
	res.AssertOk()
	s.Equal(email, s.jsonOf(res)["operator"].(map[string]any)["email"])

	// An operator's token is no member's, and the other way round.
	s.get(token, "/api/me").AssertUnauthorized()
	s.get(token, "/api/guilds").AssertUnauthorized()
	_, member := s.register()
	s.get(member, "/api/console/me").AssertUnauthorized()
	s.get("", "/api/console/me").AssertUnauthorized()
}

func (s *ConsoleTestSuite) TestMembersCannotSignIn() {
	memberEmail, _ := s.register()
	s.post("", "/api/console/login", fmt.Sprintf(`{"email":%q,"password":"password1"}`, memberEmail)).AssertUnauthorized()
}

func (s *ConsoleTestSuite) TestWrongPasswordOrCode() {
	email, secret := s.operator("a long enough secret")
	s.post("", "/api/console/login", fmt.Sprintf(`{"email":%q,"password":"not the password"}`, email)).AssertUnauthorized()

	res := s.post("", "/api/console/login", fmt.Sprintf(`{"email":%q,"password":"a long enough secret"}`, email))
	res.AssertOk()
	challenge := s.jsonOf(res)["challenge"].(string)
	s.post("", "/api/console/totp/verify", fmt.Sprintf(`{"challenge":%q,"code":"000000"}`, challenge)).AssertUnauthorized()
	// A challenge is not a token.
	s.get(challenge, "/api/console/me").AssertUnauthorized()
	_ = secret
}

func (s *ConsoleTestSuite) TestSignInIsRateLimited() {
	// Five tries a minute per client for each step; the sixth is refused.
	seen429 := false
	for range 6 {
		res := s.post("", "/api/console/login", `{"email":"nobody@bakery.test","password":"wrong wrong wrong"}`)
		if tooMany(res) {
			seen429 = true
		}
	}
	s.True(seen429, "the sixth sign-in in a minute is refused")
}
