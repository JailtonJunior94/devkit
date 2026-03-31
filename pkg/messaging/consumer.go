package messaging

import "context"

type Message struct {
	EventType string
	Headers   map[string]string
	Body      []byte
	Topic     string
	Partition int
	Offset    int64
}

type Handler interface {
	Handle(ctx context.Context, msg Message) error
}

type HandlerFunc func(ctx context.Context, msg Message) error

func (f HandlerFunc) Handle(ctx context.Context, msg Message) error {
	return f(ctx, msg)
}

type Consumer interface {
	RegisterHandler(eventType string, handler Handler)
	Consume(ctx context.Context) error
	Shutdown(ctx context.Context) error
}
