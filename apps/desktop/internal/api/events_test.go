package api

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReadEvents(t *testing.T) {
	stream := "retry: 3000\n\n" +
		": ping\n\n" +
		"event: task.moved\n" +
		`data: {"type":"task.moved","board_id":1,"actor_id":2,"data":{"task_id":6,"to":3}}` + "\n\n" +
		"event: column.created\r\n" +
		`data: {"type":"column.created","board_id":1,"data":{"column_id":9,"name":"Review"}}` + "\r\n\r\n" +
		"data: not json\n\n"
	var got []BoardEvent
	lines := 0
	err := readEvents(strings.NewReader(stream), func() { lines++ }, func(ev BoardEvent) { got = append(got, ev) })
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("end of stream = %v, want io.ErrUnexpectedEOF", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(got), got)
	}
	if got[0].Type != "task.moved" || got[0].ActorID != 2 || got[0].Data["task_id"] != float64(6) {
		t.Errorf("first event = %+v", got[0])
	}
	if got[1].Type != "column.created" || got[1].Data["name"] != "Review" {
		t.Errorf("second event = %+v", got[1])
	}
	if lines != 12 {
		t.Errorf("alive called %d times, want once per line (12)", lines)
	}
}
