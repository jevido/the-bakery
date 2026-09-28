package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/http/limit"

	"github.com/jevido/the-bakery/services/api/app/limits"
	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

// Limit holds each personal token to 300 MCP requests a minute. The token
// is not verified yet at this point, so the key is its hash; the member is
// looked up only when a request is refused, for the abuse signal. Requests
// without a token share one limit per IP (they are refused anyway).
func Limit(ctx contractshttp.Context) contractshttp.Limit {
	token, ok := strings.CutPrefix(ctx.Request().Header("Authorization"), "Bearer ")
	if !ok || token == "" {
		return limits.By(limit.PerMinute(300), limits.MCP, "ip:"+limits.ClientIP(ctx), 0)
	}
	sum := sha256.Sum256([]byte(token))
	return limit.PerMinute(300).By(limits.MCP + ":" + hex.EncodeToString(sum[:])).Response(func(ctx contractshttp.Context) {
		memberID, _ := identity.VerifyPersonalToken(context.WithoutCancel(ctx.Context()), token)
		limits.Refuse(ctx, limits.MCP, memberID)
	})
}
