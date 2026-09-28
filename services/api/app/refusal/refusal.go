// Package refusal is the one error shape several contexts return when a
// sanction stops a request: identity for members, guilds for guilds, and
// every context whose use cases go through them. It is plumbing, not a
// domain model: the moderation context decides; this only carries the
// answer to the HTTP layer.
package refusal

import (
	"errors"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
)

// Codes of a sanctioned request.
const (
	AccountSuspended = "account_suspended"
	AccountBanned    = "account_banned"
	GuildSuspended   = "guild_suspended"
	GuildBanned      = "guild_banned"
)

// Refusal is a request stopped by a sanction: 403 with a sentence to show,
// a code, and until when for a suspension.
type Refusal struct {
	Code    string
	Message string
	Until   *time.Time
}

func (r *Refusal) Error() string { return r.Message }

// As finds a refusal in err.
func As(err error) (*Refusal, bool) {
	var r *Refusal
	ok := errors.As(err, &r)
	return r, ok
}

// Respond writes a refusal: {"error": sentence, "code": code, "until": time}.
// "error" stays a plain sentence, as in every refusal of the API, so any
// client shows it as it is.
func Respond(ctx contractshttp.Context, r *Refusal) contractshttp.AbortableResponse {
	body := contractshttp.Json{"error": r.Message, "code": r.Code}
	if r.Until != nil {
		body["until"] = r.Until.UTC().Format(time.RFC3339)
	}
	return ctx.Response().Json(contractshttp.StatusForbidden, body)
}
