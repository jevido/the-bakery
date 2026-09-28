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
			// Operators (the console) have guards of their own: a token
			// names its guard, so a member's token is refused here and an
			// operator's on member routes. OPERATOR_JWT_SECRET, when set,
			// signs them apart from members' tokens too.
			"operator": map[string]any{
				"driver":   "jwt",
				"provider": "operator",
				"secret":   config.Env("OPERATOR_JWT_SECRET", ""),
				"ttl":      120,
			},
			// Between the password and the authenticator code.
			"operator_challenge": map[string]any{
				"driver":   "jwt",
				"provider": "operator",
				"secret":   config.Env("OPERATOR_JWT_SECRET", ""),
				"ttl":      5,
			},
		},

		// Supported: "orm"
		"providers": map[string]any{
			"member": map[string]any{
				"driver": "orm",
			},
			"operator": map[string]any{
				"driver": "orm",
			},
		},
	})
}
