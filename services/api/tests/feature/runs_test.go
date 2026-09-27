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

type RunsTestSuite struct {
	featureSuite
	server *httptest.Server
}

func TestRunsTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(RunsTestSuite))
}

func (s *RunsTestSuite) SetupSuite()    { s.server = httptest.NewServer(facades.Route()) }
func (s *RunsTestSuite) TearDownSuite() { s.server.Close() }

// boardWithTask makes a guild with ada and bram in it, a board and a task,
// and returns their tokens and ids.
func (s *RunsTestSuite) boardWithTask() (ada, bram string, guildID, boardID, taskID uint64) {
	_, ada = s.register()
	guildID = s.foundGuild(ada, "Run Colony")
	_, bram = s.register()
	s.join(ada, guildID, bram)
	res := s.post(ada, fmt.Sprintf("/api/guilds/%d/boards", guildID), `{"name":"Getting settled"}`)
	res.AssertCreated()
	boardID = uint64(s.jsonOf(res)["board"].(map[string]any)["id"].(float64))
	res = s.post(ada, fmt.Sprintf("/api/boards/%d/tasks", boardID), `{"title":"Build a research bench"}`)
	res.AssertCreated()
	taskID = uint64(s.jsonOf(res)["task"].(map[string]any)["id"].(float64))
	return
}

func runOf(m map[string]any) map[string]any { return m["run"].(map[string]any) }

func (s *RunsTestSuite) TestStartFinishList() {
	ada, bram, _, _, taskID := s.boardWithTask()
	res := s.post(ada, fmt.Sprintf("/api/tasks/%d/runs", taskID), `{"agent_id":9,"agent_name":"Vera","machine":"ada-laptop","branch":"bakery/1-build"}`)
	res.AssertCreated()
	run := runOf(s.jsonOf(res))
	s.Equal("running", run["status"])
	s.Equal("Vera", run["agent_name"])
	s.NotEmpty(run["member_name"])
	runID := uint64(run["id"].(float64))

	finish := `{"status":"succeeded","cost_usd":0.42,"turns":7,"summary":"Bench built.","files_changed":2,"additions":120,"deletions":14}`
	s.send("PATCH", bram, fmt.Sprintf("/api/runs/%d", runID), finish).AssertForbidden()
	res = s.send("PATCH", ada, fmt.Sprintf("/api/runs/%d", runID), finish)
	res.AssertOk()
	run = runOf(s.jsonOf(res))
	s.Equal("succeeded", run["status"])
	s.InDelta(0.42, run["cost_usd"], 0.0001)
	s.NotNil(run["ended_at"])
	s.send("PATCH", ada, fmt.Sprintf("/api/runs/%d", runID), finish).AssertUnprocessableEntity()

	// Everyone in the guild sees it, newest first.
	res = s.post(ada, fmt.Sprintf("/api/tasks/%d/runs", taskID), `{"agent_id":9,"agent_name":"Vera"}`)
	res.AssertCreated()
	list := s.jsonOf(s.get(bram, fmt.Sprintf("/api/tasks/%d/runs", taskID)))["runs"].([]any)
	s.Len(list, 2)
	s.Equal("running", list[0].(map[string]any)["status"])
	s.Equal("succeeded", list[1].(map[string]any)["status"])

	// The task's history has both.
	acts := s.jsonOf(s.get(ada, fmt.Sprintf("/api/tasks/%d/activity", taskID)))["activity"].([]any)
	kinds := map[string]int{}
	for _, a := range acts {
		kinds[a.(map[string]any)["kind"].(string)]++
	}
	s.Equal(2, kinds["run_started"])
	s.Equal(1, kinds["run_finished"])
}

