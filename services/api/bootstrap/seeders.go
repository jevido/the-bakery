package bootstrap

import (
	"github.com/goravel/framework/contracts/database/seeder"

	"github.com/jevido/the-bakery/services/api/database/seeders"
)

func Seeders() []seeder.Seeder {
	return []seeder.Seeder{
		&seeders.DatabaseSeeder{},
	}
}
