package app

import "context"

// Handler receives boards' domain events (domain.TaskCreated,
// domain.TaskMoved, ...) after the change that caused them is stored. It
// cannot fail the change: a handler that cannot do its job reports that
// itself.
type Handler func(ctx context.Context, event any)

// Dispatcher hands published events to every handler, in order, in process
// and synchronously.
type Dispatcher struct {
	handlers []Handler
}

func NewDispatcher(handlers ...Handler) *Dispatcher {
	return &Dispatcher{handlers: handlers}
}

func (d *Dispatcher) Publish(ctx context.Context, events ...any) {
	for _, ev := range events {
		for _, h := range d.handlers {
			h(ctx, ev)
		}
	}
}
