package tests

import (
	"testing"

	frameworktesting "github.com/goravel/framework/testing"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/bootstrap"
)

func init() {
	bootstrap.Boot()
}

type TestCase struct {
	frameworktesting.TestCase
}

// RequireDatabase skips the test when the dev database is not reachable, so
// the pure unit tests still run without Postgres. Feature tests call it
// first; run `task db:up` and `task api:migrate` to include them.
func RequireDatabase(t *testing.T) {
	t.Helper()
	db, err := facades.Orm().DB()
	if err == nil {
		err = db.PingContext(t.Context())
	}
	if err != nil {
		t.Skipf("dev database not reachable (task db:up): %v", err)
	}
}
