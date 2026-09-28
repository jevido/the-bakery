package feature

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/tests"
)

type ClaimsTestSuite struct {
	featureSuite
	server *httptest.Server
}

func TestClaimsTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(ClaimsTestSuite))
}

func (s *ClaimsTestSuite) SetupSuite()    { s.server = httptest.NewServer(facades.Route()) }
func (s *ClaimsTestSuite) TearDownSuite() { s.server.Close() }

// colony is a guild with ada and bram, a board, a task, and an agent each.
type colony struct {
	ada, bram           string
	boardID, taskID     uint64
	adaAgent, bramAgent uint64
}

func (s *ClaimsTestSuite) colony() colony {
	var c colony
	_, c.ada = s.register()
	guildID := s.foundGuild(c.ada, "Claim Colony")
	_, c.bram = s.register()
	s.join(c.ada, guildID, c.bram)
	res := s.post(c.ada, fmt.Sprintf("/api/guilds/%d/boards", guildID), `{"name":"Getting settled"}`)
	res.AssertCreated()
	c.boardID = uint64(s.jsonOf(res)["board"].(map[string]any)["id"].(float64))
	res = s.post(c.ada, fmt.Sprintf("/api/boards/%d/tasks", c.boardID), `{"title":"Build a research bench","work_type":"coding"}`)
	res.AssertCreated()
	c.taskID = uint64(s.jsonOf(res)["task"].(map[string]any)["id"].(float64))
	agent := func(token, name string) uint64 {
		res := s.post(token, "/api/agents", agentBody(name, nil))
		res.AssertCreated()
		return uint64(s.jsonOf(res)["agent"].(map[string]any)["id"].(float64))
	}
	c.adaAgent, c.bramAgent = agent(c.ada, "Vera"), agent(c.bram, "Ivo")
	return c
}

func (s *ClaimsTestSuite) claim(token string, taskID, agentID uint64, machine string) response {
	return s.post(token, fmt.Sprintf("/api/tasks/%d/claim", taskID), fmt.Sprintf(`{"agent_id":%d,"machine_id":%q}`, agentID, machine))
}

func claimID(m map[string]any) uint64 { return uint64(m["claim"].(map[string]any)["id"].(float64)) }

func (s *ClaimsTestSuite) TestOneClaimAtATime() {
	c := s.colony()
	// Two machines claim at the same moment: one wins.
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i, try := range []struct {
		token string
		agent uint64
	}{{c.ada, c.adaAgent}, {c.bram, c.bramAgent}} {
		wg.Go(func() {
			req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/tasks/%d/claim", s.server.URL, c.taskID),
				strings.NewReader(fmt.Sprintf(`{"agent_id":%d,"machine_id":"m%d"}`, try.agent, i)))
			req.Header.Set("Authorization", "Bearer "+try.token)
			req.Header.Set("Content-Type", "application/json")
			res, err := http.DefaultClient.Do(req)
			if err == nil {
				codes[i] = res.StatusCode
				res.Body.Close()
			}
		})
	}
	wg.Wait()
	s.ElementsMatch([]int{201, 409}, codes)

	// The board shows who holds it.
	board := s.jsonOf(s.get(c.bram, fmt.Sprintf("/api/boards/%d", c.boardID)))
	task := board["columns"].([]any)[0].(map[string]any)["tasks"].([]any)[0].(map[string]any)
	s.NotNil(task["claim"])
	s.Contains([]float64{float64(c.adaAgent), float64(c.bramAgent)}, task["claim"].(map[string]any)["agent_id"])
}

func (s *ClaimsTestSuite) TestHeartbeatReleaseExpire() {
	c := s.colony()
	res := s.claim(c.ada, c.taskID, c.adaAgent, "ada-laptop")
	res.AssertCreated()
	id := claimID(s.jsonOf(res))
	s.claim(c.bram, c.taskID, c.bramAgent, "bram-pc").AssertConflict()

	s.post(c.bram, fmt.Sprintf("/api/claims/%d/heartbeat", id), "").AssertForbidden()
	res = s.post(c.ada, fmt.Sprintf("/api/claims/%d/heartbeat", id), "")
	res.AssertOk()

	// Without heartbeats the claim lapses and the task is free again.
	_, err := facades.DB().Table("task_claims").Where("id", id).Update("expires_at", time.Now().Add(-time.Second))
	s.Require().NoError(err)
	s.post(c.ada, fmt.Sprintf("/api/claims/%d/heartbeat", id), "").AssertStatus(410)
	res = s.claim(c.bram, c.taskID, c.bramAgent, "bram-pc")
	res.AssertCreated()
	second := claimID(s.jsonOf(res))

	// Released: free at once.
	s.send("DELETE", c.ada, fmt.Sprintf("/api/claims/%d", second), "").AssertForbidden()
	s.send("DELETE", c.bram, fmt.Sprintf("/api/claims/%d", second), "").AssertNoContent()
	s.send("DELETE", c.bram, fmt.Sprintf("/api/claims/%d", second), "").AssertNoContent()
	s.claim(c.ada, c.taskID, c.adaAgent, "ada-laptop").AssertCreated()
}

