package config

import (
	"github.com/jevido/the-bakery/services/api/app/facades"
)

func init() {
	config := facades.Config()
	config.Add("auth", map[string]any{
		// Authentication Defaults
		//
		// This option controls the default authentication "guard"
		// reset options for your application. You may change these defaults
		// as required, but they're a perfect start for most applications.
		"defaults": map[string]any{
			"guard": "member",
		},

		// Authentication Guards
		//
		// Next, you may define every authentication guard for your application.
		// Of course, a great default configuration has been defined for you
		// here which uses session storage and the Eloquent user provider.
		//
		// All authentication drivers have a user provider. This defines how the
		// users are actually retrieved out of your database or other storage
		// mechanisms used by this application to persist your user's data.
		//
		// Supported drivers: "jwt", "session"
		"guards": map[string]any{
			"member": map[string]any{
				"driver":   "jwt",
				"provider": "member",
			},
		},

		// Supported: "orm"
		"providers": map[string]any{
			"member": map[string]any{
				"driver": "orm",
			},
		},
	})
}
