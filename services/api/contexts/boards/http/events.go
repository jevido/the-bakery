package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	nethttp "net/http"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
)

// BoardEvents is where the stream gets a board's events from: JSON board
// events, and a function to stop. The channel closes when the source drops
// the stream.
type BoardEvents interface {
	Subscribe(boardID uint64) (<-chan []byte, func())
}

const pingEvery = 20 * time.Second

// newConnID names one open stream, for presence.
func newConnID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type EventsController struct {
	service  *app.Service
	events   BoardEvents
	memberID MemberID
}

func NewEventsController(service *app.Service, events BoardEvents, memberID MemberID) *EventsController {
	return &EventsController{service: service, events: events, memberID: memberID}
}

// Stream sends the board's events as server-sent events until the client
// goes away, with a ping comment every 20 seconds so proxies keep the
// connection open. It writes the response itself, like /mcp.
func (c *EventsController) Stream(ctx contractshttp.Context) contractshttp.Response {
	boardID, ok := routeID(ctx, "board")
	if !ok {
		return notFound(ctx)
	}
	me, _ := c.memberID(ctx)
	if err := c.service.WatchBoard(ctx.Context(), boardID, me); err != nil {
		return failure(ctx, err)
	}
	connID := newConnID()
	w := ctx.Response().Writer()
	flusher, ok := w.(nethttp.Flusher)
	if !ok {
		return ctx.Response().Json(contractshttp.StatusInternalServerError, contractshttp.Json{"error": "streaming is not supported"})
	}
	events, stop := c.events.Subscribe(boardID)
	defer stop()
	// The member is present while this stream is open. The request's context
	// is gone by the time the deferred leave runs, so it gets its own.
	present, err := c.service.EnterBoard(ctx.Context(), boardID, me, connID)
	if err != nil {
		return failure(ctx, err)
	}
	defer func() {
		leaveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.service.LeaveBoard(leaveCtx, boardID, me, connID)
	}()

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(contractshttp.StatusOK)
	// Ask clients to wait 3 s before reconnecting after a drop.
	fmt.Fprint(w, "retry: 3000\n\n")
	// Everyone here now, before any other event.
	snapshot := make([]map[string]any, len(present))
	for i, p := range present {
		snapshot[i] = map[string]any{"conn_id": p.ConnID, "member_id": p.MemberID, "display_name": p.DisplayName}
	}
	b, _ := json.Marshal(map[string]any{
		"type": "presence", "board_id": boardID, "at": time.Now().UTC(), "actor_id": me,
		"data": map[string]any{"state": "snapshot", "conn_id": connID, "present": snapshot},
	})
	fmt.Fprintf(w, "event: presence\ndata: %s\n\n", b)
	flusher.Flush()

	ping := time.NewTicker(pingEvery)
	defer ping.Stop()
	gone := ctx.Request().Origin().Context().Done()
	for {
		select {
		case <-gone:
			return nil
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			_ = c.service.StillOnBoard(ctx.Context(), connID)
		case payload, open := <-events:
			if !open {
				// Dropped: the client reconnects and refetches.
				return nil
			}
			var head struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(payload, &head)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", head.Type, payload)
		}
		flusher.Flush()
	}
}
