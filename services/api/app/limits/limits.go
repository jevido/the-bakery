// Package limits is the API's rate limiting: the client's IP, the 429 answer
// every limit gives, and a hook that tells the moderation context about each
// refusal (an abuse signal). Each context declares its own limits where it
// registers its routes, with facades.RateLimiter().For and By below.
package limits

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
)

// Names of the limits, as they appear in abuse signals.
const (
	Register    = "register"     // 5 an hour per IP
	Login       = "login"        // 10 a minute per IP
	Writes      = "writes"       // 120 a minute per member
	MCP         = "mcp"          // 300 a minute per personal token
	GuildCreate = "guild-create" // 5 a day per member
)

var hit func(ctx context.Context, limit string, memberID uint64, ip string)

// OnHit sets what happens when a limit refuses a request, besides the 429:
// the moderation context records it. memberID is 0 when nobody is signed in.
func OnHit(f func(ctx context.Context, limit string, memberID uint64, ip string)) { hit = f }

// By keys a limit and gives it the API's refusal. memberID is who the
// refusal is recorded against (0 for a limit per IP).
func By(l contractshttp.Limit, name, key string, memberID uint64) contractshttp.Limit {
	return l.By(name + ":" + key).Response(func(ctx contractshttp.Context) { Refuse(ctx, name, memberID) })
}

// Allow takes one request from a limit outside the throttle middleware, for
// a limit checked inside another middleware (writes, in RequireMember). On
// a refusal it has answered 429 and aborted; the caller just returns.
func Allow(ctx contractshttp.Context, l contractshttp.Limit, name, key string, memberID uint64) bool {
	tokens, remaining, reset, ok, err := l.GetStore().Take(ctx, "throttle:"+name+":"+key)
	if err != nil {
		// A broken cache must not take the API down with it.
		return true
	}
	ctx.Response().Header("X-RateLimit-Limit", strconv.FormatUint(tokens, 10))
	ctx.Response().Header("X-RateLimit-Remaining", strconv.FormatUint(remaining, 10))
	if ok {
		return true
	}
	resetAt := time.Unix(0, int64(reset))
	ctx.Response().Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
	ctx.Response().Header("Retry-After", strconv.Itoa(max(1, int(time.Until(resetAt).Seconds()))))
	Refuse(ctx, name, memberID)
	return false
}

// Refuse answers 429 with a sentence and the limit's name, and reports the
// hit. The throttle middleware has already set Retry-After.
func Refuse(ctx contractshttp.Context, name string, memberID uint64) {
	if hit != nil {
		hit(ctx.Context(), name, memberID, ClientIP(ctx))
	}
	msg := "Too many requests. Wait a moment, then try again."
	if n, err := strconv.Atoi(ctx.Response().Origin().Header().Get("Retry-After")); err == nil {
		msg = "Too many requests. Try again in " + wait(n) + "."
	}
	_ = ctx.Response().Json(contractshttp.StatusTooManyRequests, contractshttp.Json{"error": msg, "code": "rate_limited", "limit": name}).Abort()
}

// ClientIP is the address the request came from. Behind the reverse proxy
// (a private or loopback peer) it is the right-most public address in
// X-Forwarded-For, the one the proxy saw; addresses further left are the
// client's own claims and are ignored. Otherwise it is the peer itself.
func ClientIP(ctx contractshttp.Context) string {
	req := ctx.Request().Origin()
	return clientIP(req.RemoteAddr, req.Header.Get("X-Forwarded-For"))
}

func clientIP(remoteAddr, forwardedFor string) string {
	peer, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		peer = remoteAddr
	}
	if !internal(peer) || forwardedFor == "" {
		return peer
	}
	hops := strings.Split(forwardedFor, ",")
	for i := len(hops) - 1; i >= 0; i-- {
		ip := strings.TrimSpace(hops[i])
		if net.ParseIP(ip) != nil && !internal(ip) {
			return ip
		}
	}
	return peer
}

func internal(s string) bool {
	ip := net.ParseIP(s)
	return ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// wait says how long n seconds is, roughly, for a person.
func wait(n int) string {
	switch {
	case n <= 1:
		return "a second"
	case n < 90:
		return fmt.Sprintf("%d seconds", n)
	case n < 90*60:
		return fmt.Sprintf("%d minutes", (n+30)/60)
	default:
		return fmt.Sprintf("%d hours", (n+1800)/3600)
	}
}
