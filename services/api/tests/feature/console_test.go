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

// operatorSuite is featureSuite with operators: made straight in the
// database, signed in, and cleaned up after each test.
type operatorSuite struct {
	featureSuite
	operatorEmails []string
}

type ConsoleTestSuite struct {
	operatorSuite
}

func TestConsoleTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(ConsoleTestSuite))
}

// SetupTest starts each test with fresh rate limits: they are counted per
// client in the cache, and every test here is the same client.
func (s *operatorSuite) SetupTest() {
	s.Require().True(facades.Cache().Flush())
}

// tooMany reports a refusal by the rate limiter, which says when to try
// again.
func tooMany(res response) bool {
	return !res.IsSuccessful() && res.Headers().Get("Retry-After") != ""
}

func (s *operatorSuite) TearDownTest() {
	for _, e := range s.operatorEmails {
		var ids []uint64
		s.NoError(facades.Orm().Query().Table("operators").Where("email", e).Pluck("id", &ids))
		for _, id := range ids {
			_, err := facades.DB().Table("reports").Where("handled_by", id).Delete()
			s.NoError(err)
			_, err = facades.DB().Table("sanctions").Where("by_operator_id", id).Delete()
			s.NoError(err)
		}
		_, err := facades.DB().Table("operators").Where("email", e).Delete()
		s.NoError(err)
	}
	s.operatorEmails = nil
	s.featureSuite.TearDownTest()
}

// operator makes a confirmed operator straight in the database, as
// operator:create and operator:confirm would, and returns its TOTP secret.
func (s *operatorSuite) operator(password string) (email, secret string) {
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
func (s *operatorSuite) signIn(email, password, secret string) string {
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

// audit reads the audit log as an operator.
func (s *operatorSuite) audit(token, query string) []map[string]any {
	res := s.get(token, "/api/console/audit?"+query)
	res.AssertOk()
	var out []map[string]any
	for _, e := range s.jsonOf(res)["entries"].([]any) {
		out = append(out, e.(map[string]any))
	}
	return out
}

func (s *ConsoleTestSuite) TestAuditLog() {
	email, secret := s.operator("a long enough secret")
	op := s.signIn(email, "a long enough secret", secret)

	_, ada := s.register()
	guildID := s.foundGuild(ada, "Audited Colony")
	_, bram := s.register()
	s.join(ada, guildID, bram)
	bramID := uint64(s.jsonOf(s.get(bram, "/api/me"))["member"].(map[string]any)["id"].(float64))
	adaID := uint64(s.jsonOf(s.get(ada, "/api/me"))["member"].(map[string]any)["id"].(float64))

	// Removing a member from a guild is in the log, with who did it.
	s.send("DELETE", ada, fmt.Sprintf("/api/guilds/%d/members/%d", guildID, bramID), "").AssertNoContent()
	removed := s.audit(op, fmt.Sprintf("action=guild.member_removed&target_kind=member&target_id=%d", bramID))
	s.Require().Len(removed, 1)
	s.Equal(float64(adaID), removed[0]["actor_id"])
	s.Equal("member", removed[0]["actor_kind"])
	s.Equal(float64(guildID), removed[0]["meta"].(map[string]any)["guild_id"])
	s.NotEmpty(removed[0]["ip"])

	// Personal tokens made and revoked, and archiving.
	res := s.post(ada, "/api/tokens", `{"name":"audited"}`)
	res.AssertCreated()
	tokenID := uint64(s.jsonOf(res)["personal_token"].(map[string]any)["id"].(float64))
	s.send("DELETE", ada, fmt.Sprintf("/api/tokens/%d", tokenID), "").AssertNoContent()
	s.post(ada, fmt.Sprintf("/api/guilds/%d/archive", guildID), "").AssertOk()
	actions := map[string]bool{}
	for _, e := range s.audit(op, fmt.Sprintf("actor_kind=member&actor_id=%d", adaID)) {
		actions[e["action"].(string)] = true
	}
	for _, want := range []string{"guild.member_removed", "personal_token.created", "personal_token.revoked", "guild.archived"} {
		s.True(actions[want], want)
	}

	// The operator's own sign-in too; and members cannot read the log.
	s.NotEmpty(s.audit(op, "actor_kind=operator&action=operator.signed_in"))
	s.get(ada, "/api/console/audit").AssertUnauthorized()
	s.get(op, "/api/console/audit?from=yesterday").AssertUnprocessableEntity()
}

func (s *ConsoleTestSuite) TestMembersAndGuilds() {
	email, secret := s.operator("a long enough secret")
	op := s.signIn(email, "a long enough secret", secret)
	adaEmail, ada := s.register()
	adaID := uint64(s.jsonOf(s.get(ada, "/api/me"))["member"].(map[string]any)["id"].(float64))
	guildID := s.foundGuild(ada, "Directory Colony")
	s.post(ada, fmt.Sprintf("/api/guilds/%d/boards", guildID), `{"name":"One"}`).AssertCreated()

	// Search by email, with the guild count.
	res := s.get(op, "/api/console/members?q="+adaEmail)
	res.AssertOk()
	members := s.jsonOf(res)["members"].([]any)
	s.Require().Len(members, 1)
	s.Equal(float64(1), members[0].(map[string]any)["guild_count"])
	s.Nil(members[0].(map[string]any)["sanction"])

	// A suspension shows on the member's row and record.
	until := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	s.post(op, "/api/console/sanctions", fmt.Sprintf(`{"target":"member:%d","kind":"suspension","reason":"Cooling off.","until":%q}`, adaID, until)).AssertCreated()
	members = s.jsonOf(s.get(op, "/api/console/members?q="+adaEmail))["members"].([]any)
	s.Equal("suspension", members[0].(map[string]any)["sanction"].(map[string]any)["kind"])
	record := s.jsonOf(s.get(op, fmt.Sprintf("/api/console/members/%d", adaID)))
	s.Len(record["guilds"].([]any), 1)
	s.Len(record["sanctions"].([]any), 1)
	s.NotEmpty(record["audit"])

	// Guilds: search by name, and the record counts boards and members.
	gs := s.jsonOf(s.get(op, "/api/console/guilds?q=directory%20colony"))["guilds"].([]any)
	s.Require().NotEmpty(gs)
	g := s.jsonOf(s.get(op, fmt.Sprintf("/api/console/guilds/%d", guildID)))
	s.Equal(float64(1), g["board_count"])
	s.Len(g["members"].([]any), 1)
	s.get(op, "/api/console/guilds/999999999").AssertNotFound()
	s.get(ada, "/api/console/members").AssertUnauthorized()
}
