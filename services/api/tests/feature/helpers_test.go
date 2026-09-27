package feature

import (
	"fmt"
	"strings"
	"time"

	testinghttp "github.com/goravel/framework/contracts/testing/http"
	"github.com/stretchr/testify/suite"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/tests"
)

type response = testinghttp.Response

// featureSuite runs against the dev database. It registers throwaway
// members and remembers the guilds a test founds, and removes both after
// each test.
type featureSuite struct {
	suite.Suite
	tests.TestCase
	emails   []string
	guildIDs []uint64
}

// guildCleanup lets later contexts' tests remove rows that hang off a guild
// before the guild itself goes.
var guildCleanup []func(guildID uint64) error

func (s *featureSuite) TearDownTest() {
	for _, id := range s.guildIDs {
		for _, clean := range guildCleanup {
			s.NoError(clean(id))
		}
		_, err := facades.DB().Table("guild_memberships").Where("guild_id", id).Delete()
		s.NoError(err)
		_, err = facades.DB().Table("guilds").Where("id", id).Delete()
		s.NoError(err)
	}
	for _, email := range s.emails {
		var ids []uint64
		s.NoError(facades.Orm().Query().Table("members").Where("email", email).Pluck("id", &ids))
		for _, id := range ids {
			_, err := facades.DB().Table("guild_memberships").Where("member_id", id).Delete()
			s.NoError(err)
		}
		_, err := facades.DB().Table("members").Where("email", email).Delete()
		s.NoError(err)
	}
	s.emails, s.guildIDs = nil, nil
}

func (s *featureSuite) newEmail() string {
	email := fmt.Sprintf("test-%d@bakery.test", time.Now().UnixNano())
	s.emails = append(s.emails, email)
	return email
}

// register creates a throwaway member and returns its email and token.
func (s *featureSuite) register() (email, token string) {
	email = s.newEmail()
	res := s.post("", "/api/register", fmt.Sprintf(`{"email":%q,"display_name":"Tester","password":"firstlanding"}`, email))
	res.AssertCreated()
	return email, s.jsonOf(res)["token"].(string)
}

func (s *featureSuite) login(email, password string) string {
	res := s.post("", "/api/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, password))
	res.AssertOk()
	return s.jsonOf(res)["token"].(string)
}

// foundGuild founds a guild as the token's member and returns its id.
func (s *featureSuite) foundGuild(token, name string) uint64 {
	res := s.post(token, "/api/guilds", fmt.Sprintf(`{"name":%q}`, name))
	res.AssertCreated()
	id := uint64(s.jsonOf(res)["guild"].(map[string]any)["id"].(float64))
	s.guildIDs = append(s.guildIDs, id)
	return id
}

func (s *featureSuite) send(method, token, uri, body string) response {
	req := s.Http(s.T()).WithHeader("Content-Type", "application/json")
	if token != "" {
		req = req.WithToken(token)
	}
	var (
		res response
		err error
	)
	switch method {
	case "GET":
		res, err = req.Get(uri)
	case "POST":
		res, err = req.Post(uri, strings.NewReader(body))
	case "PATCH":
		res, err = req.Patch(uri, strings.NewReader(body))
	case "DELETE":
		res, err = req.Delete(uri, strings.NewReader(body))
	default:
		s.FailNow("unsupported method " + method)
	}
	s.Require().NoError(err)
	return res
}

func (s *featureSuite) get(token, uri string) response { return s.send("GET", token, uri, "") }

func (s *featureSuite) post(token, uri, body string) response {
	return s.send("POST", token, uri, body)
}

func (s *featureSuite) jsonOf(res response) map[string]any {
	got, err := res.Json()
	s.Require().NoError(err)
	return got
}

// seededGuildID is the id of the seeded "First Colony" (task db:seed).
func (s *featureSuite) seededGuildID() uint64 {
	var ids []uint64
	s.Require().NoError(facades.Orm().Query().Table("guilds").Where("name", "First Colony").OrderBy("id").Pluck("id", &ids))
	if len(ids) == 0 {
		s.T().Skip("First Colony not seeded (go run . artisan db:seed)")
	}
	return ids[0]
}
