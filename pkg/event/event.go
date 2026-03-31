package event

import (
	"context"
	"time"
)

type Event interface {
	EventType() string
	Payload() any
	OccurredAt() time.Time
}

type EventHandler interface {
	Handle(ctx context.Context, event Event) error
}

type EventDispatcher interface {
	Register(eventType string, handler EventHandler)
	Dispatch(ctx context.Context, event Event) error
	Remove(eventType string, handler EventHandler) error
	Has(eventType string, handler EventHandler) bool
	Clear()
}