func (s *RunsTestSuite) TestRefusals() {
	ada, _, _, _, taskID := s.boardWithTask()
	_, outsider := s.register()
	s.post(outsider, fmt.Sprintf("/api/tasks/%d/runs", taskID), `{"agent_id":9,"agent_name":"Vera"}`).AssertForbidden()
	s.get(outsider, fmt.Sprintf("/api/tasks/%d/runs", taskID)).AssertForbidden()
	s.post(ada, fmt.Sprintf("/api/tasks/%d/runs", taskID), `{"agent_id":0,"agent_name":"Vera"}`).AssertUnprocessableEntity()
	s.post(ada, "/api/tasks/999999999/runs", `{"agent_id":9,"agent_name":"Vera"}`).AssertNotFound()
	s.send("PATCH", ada, "/api/runs/999999999", `{"status":"failed"}`).AssertNotFound()

	res := s.post(ada, fmt.Sprintf("/api/tasks/%d/subtasks", taskID), `{"title":"Haul steel"}`)
	res.AssertCreated()
	subID := uint64(s.jsonOf(res)["subtask"].(map[string]any)["id"].(float64))
	s.post(ada, fmt.Sprintf("/api/tasks/%d/runs", subID), `{"agent_id":9,"agent_name":"Vera"}`).AssertUnprocessableEntity()

	res = s.post(ada, fmt.Sprintf("/api/tasks/%d/runs", taskID), `{"agent_id":9,"agent_name":"Vera"}`)
	runID := uint64(runOf(s.jsonOf(res))["id"].(float64))
	s.send("PATCH", ada, fmt.Sprintf("/api/runs/%d", runID), `{"status":"running"}`).AssertUnprocessableEntity()
	s.send("PATCH", ada, fmt.Sprintf("/api/runs/%d", runID), `{"status":"failed","cost_usd":-1}`).AssertUnprocessableEntity()
}

func (s *RunsTestSuite) TestOldRunReadsAsLost() {
	ada, _, _, _, taskID := s.boardWithTask()
	res := s.post(ada, fmt.Sprintf("/api/tasks/%d/runs", taskID), `{"agent_id":9,"agent_name":"Vera"}`)
	runID := uint64(runOf(s.jsonOf(res))["id"].(float64))
	_, err := facades.DB().Table("task_runs").Where("id", runID).Update("started_at", time.Now().Add(-25*time.Hour))
	s.Require().NoError(err)
	list := s.jsonOf(s.get(ada, fmt.Sprintf("/api/tasks/%d/runs", taskID)))["runs"].([]any)
	s.Equal("lost", list[0].(map[string]any)["status"])
}

func (s *RunsTestSuite) TestRunEventsOnTheStream() {
	ada, bram, _, boardID, taskID := s.boardWithTask()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/boards/%d/events", s.server.URL, boardID), nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+bram)
	res, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer res.Body.Close()
	events := make(chan map[string]any, 16)
	go func() {
		defer close(events)
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			var ev map[string]any
			if line, ok := strings.CutPrefix(sc.Text(), "data: "); ok && json.Unmarshal([]byte(line), &ev) == nil && strings.HasPrefix(ev["type"].(string), "run.") {
				events <- ev
			}
		}
	}()
	// The stream is open once its presence snapshot is out; give it a moment.
	time.Sleep(300 * time.Millisecond)

	r := s.post(ada, fmt.Sprintf("/api/tasks/%d/runs", taskID), `{"agent_id":9,"agent_name":"Vera"}`)
	runID := uint64(runOf(s.jsonOf(r))["id"].(float64))
	s.send("PATCH", ada, fmt.Sprintf("/api/runs/%d", runID), `{"status":"stopped","cost_usd":0.1}`).AssertOk()

	for _, want := range []string{"run.started", "run.finished"} {
		select {
		case ev := <-events:
			s.Equal(want, ev["type"])
			data := ev["data"].(map[string]any)
			s.Equal(float64(runID), data["run_id"])
			s.Equal(float64(taskID), data["task_id"])
			if want == "run.finished" {
				s.Equal("stopped", data["status"])
			}
		case <-time.After(5 * time.Second):
			s.FailNow("no " + want + " within 5 s")
		}
	}
}
