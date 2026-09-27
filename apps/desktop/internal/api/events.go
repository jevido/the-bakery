package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// BoardEvent is one change on a board, as the board event stream sends it:
// the API's published language (docs/domain/contexts/boards/README.md).
type BoardEvent struct {
	Type    string         `json:"type"`
	BoardID uint64         `json:"board_id"`
	At      time.Time      `json:"at"`
	ActorID uint64         `json:"actor_id"`
	Data    map[string]any `json:"data"`
}

// streamIdle is how long a stream may stay silent before it counts as
// dead. The API sends a ping every 20 seconds.
const streamIdle = 60 * time.Second

// WatchBoard reads the board's event stream until ctx ends or the stream
// does, calling connected once it is open and onEvent for every event. A
// stream that stays silent for a minute is dropped. It returns
// ErrUnauthorized when the token is refused, and nil when ctx was cancelled.
func (c *Client) WatchBoard(ctx context.Context, token string, boardID uint64, connected func(), onEvent func(BoardEvent)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/boards/%d/events", c.baseURL, boardID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	// The shared client has a 10 s timeout, which would end every stream.
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return &Unreachable{URL: c.baseURL, Err: err}
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case res.StatusCode != http.StatusOK:
		return &Error{Status: res.StatusCode, Message: http.StatusText(res.StatusCode)}
	}
	connected()

	watchdog := time.AfterFunc(streamIdle, cancel)
	defer watchdog.Stop()
	err = readEvents(res.Body, func() { watchdog.Reset(streamIdle) }, onEvent)
	if ctx.Err() != nil && err != nil {
		// Cancelled by the caller or by the watchdog: either way the stream
		// is over; the caller knows which from its own context.
		return fmt.Errorf("board stream went quiet or was closed: %w", err)
	}
	return err
}

// readEvents parses server-sent events: "event:" and "data:" lines, a blank
// line ending each event, ":" comments (pings). alive is called for every
// line received.
func readEvents(r io.Reader, alive func(), onEvent func(BoardEvent)) error {
	br := bufio.NewReader(r)
	var data strings.Builder
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		alive()
		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if data.Len() > 0 {
				var ev BoardEvent
				if json.Unmarshal([]byte(data.String()), &ev) == nil {
					onEvent(ev)
				}
				data.Reset()
			}
		case strings.HasPrefix(line, ":"):
			// A ping.
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
		// "event:" repeats the type that data carries; "retry:" is ignored,
		// the caller has its own backoff.
	}
}
