package main

import (
	"github.com/jevido/the-bakery/services/api/bootstrap"
)

func main() {
	app := bootstrap.Boot()

	app.Start()
}