func (s *ClaimsTestSuite) TestDrafting() {
	c := s.colony()
	// Forbidden: no agent may take it.
	res := s.send("PATCH", c.ada, fmt.Sprintf("/api/tasks/%d", c.taskID), `{"forbidden":true}`)
	res.AssertOk()
	s.Equal(true, s.jsonOf(res)["task"].(map[string]any)["forbidden"])
	s.claim(c.ada, c.taskID, c.adaAgent, "m").AssertConflict()

	// Prioritized for Ivo: only Ivo.
	s.send("PATCH", c.ada, fmt.Sprintf("/api/tasks/%d", c.taskID), fmt.Sprintf(`{"forbidden":false,"prioritized_agent_id":%d}`, c.bramAgent)).AssertOk()
	s.claim(c.ada, c.taskID, c.adaAgent, "m").AssertConflict()
	s.claim(c.bram, c.taskID, c.bramAgent, "m").AssertCreated()

	// Cleared with 0.
	res = s.send("PATCH", c.ada, fmt.Sprintf("/api/tasks/%d", c.taskID), `{"prioritized_agent_id":0}`)
	res.AssertOk()
	s.Nil(s.jsonOf(res)["task"].(map[string]any)["prioritized_agent_id"])

	// Steering is not history.
	acts := s.jsonOf(s.get(c.ada, fmt.Sprintf("/api/tasks/%d/activity", c.taskID)))["activity"].([]any)
	for _, a := range acts {
		s.NotEqual("edited", a.(map[string]any)["kind"])
	}
}

func (s *ClaimsTestSuite) TestRefusals() {
	c := s.colony()
	_, outsider := s.register()
	s.claim(outsider, c.taskID, c.adaAgent, "m").AssertForbidden()
	// Someone else's agent.
	s.claim(c.ada, c.taskID, c.bramAgent, "m").AssertForbidden()
	s.claim(c.ada, c.taskID, c.adaAgent, "").AssertUnprocessableEntity()
	s.claim(c.ada, 999999999, c.adaAgent, "m").AssertNotFound()
	s.post(c.ada, "/api/claims/999999999/heartbeat", "").AssertNotFound()

	res := s.post(c.ada, fmt.Sprintf("/api/tasks/%d/subtasks", c.taskID), `{"title":"Haul steel"}`)
	subID := uint64(s.jsonOf(res)["subtask"].(map[string]any)["id"].(float64))
	s.claim(c.ada, subID, c.adaAgent, "m").AssertUnprocessableEntity()
	s.send("PATCH", c.ada, fmt.Sprintf("/api/tasks/%d", subID), `{"forbidden":true}`).AssertUnprocessableEntity()
}

func (s *ClaimsTestSuite) TestClaimEventsOnTheStream() {
	c := s.colony()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/boards/%d/events", s.server.URL, c.boardID), nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+c.bram)
	res, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer res.Body.Close()
	events := make(chan map[string]any, 16)
	go func() {
		defer close(events)
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			var ev map[string]any
			if line, ok := strings.CutPrefix(sc.Text(), "data: "); ok && json.Unmarshal([]byte(line), &ev) == nil && strings.HasPrefix(ev["type"].(string), "task.") {
				events <- ev
			}
		}
	}()
	time.Sleep(300 * time.Millisecond)

	r := s.claim(c.ada, c.taskID, c.adaAgent, "ada-laptop")
	id := claimID(s.jsonOf(r))
	s.send("DELETE", c.ada, fmt.Sprintf("/api/claims/%d", id), "").AssertNoContent()
	for _, want := range []string{"task.claimed", "task.released"} {
		select {
		case ev := <-events:
			s.Equal(want, ev["type"])
			s.Equal(float64(c.taskID), ev["data"].(map[string]any)["task_id"])
		case <-time.After(5 * time.Second):
			s.FailNow("no " + want + " within 5 s")
		}
	}
}
