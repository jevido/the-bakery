package feature

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/tests"
)

func init() {
	// Files go with their agent (ON DELETE CASCADE).
	memberCleanup = append(memberCleanup, func(memberID uint64) error {
		_, err := facades.DB().Table("agents").Where("owner_member_id", memberID).Delete()
		return err
	})
}

type AgentsTestSuite struct{ featureSuite }

func TestAgentsTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(AgentsTestSuite))
}

const goTestsSkill = "---\nname: go-tests\ndescription: Write table-driven Go tests.\nlicense: MIT\n---\n\nUse t.Run.\n"

func agentBody(name string, extra map[string]any) string {
	body := map[string]any{
		"name": name, "title": "Backend engineer", "traits": []string{"careful", "tidy"},
		"work_priorities": map[string]int{"coding": 1, "testing": 2},
		"files":           []map[string]string{{"path": "go-tests/SKILL.md", "content": goTestsSkill}},
	}
	for k, v := range extra {
		body[k] = v
	}
	b, _ := json.Marshal(body)
	return string(b)
}

// put sends a whole agent with If-Match.
func (s *AgentsTestSuite) put(token string, id uint64, revision int, body string) response {
	res, err := s.Http(s.T()).
		WithHeaders(map[string]string{"Authorization": "Bearer " + token, "If-Match": fmt.Sprint(revision), "Content-Type": "application/json"}).
		Put(fmt.Sprintf("/api/agents/%d", id), strings.NewReader(body))
	s.Require().NoError(err)
	return res
}

func (s *AgentsTestSuite) TestCreateReviseAndConflict() {
	_, ada := s.register()
	res := s.post(ada, "/api/agents", agentBody("Vera", nil))
	res.AssertCreated()
	vera := s.jsonOf(res)["agent"].(map[string]any)
	id := uint64(vera["id"].(float64))
	s.Equal("vera", vera["slug"])
	s.Equal(float64(1), vera["revision"])
	s.Equal("sonnet", vera["model"])
	s.Equal("manual", vera["permission_mode"])
	s.NotEmpty(vera["portrait_seed"])

	got := s.jsonOf(s.get(ada, fmt.Sprintf("/api/agents/%d", id)))["agent"].(map[string]any)
	files := got["files"].([]any)
	s.Require().Len(files, 1)
	s.Equal("go-tests/SKILL.md", files[0].(map[string]any)["path"])
	s.Equal(goTestsSkill, files[0].(map[string]any)["content"], "front matter kept as written")
	s.Len(files[0].(map[string]any)["sha256"], 64)
	s.Equal("Write table-driven Go tests.", got["skills"].([]any)[0].(map[string]any)["description"])

	list := s.jsonOf(s.get(ada, "/api/agents"))["agents"].([]any)
	s.Require().Len(list, 1)
	s.Nil(list[0].(map[string]any)["files"], "lists leave files out")

	// A write based on the current revision goes through; the same again is stale.
	res = s.put(ada, id, 1, agentBody("Vera", map[string]any{"title": "Staff engineer"}))
	res.AssertOk()
	s.Equal(float64(2), s.jsonOf(res)["agent"].(map[string]any)["revision"])
	res = s.put(ada, id, 1, agentBody("Vera", map[string]any{"title": "Lost"}))
	res.AssertStatus(409)
	current := s.jsonOf(res)["agent"].(map[string]any)
	s.Equal(float64(2), current["revision"], "a conflict carries the agent as it is now")
	s.Equal("Staff engineer", current["title"])

	res, err := s.Http(s.T()).WithHeader("Authorization", "Bearer "+ada).Put(fmt.Sprintf("/api/agents/%d", id), strings.NewReader(agentBody("Vera", nil)))
	s.Require().NoError(err)
	res.AssertStatus(428)

	// Only the owner sees it.
	_, bram := s.register()
	s.get(bram, fmt.Sprintf("/api/agents/%d", id)).AssertNotFound()
	s.put(bram, id, 2, agentBody("Mine", nil)).AssertNotFound()
	s.Empty(s.jsonOf(s.get(bram, "/api/agents"))["agents"])
}

func (s *AgentsTestSuite) TestRules() {
	_, ada := s.register()
	bad := func(extra map[string]any) response {
		return s.post(ada, "/api/agents", agentBody("Ivo", extra))
	}
	bad(map[string]any{"traits": []string{"careful", "fast-worker"}}).AssertUnprocessableEntity()
	bad(map[string]any{"permission_mode": "bypassPermissions"}).AssertUnprocessableEntity()
	bad(map[string]any{"model": "gpt-4"}).AssertUnprocessableEntity()
	res := bad(map[string]any{"files": []map[string]string{{"path": "go-tests/notes.md", "content": "no manifest"}}})
	res.AssertUnprocessableEntity()
	s.Equal("go-tests/", s.jsonOf(res)["path"])
	res = bad(map[string]any{"files": []map[string]string{{"path": "../../etc/passwd", "content": "x"}}})
	res.AssertUnprocessableEntity()
	s.Equal("../../etc/passwd", s.jsonOf(res)["path"])
	res = bad(map[string]any{"files": []map[string]string{{"path": "big/SKILL.md", "content": strings.Repeat("x", 3<<20)}}})
	res.AssertStatus(413)

	traits := s.jsonOf(s.get(ada, "/api/agent-traits"))
	s.Len(traits["traits"], 7)
	s.NotContains(traits["permission_modes"], "bypassPermissions")
}

