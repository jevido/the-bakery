package feature

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/mcp"
	"github.com/jevido/the-bakery/services/api/tests"
)

type SanctionsTestSuite struct {
	operatorSuite
	api, mcpServer *httptest.Server
}

func TestSanctionsTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(SanctionsTestSuite))
}

func (s *SanctionsTestSuite) SetupSuite() {
	s.api = httptest.NewServer(facades.Route())
	s.mcpServer = httptest.NewServer(mcp.Handler())
}

func (s *SanctionsTestSuite) TearDownSuite() {
	s.api.Close()
	s.mcpServer.Close()
}

// sanctionsWait is how long a sanction or a lift may take to be seen:
// answers are cached a few seconds.
const sanctionsWait = 6 * time.Second

func (s *SanctionsTestSuite) eventually(check func() bool, what string) {
	deadline := time.Now().Add(sanctionsWait)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	s.FailNow(what)
}

func (s *SanctionsTestSuite) memberID(token string) uint64 {
	return uint64(s.jsonOf(s.get(token, "/api/me"))["member"].(map[string]any)["id"].(float64))
}

func (s *SanctionsTestSuite) TestBannedMemberIsKeptOut() {
	opEmail, secret := s.operator("a long enough secret")
	op := s.signIn(opEmail, "a long enough secret", secret)
	casEmail, cas := s.register()
	casID := s.memberID(cas)
	guildID := s.foundGuild(cas, "Cas Colony")
	res := s.post(cas, fmt.Sprintf("/api/guilds/%d/boards", guildID), `{"name":"Watched"}`)
	boardID := uint64(s.jsonOf(res)["board"].(map[string]any)["id"].(float64))
	res = s.post(cas, "/api/tokens", `{"name":"cas claude"}`)
	personal := s.jsonOf(res)["token"].(string)

	// An open live stream, to see it end.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/boards/%d/events", s.api.URL, boardID), nil)
	req.Header.Set("Authorization", "Bearer "+cas)
	stream, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	ended := make(chan struct{})
	go func() {
		sc := bufio.NewScanner(stream.Body)
		for sc.Scan() {
		}
		close(ended)
	}()

	res = s.post(op, "/api/console/sanctions", fmt.Sprintf(`{"target":"member:%d","kind":"ban","reason":"Spamming guilds."}`, casID))
	res.AssertCreated()
	sanctionID := uint64(s.jsonOf(res)["sanction"].(map[string]any)["id"].(float64))
	s.post(op, "/api/console/sanctions", fmt.Sprintf(`{"target":"member:%d","kind":"ban","reason":"again"}`, casID)).AssertConflict()

	// REST with the session token, and the refusal says why.
	var body map[string]any
	s.eventually(func() bool {
		r := s.get(cas, "/api/guilds")
		body, _ = r.Json()
		return body["code"] == "account_banned"
	}, "the banned member still gets in")
	s.Contains(body["error"], "Spamming guilds.")
	// The personal token, signing in again, and MCP.
	got, _ := s.get(personal, "/api/guilds").Json()
	s.Equal("account_banned", got["code"])
	got, _ = s.post("", "/api/login", fmt.Sprintf(`{"email":%q,"password":"firstlanding"}`, casEmail)).Json()
	s.Equal("account_banned", got["code"])
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	_, err = client.Connect(s.T().Context(), &sdk.StreamableClientTransport{
		Endpoint:   s.mcpServer.URL,
		HTTPClient: &http.Client{Transport: bearer{token: personal, next: http.DefaultTransport}},
	}, nil)
	s.Error(err, "MCP lets a banned member in")
	// The open stream was ended.
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		s.Fail("the live stream stayed open")
	}

	// The audit log has it, with the reason.
	entries := s.audit(op, fmt.Sprintf("action=sanction.ban&target_kind=member&target_id=%d", casID))
	s.Require().Len(entries, 1)
	s.Equal("Spamming guilds.", entries[0]["reason"])

	// Lifting lets them back in.
	s.post(op, fmt.Sprintf("/api/console/sanctions/%d/lift", sanctionID), `{"reason":"appeal"}`).AssertOk()
	s.eventually(func() bool { return s.get(cas, "/api/guilds").IsSuccessful() }, "the member is still out after the lift")
	s.NotEmpty(s.audit(op, fmt.Sprintf("action=sanction.lifted&target_kind=member&target_id=%d", casID)))
}

func (s *SanctionsTestSuite) TestSuspendedGuildIsHidden() {
	opEmail, secret := s.operator("a long enough secret")
	op := s.signIn(opEmail, "a long enough secret", secret)
	_, ada := s.register()
	guildID := s.foundGuild(ada, "Spam Colony")
	res := s.post(ada, fmt.Sprintf("/api/guilds/%d/boards", guildID), `{"name":"Spam"}`)
	boardID := uint64(s.jsonOf(res)["board"].(map[string]any)["id"].(float64))

	until := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	s.post(op, "/api/console/sanctions", fmt.Sprintf(`{"target":"guild:%d","kind":"suspension","reason":"Spam boards.","until":%q}`, guildID, until)).AssertCreated()
	s.post(op, "/api/console/sanctions", fmt.Sprintf(`{"target":"guild:%d","kind":"suspension","reason":"x"}`, guildID)).AssertUnprocessableEntity()

	s.eventually(func() bool {
		body, _ := s.get(ada, fmt.Sprintf("/api/boards/%d", boardID)).Json()
		return body["code"] == "guild_suspended"
	}, "the suspended guild's board still opens")
	list := s.jsonOf(s.get(ada, "/api/guilds"))["guilds"].([]any)
	for _, g := range list {
		s.NotEqual(float64(guildID), g.(map[string]any)["id"], "the suspended guild is listed")
	}
	body, _ := s.get(ada, fmt.Sprintf("/api/guilds/%d", guildID)).Json()
	s.Equal("guild_suspended", body["code"])
	s.NotEmpty(body["until"])
	// The member is fine elsewhere.
	s.get(ada, "/api/me").AssertOk()

	sanctions := s.jsonOf(s.get(op, fmt.Sprintf("/api/console/sanctions?target=guild:%d", guildID)))["sanctions"].([]any)
	s.Require().Len(sanctions, 1)
	id := uint64(sanctions[0].(map[string]any)["id"].(float64))
	s.post(op, fmt.Sprintf("/api/console/sanctions/%d/lift", id), `{}`).AssertOk()
	s.eventually(func() bool { return s.get(ada, fmt.Sprintf("/api/boards/%d", boardID)).IsSuccessful() }, "the guild is still out after the lift")
}

func (s *SanctionsTestSuite) TestMembersCannotSanction() {
	_, ada := s.register()
	s.post(ada, "/api/console/sanctions", `{"target":"member:1","kind":"ban","reason":"x"}`).AssertUnauthorized()
}
