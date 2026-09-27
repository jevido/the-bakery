package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/url"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jevido/the-bakery/services/api/app/facades"
)

// subscriberBuffer is how many events a stream may fall behind before the
// hub drops it; the client then reconnects and refetches the board.
const subscriberBuffer = 64

// BoardHub hands board events from Postgres to the streams open in this
// process. It listens on one dedicated connection, started with the first
// subscriber. If that connection drops, every stream is closed, because
// events may have been missed; clients reconnect and refetch.
type BoardHub struct {
	mu     sync.Mutex
	subs   map[uint64]map[chan []byte]struct{}
	closed bool
	start  sync.Once
	// listen is replaced in tests.
	listen func(ctx context.Context, deliver func(payload []byte)) error
}

func NewBoardHub() *BoardHub {
	return &BoardHub{subs: map[uint64]map[chan []byte]struct{}{}, listen: listenPostgres}
}

// Subscribe returns the board's events as JSON, and a function that ends the
// subscription. The channel is closed when the hub drops the subscriber.
func (h *BoardHub) Subscribe(boardID uint64) (<-chan []byte, func()) {
	h.start.Do(func() { go h.run(context.Background()) })
	ch := make(chan []byte, subscriberBuffer)
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		close(ch)
		return ch, func() {}
	}
	if h.subs[boardID] == nil {
		h.subs[boardID] = map[chan []byte]struct{}{}
	}
	h.subs[boardID][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() { h.remove(boardID, ch) }
}

func (h *BoardHub) remove(boardID uint64, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subs[boardID][ch]; ok {
		delete(h.subs[boardID], ch)
		close(ch)
		if len(h.subs[boardID]) == 0 {
			delete(h.subs, boardID)
		}
	}
}

// deliver passes one NOTIFY payload to the board's subscribers. One that is
// too far behind is dropped rather than slowing everyone down.
func (h *BoardHub) deliver(payload []byte) {
	var head struct {
		BoardID uint64 `json:"board_id"`
	}
	if json.Unmarshal(payload, &head) != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[head.BoardID] {
		select {
		case ch <- payload:
		default:
			delete(h.subs[head.BoardID], ch)
			close(ch)
		}
	}
}

// Close ends every stream and refuses new ones: the process is about to
// stop, and an open stream would keep it waiting. Clients reconnect, to
// another process.
func (h *BoardHub) Close() {
	h.mu.Lock()
	h.closed = true
	h.mu.Unlock()
	h.dropAll()
}

// dropAll closes every stream.
func (h *BoardHub) dropAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for boardID, chans := range h.subs {
		for ch := range chans {
			close(ch)
		}
		delete(h.subs, boardID)
	}
}

// run keeps listening, reconnecting with backoff (1 s up to 30 s, with
// jitter) whenever the connection fails.
func (h *BoardHub) run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		began := time.Now()
		err := h.listen(ctx, h.deliver)
		h.dropAll()
		if time.Since(began) > time.Minute {
			backoff = time.Second
		}
		facades.Log().Warningf("board events: listening stopped, retrying in %s: %v", backoff, err)
		time.Sleep(backoff + rand.N(backoff/2))
		backoff = min(backoff*2, 30*time.Second)
	}
}

func listenPostgres(ctx context.Context, deliver func(payload []byte)) error {
	conn, err := pgx.Connect(ctx, postgresURL())
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(ctx, "LISTEN "+notifyChannel); err != nil {
		return err
	}
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		deliver([]byte(n.Payload))
	}
}

// postgresURL builds a connection URL from the same settings the ORM uses.
func postgresURL() string {
	c := facades.Config()
	key := func(k string) string { return c.GetString("database.connections.postgres." + k) }
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(key("username"), key("password")),
		Host:     fmt.Sprintf("%s:%s", key("host"), key("port")),
		Path:     "/" + key("database"),
		RawQuery: url.Values{"sslmode": {key("sslmode")}}.Encode(),
	}
	return u.String()
}
