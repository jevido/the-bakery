package feature

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/tests"
)

type LimitsTestSuite struct {
	operatorSuite
}

func TestLimitsTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(LimitsTestSuite))
}

// signals reads the console's Signals screen.
func (s *LimitsTestSuite) signals(op string) map[string][]map[string]any {
	res := s.get(op, "/api/console/signals")
	res.AssertOk()
	out := map[string][]map[string]any{}
	for key, v := range s.jsonOf(res) {
		rows, ok := v.([]any)
		if !ok {
			continue
		}
		for _, r := range rows {
			out[key] = append(out[key], r.(map[string]any))
		}
	}
	return out
}

func (s *LimitsTestSuite) refusedAsTooMany(res response, limit string) {
	res.AssertTooManyRequests()
	s.NotEmpty(res.Headers().Get("Retry-After"))
	body, _ := res.Json()
	s.Equal("rate_limited", body["code"])
	s.Equal(limit, body["limit"])
}

func (s *LimitsTestSuite) TestSixthRegistrationInAnHourIsRefused() {
	email, secret := s.operator("a long enough secret")
	op := s.signIn(email, "a long enough secret", secret)
	for range 5 {
		s.register()
	}
	res := s.post("", "/api/web/register", fmt.Sprintf(`{"email":%q,"display_name":"Tester","password":"firstlanding"}`, s.newEmail()))
	s.refusedAsTooMany(res, "register")

	// The IP is among the top sign-ups (the dev database may hold other
	// IPs), and its refusal is a signal too.
	sig := s.signals(op)
	var top map[string]any
	for _, r := range sig["sign_ups_by_ip"] {
		if r["ip"] == "192.0.2.1" {
			top = r
		}
	}
	s.Require().NotNil(top, "the IP is not among the sign-ups")
	s.Equal(float64(5), top["count"])
	found := false
	for _, r := range sig["limit_hits_by_ip"] {
		if r["ip"] == "192.0.2.1" {
			found = true
			s.Contains(r["limits"], "register")
		}
	}
	s.True(found, "the refused IP is not among the limit hits")
}

func (s *LimitsTestSuite) TestSixthGuildInADayIsRefused() {
	email, secret := s.operator("a long enough secret")
	op := s.signIn(email, "a long enough secret", secret)
	_, ada := s.register()
	adaID := s.jsonOf(s.get(ada, "/api/me"))["member"].(map[string]any)["id"]
	for i := range 5 {
		s.foundGuild(ada, fmt.Sprintf("Limit Colony %d", i))
	}
	s.refusedAsTooMany(s.post(ada, "/api/guilds", `{"name":"One too many"}`), "guild-create")

	sig := s.signals(op)
	var founded, hits map[string]any
	for _, r := range sig["guilds_founded_by"] {
		if r["member_id"] == adaID {
			founded = r
		}
	}
	for _, r := range sig["limit_hits_by_member"] {
		if r["member_id"] == adaID {
			hits = r
		}
	}
	s.Require().NotNil(founded, "the member is not among the guild founders")
	s.Equal(float64(5), founded["count"])
	s.Equal("Tester", founded["name"])
	s.Require().NotNil(hits, "the member is not among the limit hits")
	s.Contains(hits["limits"], "guild-create")
}

func (s *LimitsTestSuite) TestSignInIsLimitedPerIP() {
	email, _ := s.register()
	for range 10 {
		s.post("", "/api/login", fmt.Sprintf(`{"email":%q,"password":"wrong wrong"}`, email)).AssertUnauthorized()
	}
	s.refusedAsTooMany(s.post("", "/api/login", fmt.Sprintf(`{"email":%q,"password":"firstlanding"}`, email)), "login")
}

func (s *LimitsTestSuite) TestWritesAreLimitedPerMember() {
	_, ada := s.register()
	for range 120 {
		s.send("PATCH", ada, "/api/guilds/999999999", `{"name":"x"}`).AssertForbidden()
	}
	s.refusedAsTooMany(s.send("PATCH", ada, "/api/guilds/999999999", `{"name":"x"}`), "writes")
	// Reading goes on, and another member is not held back.
	s.get(ada, "/api/guilds").AssertOk()
	_, bram := s.register()
	s.send("PATCH", bram, "/api/guilds/999999999", `{"name":"x"}`).AssertForbidden()
}

func (s *LimitsTestSuite) TestMCPIsLimitedPerToken() {
	_, ada := s.register()
	res := s.post(ada, "/api/tokens", `{"name":"busy claude"}`)
	res.AssertCreated()
	personal := s.jsonOf(res)["token"].(string)
	ping := `{"jsonrpc":"2.0","id":1,"method":"ping"}`
	call := func() response {
		r, err := s.Http(s.T()).WithToken(personal).
			WithHeaders(map[string]string{"Content-Type": "application/json", "Accept": "application/json, text/event-stream"}).
			Post("/mcp", strings.NewReader(ping))
		s.Require().NoError(err)
		return r
	}
	for range 300 {
		call().AssertOk()
	}
	s.refusedAsTooMany(call(), "mcp")
}
