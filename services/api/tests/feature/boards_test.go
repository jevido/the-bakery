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

func (s *BoardsTestSuite) TestSubtasks() {
	_, token := s.register()
	guildID := s.foundGuild(token, "Subtask Colony")
	boardID := s.createBoard(token, guildID, "Getting settled")
	parentID := s.createTask(token, boardID, "Build a freezer")
	task := func(id uint64) map[string]any {
		res := s.get(token, fmt.Sprintf("/api/tasks/%d", id))
		res.AssertOk()
		return s.jsonOf(res)
	}
	boardTask := func() map[string]any {
		res := s.get(token, fmt.Sprintf("/api/boards/%d", boardID))
		res.AssertOk()
		tasks := s.jsonOf(res)["columns"].([]any)[0].(map[string]any)["tasks"].([]any)
		s.Require().Len(tasks, 1, "subtasks are not cards")
		return tasks[0].(map[string]any)
	}

	res := s.post(token, fmt.Sprintf("/api/tasks/%d/expand", parentID), `{"titles":["Dig the room","Wall it in","Add a cooler"]}`)
	res.AssertCreated()
	added := s.jsonOf(res)["subtasks"].([]any)
	s.Require().Len(added, 3)
	first := uint64(added[0].(map[string]any)["id"].(float64))
	third := uint64(added[2].(map[string]any)["id"].(float64))
	s.Nil(added[0].(map[string]any)["column"], "a subtask has no column")

	got := task(parentID)
	s.Equal(float64(3), got["task"].(map[string]any)["subtasks_total"])
	var titles []string
	for _, st := range got["subtasks"].([]any) {
		titles = append(titles, st.(map[string]any)["title"].(string))
	}
	s.Equal([]string{"Dig the room", "Wall it in", "Add a cooler"}, titles)
	s.Equal(float64(3), boardTask()["subtasks_total"])
	s.Equal(float64(0), boardTask()["subtasks_done"])

	// Tick one off.
	s.send("PATCH", token, fmt.Sprintf("/api/tasks/%d", first), `{"done":true}`).AssertOk()
	s.Equal(float64(1), boardTask()["subtasks_done"])
	// Only a subtask can be ticked off.
	s.send("PATCH", token, fmt.Sprintf("/api/tasks/%d", parentID), `{"done":true}`).AssertUnprocessableEntity()

	// Add one at the end, then move the last one to the top.
	res = s.post(token, fmt.Sprintf("/api/tasks/%d/subtasks", parentID), `{"title":"Stock it"}`)
	res.AssertCreated()
	s.move(token, third, fmt.Sprintf(`{"before_id":%d}`, first)).AssertOk()
	titles = nil
	for _, st := range task(parentID)["subtasks"].([]any) {
		titles = append(titles, st.(map[string]any)["title"].(string))
	}
	s.Equal([]string{"Add a cooler", "Dig the room", "Wall it in", "Stock it"}, titles)
	// A neighbour must be a sibling.
	s.move(token, third, fmt.Sprintf(`{"after_id":%d}`, parentID)).AssertUnprocessableEntity()

	// One level deep, and 1 to 50 at a time.
	s.post(token, fmt.Sprintf("/api/tasks/%d/expand", first), `{"titles":["Too deep"]}`).AssertUnprocessableEntity()
	s.post(token, fmt.Sprintf("/api/tasks/%d/subtasks", first), `{"title":"Too deep"}`).AssertUnprocessableEntity()
	s.post(token, fmt.Sprintf("/api/tasks/%d/expand", parentID), `{"titles":[]}`).AssertUnprocessableEntity()
	s.post(token, fmt.Sprintf("/api/tasks/%d/expand", parentID), `{"titles":["Fine"," "]}`).AssertUnprocessableEntity()
	s.Equal(float64(4), boardTask()["subtasks_total"], "a refused expand adds nothing")

	// Moving the parent keeps its subtasks with it.
	s.move(token, parentID, `{"column":"doing"}`).AssertOk()
	s.Equal(float64(4), task(parentID)["task"].(map[string]any)["subtasks_total"])

	// Others cannot see or add subtasks.
	_, stranger := s.register()
	s.get(stranger, fmt.Sprintf("/api/tasks/%d", parentID)).AssertForbidden()
	s.post(stranger, fmt.Sprintf("/api/tasks/%d/expand", parentID), `{"titles":["Mine"]}`).AssertForbidden()

	// Deleting the parent deletes its subtasks.
	s.send("DELETE", token, fmt.Sprintf("/api/tasks/%d", parentID), "").AssertNoContent()
	s.get(token, fmt.Sprintf("/api/tasks/%d", first)).AssertNotFound()
	var left int64
	left, err := facades.Orm().Query().Table("tasks").Where("parent_id", parentID).Count()
	s.Require().NoError(err)
	s.Zero(left)
}

func (s *BoardsTestSuite) TestComments() {
	_, ada := s.register()
	guildID := s.foundGuild(ada, "Comment Colony")
	_, bram := s.register()
	s.join(ada, guildID, bram)
	boardID := s.createBoard(ada, guildID, "Getting settled")
	taskID := s.createTask(ada, boardID, "Build a freezer")
	comments := fmt.Sprintf("/api/tasks/%d/comments", taskID)

	res := s.post(ada, comments, `{"body":"Needs **two** coolers."}`)
	res.AssertCreated()
	first := s.jsonOf(res)["comment"].(map[string]any)
	s.Nil(first["edited_at"])
	s.NotEmpty(first["author_name"])
	firstID := uint64(first["id"].(float64))
	s.post(bram, comments, `{"body":"On it."}`).AssertCreated()
	s.post(ada, comments, `{"body":"  "}`).AssertUnprocessableEntity()

	res = s.get(bram, comments)
	res.AssertOk()
	list := s.jsonOf(res)["comments"].([]any)
	s.Require().Len(list, 2)
	s.Equal("Needs **two** coolers.", list[0].(map[string]any)["body"], "oldest first")
	s.Equal("On it.", list[1].(map[string]any)["body"])

	// Only the author edits or deletes.
	one := fmt.Sprintf("/api/comments/%d", firstID)
	s.send("PATCH", bram, one, `{"body":"Mine now"}`).AssertForbidden()
	s.send("DELETE", bram, one, "").AssertForbidden()
	res = s.send("PATCH", ada, one, `{"body":"Needs three coolers."}`)
	res.AssertOk()
	edited := s.jsonOf(res)["comment"].(map[string]any)
	s.Equal("Needs three coolers.", edited["body"])
	s.NotNil(edited["edited_at"])

	// Outsiders see nothing.
	_, stranger := s.register()
	s.get(stranger, comments).AssertForbidden()
	s.post(stranger, comments, `{"body":"Hello"}`).AssertForbidden()
	s.send("PATCH", stranger, one, `{"body":"Hello"}`).AssertForbidden()

	s.send("DELETE", ada, one, "").AssertNoContent()
	s.send("DELETE", ada, one, "").AssertNotFound()

	// An archived guild's comments can be read, not written.
	s.post(ada, fmt.Sprintf("/api/guilds/%d/archive", guildID), "").AssertSuccessful()
	s.get(bram, comments).AssertOk()
	s.post(bram, comments, `{"body":"Too late"}`).AssertConflict()
}
