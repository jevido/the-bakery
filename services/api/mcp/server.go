// Package mcp is the MCP server on /mcp: an open host service over the
// guilds and boards contexts for Claude and other MCP clients. It has no
// domain of its own; every tool calls a function a context publishes, with
// the member signed in by their personal token, so it follows the same rules
// as the REST API.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jevido/the-bakery/services/api/contexts/identity"
)

// Version is reported to MCP clients.
const Version = "0.1.0"

// Handler serves MCP over streamable HTTP. It is stateless (every request
// stands alone, no server-side session), so it keeps working across rolling
// deploys, and answers in plain JSON.
func Handler() http.Handler {
	server := sdk.NewServer(&sdk.Implementation{Name: "the-bakery", Title: "The Bakery", Version: Version}, &sdk.ServerOptions{
		Instructions: "The Bakery keeps a guild's work as tasks on boards. Start with list_guilds, then list_boards " +
			"and get_board to see tasks by column (backlog, todo, doing, done), and get_task to read one task in full " +
			"with its subtasks and comments. Use the ids these return in later calls.",
	})
	addReadTools(server)
	addWriteTools(server)

	handler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server },
		&sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	// Personal tokens do not expire; they are revoked instead.
	return auth.RequireBearerToken(verify, &auth.RequireBearerTokenOptions{AllowMissingExpiration: true})(handler)
}

// verify signs the caller in with their personal token (bky_…).
func verify(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
	memberID, err := identity.VerifyPersonalToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, err)
	}
	return &auth.TokenInfo{UserID: strconv.FormatUint(memberID, 10), Extra: map[string]any{"member_id": memberID}}, nil
}

// memberID is the signed-in member, from the token verify accepted.
func memberID(ctx context.Context) (uint64, error) {
	info := auth.TokenInfoFromContext(ctx)
	if info == nil {
		return 0, errors.New("not signed in")
	}
	id, ok := info.Extra["member_id"].(uint64)
	if !ok {
		return 0, errors.New("not signed in")
	}
	return id, nil
}

// failed turns a use case's error into a tool error Claude can read, such as
// "Not a member of this guild".
func failed(err error) (*sdk.CallToolResult, error) {
	msg := err.Error()
	if msg != "" {
		msg = strings.ToUpper(msg[:1]) + msg[1:]
	}
	return &sdk.CallToolResult{IsError: true, Content: []sdk.Content{&sdk.TextContent{Text: msg}}}, nil
}
