package feature

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/tests"
)

type PresenceTestSuite struct {
	featureSuite
	server *httptest.Server
}

func TestPresenceTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(PresenceTestSuite))
}

func (s *PresenceTestSuite) SetupSuite()    { s.server = httptest.NewServer(facades.Route()) }
func (s *PresenceTestSuite) TearDownSuite() { s.server.Close() }

// stream opens the board's event stream as the token's member and returns
// its presence events, and a function that closes the stream.
func (s *PresenceTestSuite) stream(token string, boardID uint64) (<-chan map[string]any, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/boards/%d/events", s.server.URL, boardID), nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, res.StatusCode)
	out := make(chan map[string]any, 16)
	go func() {
		defer res.Body.Close()
		defer close(out)
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var ev map[string]any
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev) == nil && ev["type"] == "presence" {
				out <- ev["data"].(map[string]any)
			}
		}
	}()
	return out, cancel
}

func (s *PresenceTestSuite) next(events <-chan map[string]any) map[string]any {
	select {
	case ev, open := <-events:
		s.Require().True(open, "stream ended")
		return ev
	case <-time.After(5 * time.Second):
		s.FailNow("no presence event within 5 s")
		return nil
	}
}

func members(snapshot map[string]any) []string {
	var out []string
	for _, p := range snapshot["present"].([]any) {
		out = append(out, p.(map[string]any)["display_name"].(string))
	}
	return out
}

func (s *PresenceTestSuite) TestWhoHasTheBoardOpen() {
	_, ada := s.register()
	guildID := s.foundGuild(ada, "Presence Colony")
	_, bram := s.register()
	s.join(ada, guildID, bram)
	res := s.post(ada, fmt.Sprintf("/api/guilds/%d/boards", guildID), `{"name":"Getting settled"}`)
	res.AssertCreated()
	boardID := uint64(s.jsonOf(res)["board"].(map[string]any)["id"].(float64))

	adaEvents, closeAda := s.stream(ada, boardID)
	defer closeAda()
	first := s.next(adaEvents)
	s.Equal("snapshot", first["state"])
	s.Len(members(first), 1, "ada sees herself")
	s.Equal("joined", s.next(adaEvents)["state"], "her own arrival comes round too")

	bramEvents, closeBram := s.stream(bram, boardID)
	snap := s.next(bramEvents)
	s.Len(members(snap), 2)
	bramConn := snap["conn_id"]
	joined := s.next(adaEvents)
	s.Equal("joined", joined["state"])
	s.Equal(bramConn, joined["conn_id"])
	s.NotEmpty(joined["display_name"])

	closeBram()
	left := s.next(adaEvents)
	s.Equal("left", left["state"])
	s.Equal(bramConn, left["conn_id"])

	// A stream nobody has heard from in over a minute is pruned when the next
	// one opens.
	_, err := facades.DB().Table("board_presence").Insert(map[string]any{
		"conn_id": "stale0000000000000000000000000000"[:32], "board_id": boardID,
		"member_id": 1, "seen_at": time.Now().Add(-2 * time.Minute),
	})
	s.Require().NoError(err)
	again, closeAgain := s.stream(bram, boardID)
	defer closeAgain()
	s.Len(members(s.next(again)), 2, "the stale stream is not in the snapshot")
	var gone bool
	for range 3 {
		ev := s.next(adaEvents)
		if ev["state"] == "left" && ev["conn_id"] == "stale0000000000000000000000000000"[:32] {
			gone = true
			break
		}
	}
	s.True(gone, "the stale stream's departure is announced")
}
