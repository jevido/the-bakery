package main

import (
	"log"
	"os"

	"github.com/goravel/framework/database/migration"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/bootstrap"
)

func main() {
	app := bootstrap.Boot()

	// Deployed containers set MIGRATE_ON_START so a new release migrates the
	// database before it serves. This calls the migrator directly because
	// `artisan migrate` exits 0 even when a migration fails.
	if len(os.Args) == 1 && os.Getenv("MIGRATE_ON_START") == "true" {
		migrator := migration.NewMigrator(facades.Artisan(), facades.Schema(), facades.Config().GetString("database.migrations.table"))
		if err := migrator.Run(); err != nil {
			log.Fatalf("migrating before start: %v", err)
		}
	}

	// Start blocks until SIGINT or SIGTERM, then shuts the HTTP server down
	// gracefully: it stops accepting and lets in-flight requests finish,
	// which the 3 second request timeout (routes/api.go) bounds. Board event
	// streams are ended on the same signal (routes/api.go).
	app.Start()
}
