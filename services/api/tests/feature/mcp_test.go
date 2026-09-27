package feature

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/mcp"
	"github.com/jevido/the-bakery/services/api/tests"
)

type MCPTestSuite struct {
	featureSuite
	server *httptest.Server
}

func TestMCPTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(MCPTestSuite))
}

func (s *MCPTestSuite) SetupSuite()    { s.server = httptest.NewServer(mcp.Handler()) }
func (s *MCPTestSuite) TearDownSuite() { s.server.Close() }

func (s *MCPTestSuite) TearDownTest() {
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

type bearer struct {
	token string
	next  http.RoundTripper
}

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return b.next.RoundTrip(r)
}

// connect opens an MCP session signed in with a new personal token of the
// session's member.
func (s *MCPTestSuite) connect(session string) *sdk.ClientSession {
	res := s.post(session, "/api/tokens", `{"name":"test claude"}`)
	res.AssertCreated()
	token := s.jsonOf(res)["token"].(string)
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := client.Connect(s.T().Context(), &sdk.StreamableClientTransport{
		Endpoint:   s.server.URL,
		HTTPClient: &http.Client{Transport: bearer{token: token, next: http.DefaultTransport}},
	}, nil)
	s.Require().NoError(err)
	s.T().Cleanup(func() { cs.Close() })
	return cs
}

func (s *MCPTestSuite) call(cs *sdk.ClientSession, name string, args map[string]any) (map[string]any, bool) {
	res, err := cs.CallTool(s.T().Context(), &sdk.CallToolParams{Name: name, Arguments: args})
	s.Require().NoError(err)
	if res.IsError {
		return map[string]any{"error": res.Content[0].(*sdk.TextContent).Text}, false
	}
	b, err := json.Marshal(res.StructuredContent)
	s.Require().NoError(err)
	var out map[string]any
	s.Require().NoError(json.Unmarshal(b, &out))
	return out, true
}

func (s *MCPTestSuite) TestReadTools() {
	_, session := s.register()
	guildID := s.foundGuild(session, "MCP Colony")
	board := s.post(session, fmt.Sprintf("/api/guilds/%d/boards", guildID), `{"name":"Getting settled"}`)
	board.AssertCreated()
	boardID := s.jsonOf(board)["board"].(map[string]any)["id"].(float64)
	s.post(session, fmt.Sprintf("/api/boards/%d/tasks", int(boardID)), `{"title":"Build a research bench"}`).AssertCreated()

	cs := s.connect(session)
	tools, err := cs.ListTools(s.T().Context(), nil)
	s.Require().NoError(err)
	var names []string
	for _, t := range tools.Tools {
		names = append(names, t.Name)
	}
	s.Subset(names, []string{"list_guilds", "list_boards", "get_board"})

	out, ok := s.call(cs, "list_guilds", nil)
	s.Require().True(ok, out)
	s.Equal("MCP Colony", out["guilds"].([]any)[0].(map[string]any)["name"])

	out, ok = s.call(cs, "list_boards", map[string]any{"guild_id": guildID})
	s.Require().True(ok, out)
	s.Equal("Getting settled", out["boards"].([]any)[0].(map[string]any)["name"])

	out, ok = s.call(cs, "get_board", map[string]any{"board_id": boardID})
	s.Require().True(ok, out)
	cols := out["columns"].([]any)
	s.Require().Len(cols, 4)
	s.Equal("Build a research bench", cols[0].(map[string]any)["tasks"].([]any)[0].(map[string]any)["title"])

	// Someone else's guild is out of reach, as on REST.
	_, stranger := s.register()
	other := s.connect(stranger)
	out, ok = s.call(other, "get_board", map[string]any{"board_id": boardID})
	s.False(ok)
	s.Equal("Not a member of this guild", out["error"])
	out, _ = s.call(other, "list_guilds", nil)
	s.Empty(out["guilds"])
	s.False(slices.Contains(names, ""), "no unnamed tools")
}

func (s *MCPTestSuite) TestNeedsAToken() {
	for _, header := range []string{"", "Bearer nope", "Bearer bky_" + "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"} {
		req, _ := http.NewRequest("POST", s.server.URL, nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		res, err := http.DefaultClient.Do(req)
		s.Require().NoError(err)
		res.Body.Close()
		s.Equal(http.StatusUnauthorized, res.StatusCode, header)
	}
}