func (s *AgentsTestSuite) TestDeleteLeavesATombstone() {
	_, ada := s.register()
	res := s.post(ada, "/api/agents", agentBody("Vera", nil))
	res.AssertCreated()
	id := uint64(s.jsonOf(res)["agent"].(map[string]any)["id"].(float64))
	s.post(ada, "/api/agents", agentBody("Vera", nil)).AssertUnprocessableEntity() // slug taken
	before := time.Now().UTC().Add(-time.Second).Format(time.RFC3339Nano)

	res, err := s.Http(s.T()).WithHeaders(map[string]string{"Authorization": "Bearer " + ada, "If-Match": "1"}).Delete(fmt.Sprintf("/api/agents/%d", id), nil)
	s.Require().NoError(err)
	res.AssertNoContent()
	s.get(ada, fmt.Sprintf("/api/agents/%d", id)).AssertNotFound()
	s.Empty(s.jsonOf(s.get(ada, "/api/agents"))["agents"])

	changed := s.jsonOf(s.get(ada, "/api/agents?since="+before))["agents"].([]any)
	s.Require().Len(changed, 1)
	s.Equal(true, changed[0].(map[string]any)["deleted"], "since lists the tombstone")

	// The slug is free again.
	s.post(ada, "/api/agents", agentBody("Vera", nil)).AssertCreated()
}

func (s *AgentsTestSuite) TestShareAndRecruit() {
	_, ada := s.register()
	guildID := s.foundGuild(ada, "Recruit Colony")
	_, bram := s.register()
	s.join(ada, guildID, bram)
	_, cas := s.register()

	res := s.post(ada, "/api/agents", agentBody("Vera", nil))
	res.AssertCreated()
	vera := uint64(s.jsonOf(res)["agent"].(map[string]any)["id"].(float64))
	share := fmt.Sprintf("/api/agents/%d/shares/%d", vera, guildID)
	s.send("PUT", ada, share, "").AssertNoContent()
	s.send("PUT", ada, share, "").AssertNoContent() // twice is fine
	s.Equal([]any{float64(guildID)}, s.jsonOf(s.get(ada, fmt.Sprintf("/api/agents/%d", vera)))["shared_with"])
	s.send("PUT", bram, share, "").AssertNotFound() // not bram's to share

	// Bram sees it in the guild and recruits it.
	listed := s.jsonOf(s.get(bram, fmt.Sprintf("/api/guilds/%d/agents", guildID)))["agents"].([]any)
	s.Require().Len(listed, 1)
	s.Equal("Vera", listed[0].(map[string]any)["name"])
	s.NotEmpty(listed[0].(map[string]any)["owner_name"])
	s.Equal("go-tests", listed[0].(map[string]any)["skills"].([]any)[0].(map[string]any)["name"])
	s.Nil(listed[0].(map[string]any)["files"], "a guild sees no file contents")

	recruit := fmt.Sprintf(`{"agent_id":%d,"guild_id":%d}`, vera, guildID)
	res = s.post(bram, "/api/agents/recruit", recruit)
	res.AssertCreated()
	copy := s.jsonOf(res)["agent"].(map[string]any)
	copyID := uint64(copy["id"].(float64))
	s.Equal(float64(vera), copy["origin_agent_id"])
	s.Equal("vera", copy["slug"])
	res = s.post(bram, "/api/agents/recruit", recruit)
	res.AssertCreated()
	s.Equal("vera-2", s.jsonOf(res)["agent"].(map[string]any)["slug"], "a second copy gets a new slug")

	// Bram tunes his copy; Ada revises hers.
	s.put(bram, copyID, 1, agentBody("Vera", map[string]any{"work_priorities": map[string]int{"review": 1}})).AssertOk()
	s.put(ada, vera, 1, agentBody("Vera", map[string]any{"title": "Principal engineer"})).AssertOk()
	origin := s.jsonOf(s.get(bram, fmt.Sprintf("/api/agents/%d/origin", copyID)))
	s.Equal(true, origin["newer"])
	s.Equal(float64(2), origin["current_revision"])

	res, err := s.Http(s.T()).WithHeaders(map[string]string{"Authorization": "Bearer " + bram, "If-Match": "2"}).Post(fmt.Sprintf("/api/agents/%d/pull-origin", copyID), nil)
	s.Require().NoError(err)
	res.AssertOk()
	pulled := s.jsonOf(res)["agent"].(map[string]any)
	s.Equal("Principal engineer", pulled["title"])
	s.Equal(map[string]any{"review": float64(1)}, pulled["work_priorities"], "his own priorities stay")
	s.Equal(false, s.jsonOf(s.get(bram, fmt.Sprintf("/api/agents/%d/origin", copyID)))["newer"])
	s.get(ada, fmt.Sprintf("/api/agents/%d", copyID)).AssertNotFound() // the copy is bram's

	// Outsiders cannot see or recruit; unsharing stops new recruits.
	s.get(cas, fmt.Sprintf("/api/guilds/%d/agents", guildID)).AssertForbidden()
	s.post(cas, "/api/agents/recruit", recruit).AssertForbidden()
	s.send("DELETE", ada, share, "").AssertNoContent()
	s.post(bram, "/api/agents/recruit", recruit).AssertNotFound()
	s.get(bram, fmt.Sprintf("/api/agents/%d", copyID)).AssertOk() // copies stay

	// A deleted origin leaves the copy alone.
	res, err = s.Http(s.T()).WithHeaders(map[string]string{"Authorization": "Bearer " + ada, "If-Match": "2"}).Delete(fmt.Sprintf("/api/agents/%d", vera), nil)
	s.Require().NoError(err)
	res.AssertNoContent()
	s.Equal(true, s.jsonOf(s.get(bram, fmt.Sprintf("/api/agents/%d/origin", copyID)))["gone"])
}
