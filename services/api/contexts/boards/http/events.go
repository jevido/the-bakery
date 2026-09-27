package http

import (
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
	w := ctx.Response().Writer()
	flusher, ok := w.(nethttp.Flusher)
	if !ok {
		return ctx.Response().Json(contractshttp.StatusInternalServerError, contractshttp.Json{"error": "streaming is not supported"})
	}
	events, stop := c.events.Subscribe(boardID)
	defer stop()

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(contractshttp.StatusOK)
	// Ask clients to wait 3 s before reconnecting after a drop.
	fmt.Fprint(w, "retry: 3000\n\n")
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
