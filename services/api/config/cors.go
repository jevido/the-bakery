package config

import (
	"fmt"
	"strings"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

func init() {
	config := facades.Config()
	config.Add("cors", map[string]any{
		// Cross-Origin Resource Sharing (CORS) Configuration
		//
		// Here you may configure your settings for cross-origin resource sharing
		// or "CORS". This determines what cross-origin operations may execute
		// in web browsers. You are free to adjust these settings as needed.
		//
		// To learn more: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
		// The Bakery: only the website may call the API from a browser, with
		// its session cookie. WEB_ORIGIN is a comma-separated list.
		"paths":           []string{"api/*"},
		"allowed_methods": []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		"allowed_origins": append(
			strings.Split(fmt.Sprint(config.Env("WEB_ORIGIN", "http://127.0.0.1:4840")), ","),
			// The operator console, on its own origin.
			strings.Split(fmt.Sprint(config.Env("CONSOLE_ORIGIN", "http://127.0.0.1:4850")), ",")...,
		),
		"allowed_headers":      []string{"Content-Type", "Authorization", "X-Bakery-Web"},
		"exposed_headers":      []string{},
		"max_age":              600,
		"supports_credentials": true,
	})
}
