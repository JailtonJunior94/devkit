package event

import (
	"context"
	"testing"
)

type internalHandler struct{}

func (h *internalHandler) Handle(_ context.Context, _ Event) error { return nil }

func TestRemoveReleasesLastSlotReference(t *testing.T) {
	d := NewDispatcher()
	h1 := &internalHandler{}
	h2 := &internalHandler{}

	d.Register("evt", h1)
	d.Register("evt", h2)

	handlersBefore := d.handlers["evt"]
	if len(handlersBefore) != 2 {
		t.Fatalf("expected 2 handlers before remove, got %d", len(handlersBefore))
	}

	if err := d.Remove("evt", h1); err != nil {
		t.Fatalf("unexpected remove error: %v", err)
	}

	handlersAfter := d.handlers["evt"]
	if len(handlersAfter) != 1 {
		t.Fatalf("expected 1 handler after remove, got %d", len(handlersAfter))
	}
	if handlersAfter[0] != h2 {
		t.Fatal("expected remaining handler to be preserved")
	}
	if handlersBefore[1] != nil {
		t.Fatal("expected last slot to be cleared after remove")
	}
}

func TestClearReplacesHandlerMap(t *testing.T) {
	d := NewDispatcher()
	d.Register("evt", &internalHandler{})

	oldMap := d.handlers
	d.Clear()

	if len(d.handlers) != 0 {
		t.Fatalf("expected empty handlers map after clear, got %d entries", len(d.handlers))
	}
	if len(oldMap) != 1 {
		t.Fatalf("expected original map snapshot to retain prior state, got %d entries", len(oldMap))
	}
	delete(oldMap, "evt")
	if len(d.handlers) != 0 {
		t.Fatal("expected Clear to replace the handler map with an independent map")
	}
}
