package event_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"devkit/pkg/event"
)

type userCreatedEvent struct {
	userID string
}

func (e *userCreatedEvent) EventType() string  { return "user.created" }
func (e *userCreatedEvent) Payload() any       { return e.userID }
func (e *userCreatedEvent) OccurredAt() time.Time { return time.Now() }

type welcomeEmailHandler struct{}

func (h *welcomeEmailHandler) Handle(_ context.Context, e event.Event) error {
	fmt.Printf("sending welcome email for user %v\n", e.Payload())
	return nil
}

func ExampleDispatcher() {
	d := event.NewDispatcher()
	d.Register("user.created", &welcomeEmailHandler{})

	evt := &userCreatedEvent{userID: "u-123"}
	if err := d.Dispatch(context.Background(), evt); err != nil {
		fmt.Println("dispatch error:", err)
	}
}

func ExampleDispatcher_panicRecovery() {
	d := event.NewDispatcher()
	d.Register("evt", &panicHandler{value: "something went wrong"})

	err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"})
	var pe *event.PanicError
	if errors.As(err, &pe) {
		fmt.Println("panic recovered:", pe.Value)
	}
}
