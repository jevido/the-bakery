package feature

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/tests"
)

type ReportsTestSuite struct {
	operatorSuite
}

func TestReportsTestSuite(t *testing.T) {
	tests.RequireDatabase(t)
	suite.Run(t, new(ReportsTestSuite))
}

func (s *ReportsTestSuite) openReports(op string) []map[string]any {
	var out []map[string]any
	for _, r := range s.jsonOf(s.get(op, "/api/console/reports?status=open"))["reports"].([]any) {
		out = append(out, r.(map[string]any))
	}
	return out
}

func (s *ReportsTestSuite) TestReportDismissAndAction() {
	opEmail, secret := s.operator("a long enough secret")
	op := s.signIn(opEmail, "a long enough secret", secret)
	_, spammer := s.register()
	guildID := s.foundGuild(spammer, "Spam Guild")
	_, cas := s.register()
	s.T().Cleanup(func() {
		_, _ = facades.DB().Table("reports").Where("target_id", guildID).Where("target_kind", "guild").Delete()
	})

	res := s.post(cas, "/api/reports", fmt.Sprintf(`{"target_kind":"guild","target_id":%d,"reason":"Sells fake colonists."}`, guildID))
	res.AssertCreated()
	id := s.jsonOf(res)["report"].(map[string]any)["id"].(float64)
	var mine map[string]any
	for _, r := range s.openReports(op) {
		if r["id"] == id {
			mine = r
		}
	}
	s.Require().NotNil(mine, "the report is in the queue")
	s.Equal("Sells fake colonists.", mine["reason"])

	s.post(op, fmt.Sprintf("/api/console/reports/%d/dismiss", int(id)), "").AssertOk()
	for _, r := range s.openReports(op) {
		s.NotEqual(id, r["id"], "a dismissed report is still open")
	}
	s.post(op, fmt.Sprintf("/api/console/reports/%d/dismiss", int(id)), "").AssertConflict()

	// From a report to a ban.
	res = s.post(cas, "/api/reports", fmt.Sprintf(`{"target_kind":"guild","target_id":%d,"reason":"Still at it."}`, guildID))
	second := int(s.jsonOf(res)["report"].(map[string]any)["id"].(float64))
	res = s.post(op, "/api/console/sanctions", fmt.Sprintf(`{"target":"guild:%d","kind":"ban","reason":"Scams (report %d)."}`, guildID, second))
	res.AssertCreated()
	sanctionID := int(s.jsonOf(res)["sanction"].(map[string]any)["id"].(float64))
	res = s.post(op, fmt.Sprintf("/api/console/reports/%d/action", second), fmt.Sprintf(`{"sanction_id":%d}`, sanctionID))
	res.AssertOk()
	s.Equal("actioned", s.jsonOf(res)["report"].(map[string]any)["status"])

	// Filed and handled are in the audit log.
	s.NotEmpty(s.audit(op, fmt.Sprintf("action=report.filed&target_kind=guild&target_id=%d", guildID)))
	s.NotEmpty(s.audit(op, fmt.Sprintf("action=report.actioned&target_kind=report&target_id=%d", second)))
}

func (s *ReportsTestSuite) TestReportLimitsAndRefusals() {
	_, cas := s.register()
	casID := uint64(s.jsonOf(s.get(cas, "/api/me"))["member"].(map[string]any)["id"].(float64))
	_, other := s.register()
	otherID := uint64(s.jsonOf(s.get(other, "/api/me"))["member"].(map[string]any)["id"].(float64))
	s.T().Cleanup(func() { _, _ = facades.DB().Table("reports").Where("by_member_id", casID).Delete() })

	s.post(cas, "/api/reports", fmt.Sprintf(`{"target_kind":"member","target_id":%d,"reason":"me"}`, casID)).AssertUnprocessableEntity()
	s.post(cas, "/api/reports", `{"target_kind":"member","target_id":999999999,"reason":"ghost"}`).AssertNotFound()
	s.post(cas, "/api/reports", fmt.Sprintf(`{"target_kind":"member","target_id":%d,"reason":" "}`, otherID)).AssertUnprocessableEntity()
	s.post("", "/api/reports", fmt.Sprintf(`{"target_kind":"member","target_id":%d,"reason":"x"}`, otherID)).AssertUnauthorized()
	for i := range 10 {
		s.post(cas, "/api/reports", fmt.Sprintf(`{"target_kind":"member","target_id":%d,"reason":"number %d"}`, otherID, i)).AssertCreated()
	}
	s.post(cas, "/api/reports", fmt.Sprintf(`{"target_kind":"member","target_id":%d,"reason":"eleven"}`, otherID)).AssertStatus(429)
}
