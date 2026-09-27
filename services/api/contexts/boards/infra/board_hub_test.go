package infra

import (
	"context"
	"fmt"
	"testing"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

// newTestHub is a hub whose "Postgres" is the returned deliver function.
func newTestHub() *BoardHub {
	h := &BoardHub{subs: map[uint64]map[chan []byte]struct{}{}}
	h.listen = func(ctx context.Context, deliver func([]byte)) error {
		select {} // never fails; tests call h.deliver directly
	}
	return h
}

func event(boardID uint64, n int) []byte {
	return []byte(fmt.Sprintf(`{"type":"task.moved","board_id":%d,"data":{"n":%d}}`, boardID, n))
}

func TestHubDeliversToTheBoardsSubscribers(t *testing.T) {
	h := newTestHub()
	one, stopOne := h.Subscribe(1)
	other, stopOther := h.Subscribe(2)
	defer stopOther()

	h.deliver(event(1, 1))
	select {
	case got := <-one:
		if string(got) != string(event(1, 1)) {
			t.Errorf("got %s", got)
		}
	default:
		t.Fatal("board 1 subscriber got nothing")
	}
	select {
	case got := <-other:
		t.Fatalf("board 2 subscriber got %s", got)
	default:
	}

	stopOne()
	if _, open := <-one; open {
		t.Error("channel still open after unsubscribing")
	}
	h.deliver(event(1, 2)) // nobody left on board 1: must not panic
	stopOne()              // twice is fine
}

func TestHubDropsASlowSubscriber(t *testing.T) {
	h := newTestHub()
	slow, stop := h.Subscribe(1)
	defer stop()
	for n := range subscriberBuffer + 1 {
		h.deliver(event(1, n))
	}
	got := 0
	for range slow {
		got++
	}
	if got != subscriberBuffer {
		t.Errorf("got %d events before the drop, want %d", got, subscriberBuffer)
	}
	if len(h.subs[1]) != 0 {
		t.Error("slow subscriber still registered")
	}
}

func TestHubDropsEveryoneWhenListeningStops(t *testing.T) {
	h := newTestHub()
	a, _ := h.Subscribe(1)
	b, _ := h.Subscribe(2)
	h.dropAll()
	if _, open := <-a; open {
		t.Error("board 1 stream still open")
	}
	if _, open := <-b; open {
		t.Error("board 2 stream still open")
	}
}

func TestBoardEventOf(t *testing.T) {
	parent := uint64(7)
	tests := []struct {
		name     string
		event    any
		wantType string
		wantTask uint64
	}{
		{"created", domain.TaskCreated{TaskID: 7, BoardID: 1}, "task.created", 7},
		{"moved", domain.TaskMoved{TaskID: 7, BoardID: 1}, "task.moved", 7},
		{"deleted", domain.TaskDeleted{TaskID: 7, BoardID: 1}, "task.deleted", 7},
		{"subtask deleted is the parent updated", domain.TaskDeleted{TaskID: 9, BoardID: 1, ParentID: &parent}, "task.updated", 7},
		{"subtask renamed is the parent updated", domain.TaskEdited{TaskID: 9, BoardID: 1, ParentID: &parent, Title: true}, "task.updated", 7},
		{"subtask added", domain.SubtaskAdded{SubtaskID: 9, ParentID: 7, BoardID: 1}, "task.updated", 7},
		{"subtask moved", domain.SubtaskMoved{SubtaskID: 9, ParentID: 7, BoardID: 1}, "task.updated", 7},
		{"commented", domain.TaskCommented{TaskID: 7, BoardID: 1, CommentID: 3}, "task.updated", 7},
		{"comment edited", domain.CommentChanged{TaskID: 7, BoardID: 1, CommentID: 3}, "task.updated", 7},
		{"column added", domain.ColumnCreated{ColumnID: 5, BoardID: 1}, "column.created", 0},
		{"column renamed", domain.ColumnRenamed{ColumnID: 5, BoardID: 1}, "column.updated", 0},
		{"column moved", domain.ColumnMoved{ColumnID: 5, BoardID: 1}, "column.moved", 0},
		{"column deleted", domain.ColumnDeleted{ColumnID: 5, BoardID: 1}, "column.deleted", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, ok := boardEventOf(tt.event)
			task, _ := ev.Data["task_id"].(uint64)
			if !ok || ev.Type != tt.wantType || ev.BoardID != 1 || task != tt.wantTask {
				t.Errorf("boardEventOf() = %+v, %v; want %s for task %d", ev, ok, tt.wantType, tt.wantTask)
			}
		})
	}
	if ev, ok := boardEventOf(struct{}{}); ok {
		t.Errorf("unknown event translated: %+v", ev)
	}
}
