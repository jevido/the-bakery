package feature

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/tests"
)

func init() {
	// Tasks go with their board (ON DELETE CASCADE).
	guildCleanup = append(guildCleanup, func(guildID uint64) error {
		_, err := facades.DB().Table("boards").Where("guild_id", guildID).Delete()
		return err
	})
}

type BoardsTestSuite struct{ featureSuite }

func TestBoardsTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(BoardsTestSuite))
}

func (s *BoardsTestSuite) createBoard(token string, guildID uint64, name string) uint64 {
	res := s.post(token, fmt.Sprintf("/api/guilds/%d/boards", guildID), fmt.Sprintf(`{"name":%q}`, name))
	res.AssertCreated()
	return uint64(s.jsonOf(res)["board"].(map[string]any)["id"].(float64))
}

func (s *BoardsTestSuite) createTask(token string, boardID uint64, title string) uint64 {
	res := s.post(token, fmt.Sprintf("/api/boards/%d/tasks", boardID), fmt.Sprintf(`{"title":%q}`, title))
	res.AssertCreated()
	task := s.jsonOf(res)["task"].(map[string]any)
	s.Equal("backlog", task["column"])
	return uint64(task["id"].(float64))
}

// columnTitles returns each column's task titles in order.
func (s *BoardsTestSuite) columnTitles(token string, boardID uint64) map[string][]string {
	res := s.get(token, fmt.Sprintf("/api/boards/%d", boardID))
	res.AssertOk()
	out := map[string][]string{}
	for _, c := range s.jsonOf(res)["columns"].([]any) {
		col := c.(map[string]any)
		titles := []string{}
		for _, t := range col["tasks"].([]any) {
			titles = append(titles, t.(map[string]any)["title"].(string))
		}
		out[col["column"].(string)] = titles
	}
	return out
}

func (s *BoardsTestSuite) move(token string, taskID uint64, body string) response {
	return s.post(token, fmt.Sprintf("/api/tasks/%d/move", taskID), body)
}

func (s *BoardsTestSuite) TestCreateMoveAndReorder() {
	_, token := s.register()
	guildID := s.foundGuild(token, "Test Colony")
	boardID := s.createBoard(token, guildID, "Test board")

	boards := s.jsonOf(s.get(token, fmt.Sprintf("/api/guilds/%d/boards", guildID)))["boards"].([]any)
	s.Len(boards, 1)

	one := s.createTask(token, boardID, "one")
	two := s.createTask(token, boardID, "two")
	three := s.createTask(token, boardID, "three")
	s.Equal([]string{"one", "two", "three"}, s.columnTitles(token, boardID)["backlog"])

	for _, id := range []uint64{one, two, three} {
		s.move(token, id, `{"column":"todo"}`).AssertOk()
	}
	s.Equal([]string{"one", "two", "three"}, s.columnTitles(token, boardID)["todo"])

	// The third before the first.
	s.move(token, three, fmt.Sprintf(`{"column":"todo","before_id":%d}`, one)).AssertOk()
	s.Equal([]string{"three", "one", "two"}, s.columnTitles(token, boardID)["todo"])

	// Two between three and one.
	s.move(token, two, fmt.Sprintf(`{"column":"todo","after_id":%d,"before_id":%d}`, three, one)).AssertOk()
	s.Equal([]string{"three", "two", "one"}, s.columnTitles(token, boardID)["todo"])

	s.move(token, one, `{"column":"done"}`).AssertOk()
	cols := s.columnTitles(token, boardID)
	s.Equal([]string{"three", "two"}, cols["todo"])
	s.Equal([]string{"one"}, cols["done"])
	s.Empty(cols["backlog"])
	s.Empty(cols["doing"])
}

func (s *BoardsTestSuite) TestEditDeleteAndInvalid() {
	_, token := s.register()
	guildID := s.foundGuild(token, "Test Colony")
	boardID := s.createBoard(token, guildID, "Test board")
	id := s.createTask(token, boardID, "Draft")
	other := s.createTask(token, boardID, "Other")

	res := s.send("PATCH", token, fmt.Sprintf("/api/tasks/%d", id), `{"title":"Build a research bench","description":"Near the base"}`)
	res.AssertOk()
	task := s.jsonOf(res)["task"].(map[string]any)
	s.Equal("Build a research bench", task["title"])
	s.Equal("Near the base", task["description"])

	s.send("PATCH", token, fmt.Sprintf("/api/tasks/%d", id), `{"title":" "}`).AssertUnprocessableEntity()
	s.move(token, id, `{"column":"archive"}`).AssertUnprocessableEntity()
	s.move(token, id, `{"column":"todo","after_id":999999999}`).AssertUnprocessableEntity()
	s.move(token, id, fmt.Sprintf(`{"column":"todo","after_id":%d}`, other)).AssertUnprocessableEntity()
	s.post(token, fmt.Sprintf("/api/guilds/%d/boards", guildID), `{"name":""}`).AssertUnprocessableEntity()

	s.send("DELETE", token, fmt.Sprintf("/api/tasks/%d", id), "").AssertNoContent()
	s.send("DELETE", token, fmt.Sprintf("/api/tasks/%d", id), "").AssertNotFound()
	s.get(token, "/api/boards/999999999").AssertNotFound()
}

func (s *BoardsTestSuite) TestNonMemberIsRefused() {
	_, owner := s.register()
	guildID := s.foundGuild(owner, "Test Colony")
	boardID := s.createBoard(owner, guildID, "Test board")
	taskID := s.createTask(owner, boardID, "Secret")

	_, stranger := s.register()
	s.get(stranger, fmt.Sprintf("/api/guilds/%d/boards", guildID)).AssertForbidden()
	s.post(stranger, fmt.Sprintf("/api/guilds/%d/boards", guildID), `{"name":"Mine"}`).AssertForbidden()
	s.get(stranger, fmt.Sprintf("/api/boards/%d", boardID)).AssertForbidden()
	s.post(stranger, fmt.Sprintf("/api/boards/%d/tasks", boardID), `{"title":"x"}`).AssertForbidden()
	s.send("PATCH", stranger, fmt.Sprintf("/api/tasks/%d", taskID), `{"title":"x"}`).AssertForbidden()
	s.move(stranger, taskID, `{"column":"done"}`).AssertForbidden()
	s.send("DELETE", stranger, fmt.Sprintf("/api/tasks/%d", taskID), "").AssertForbidden()

	s.get("", fmt.Sprintf("/api/boards/%d", boardID)).AssertUnauthorized()
}

func (s *BoardsTestSuite) TestSeededBoard() {
	guildID := s.seededGuildID()
	token := s.login("ada@bakery.test", "password")
	var boardID uint64
	for _, b := range s.jsonOf(s.get(token, fmt.Sprintf("/api/guilds/%d/boards", guildID)))["boards"].([]any) {
		if b.(map[string]any)["name"] == "Getting settled" {
			boardID = uint64(b.(map[string]any)["id"].(float64))
		}
	}
	s.Require().NotZero(boardID, "Getting settled not seeded")
	cols := s.columnTitles(token, boardID)
	s.Contains(cols["backlog"], "Build a research bench")
}
