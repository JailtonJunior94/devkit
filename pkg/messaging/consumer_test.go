package messaging_test

import (
	"context"
	"errors"
	"testing"

	"devkit/pkg/messaging"
)

func TestHandlerFunc_Handle_success(t *testing.T) {
	called := false
	var gotMsg messaging.Message

	fn := messaging.HandlerFunc(func(_ context.Context, msg messaging.Message) error {
		called = true
		gotMsg = msg
		return nil
	})

	msg := messaging.Message{
		EventType: "order.created",
		Headers:   map[string]string{"x-trace": "abc"},
		Body:      []byte(`{"id":1}`),
		Topic:     "orders",
		Partition: 0,
		Offset:    42,
	}

	if err := fn.Handle(context.Background(), msg); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !called {
		t.Fatal("handler func was not called")
	}
	if gotMsg.EventType != msg.EventType {
		t.Errorf("EventType = %q, want %q", gotMsg.EventType, msg.EventType)
	}
}

func TestHandlerFunc_Handle_propagatesError(t *testing.T) {
	wantErr := errors.New("processing failed")

	fn := messaging.HandlerFunc(func(_ context.Context, _ messaging.Message) error {
		return wantErr
	})

	err := fn.Handle(context.Background(), messaging.Message{})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped error %v, got %v", wantErr, err)
	}
}

var _ messaging.Handler = messaging.HandlerFunc(nil)
