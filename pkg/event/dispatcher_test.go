package event_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"

	"devkit/pkg/event"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type mockEvent struct {
	eventType string
	payload   any
}

func (e *mockEvent) EventType() string     { return e.eventType }
func (e *mockEvent) Payload() any          { return e.payload }
func (e *mockEvent) OccurredAt() time.Time { return time.Time{} }

type callHandler struct {
	t      *testing.T
	mu     sync.Mutex
	calls  []event.Event
	retErr error
}

func newCallHandler(t *testing.T) *callHandler {
	t.Helper()
	return &callHandler{t: t}
}

func nilContext() context.Context {
	return nil
}

func (h *callHandler) Handle(_ context.Context, e event.Event) error {
	h.mu.Lock()
	h.calls = append(h.calls, e)
	h.mu.Unlock()
	return h.retErr
}

func (h *callHandler) callCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.calls)
}

type panicHandler struct {
	value any
}

func (h *panicHandler) Handle(_ context.Context, _ event.Event) error {
	panic(h.value)
}

func TestRegister(t *testing.T) {
	t.Run("empty event type is no-op", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("", h)
		if err := d.Dispatch(context.Background(), &mockEvent{eventType: ""}); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if h.callCount() != 0 {
			t.Fatal("handler should not be called for empty event type")
		}
	})

	t.Run("nil handler is no-op", func(t *testing.T) {
		d := event.NewDispatcher()
		d.Register("evt", nil)
		if d.Has("evt", nil) {
			t.Fatal("nil handler should not be registered")
		}
	})

	t.Run("multiple handlers for same event type", func(t *testing.T) {
		d := event.NewDispatcher()
		h1 := newCallHandler(t)
		h2 := newCallHandler(t)
		d.Register("evt", h1)
		d.Register("evt", h2)
		if err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"}); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if h1.callCount() != 1 || h2.callCount() != 1 {
			t.Fatal("both handlers should be invoked")
		}
	})

	t.Run("duplicate handler registered twice invoked twice", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("evt", h)
		d.Register("evt", h)
		if err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"}); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if h.callCount() != 2 {
			t.Fatalf("expected 2 invocations, got %d", h.callCount())
		}
	})
}

func TestDispatch(t *testing.T) {
	t.Run("nominal invokes handler and returns nil", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("evt", h)
		if err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"}); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if h.callCount() != 1 {
			t.Fatal("handler not invoked")
		}
	})

	t.Run("no handlers returns nil", func(t *testing.T) {
		d := event.NewDispatcher()
		if err := d.Dispatch(context.Background(), &mockEvent{eventType: "nothing"}); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("nil context returns ErrNilContext without invoking handlers", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("evt", h)
		err := d.Dispatch(nilContext(), &mockEvent{eventType: "evt"})
		if !errors.Is(err, event.ErrNilContext) {
			t.Fatalf("expected ErrNilContext, got %v", err)
		}
		if h.callCount() != 0 {
			t.Fatal("handler should not be called with nil context")
		}
	})

	t.Run("nil event returns ErrNilEvent without invoking handlers", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("evt", h)
		err := d.Dispatch(context.Background(), nil)
		if !errors.Is(err, event.ErrNilEvent) {
			t.Fatalf("expected ErrNilEvent, got %v", err)
		}
		if h.callCount() != 0 {
			t.Fatal("handler should not be called with nil event")
		}
	})

	t.Run("cancelled context returns error without invoking handlers", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("evt", h)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := d.Dispatch(ctx, &mockEvent{eventType: "evt"}); !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
		if h.callCount() != 0 {
			t.Fatal("handler should not be called with cancelled context")
		}
	})

	t.Run("handler returning error propagates it", func(t *testing.T) {
		d := event.NewDispatcher()
		sentinel := errors.New("handler error")
		h := newCallHandler(t)
		h.retErr = sentinel
		d.Register("evt", h)
		err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"})
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected sentinel error, got %v", err)
		}
	})

	t.Run("panic handler wrapped as PanicError", func(t *testing.T) {
		d := event.NewDispatcher()
		d.Register("evt", &panicHandler{value: "boom"})
		err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"})
		if err == nil {
			t.Fatal("expected error from panic, got nil")
		}
		var pe *event.PanicError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PanicError, got %T: %v", err, err)
		}
		if pe.Value != "boom" {
			t.Fatalf("unexpected panic value: %v", pe.Value)
		}
	})

	t.Run("multiple handlers all invoked, errors aggregated", func(t *testing.T) {
		d := event.NewDispatcher()
		err1 := errors.New("e1")
		err2 := errors.New("e2")
		h1 := newCallHandler(t)
		h1.retErr = err1
		h2 := newCallHandler(t)
		h2.retErr = err2
		d.Register("evt", h1)
		d.Register("evt", h2)
		err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"})
		if !errors.Is(err, err1) || !errors.Is(err, err2) {
			t.Fatalf("expected both errors in aggregate, got %v", err)
		}
		if h1.callCount() != 1 || h2.callCount() != 1 {
			t.Fatal("both handlers must be invoked")
		}
	})

	t.Run("errors.As finds PanicError in aggregated error", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		h.retErr = errors.New("plain error")
		d.Register("evt", h)
		d.Register("evt", &panicHandler{value: 42})
		err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"})
		var pe *event.PanicError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PanicError via errors.As, got %v", err)
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("removed handler not invoked", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("evt", h)
		if err := d.Remove("evt", h); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"}); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if h.callCount() != 0 {
			t.Fatal("removed handler should not be invoked")
		}
	})

	t.Run("preserves order after removal", func(t *testing.T) {
		d := event.NewDispatcher()
		var order []string
		var mu sync.Mutex
		makeRecorder := func(id string) event.EventHandler {
			return &recorderHandler{id: id, order: &order, mu: &mu}
		}

		hA := makeRecorder("A")
		hB := makeRecorder("B")
		hC := makeRecorder("C")
		d.Register("evt", hA)
		d.Register("evt", hB)
		d.Register("evt", hC)
		if err := d.Remove("evt", hB); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"}); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		mu.Lock()
		got := append([]string(nil), order...)
		mu.Unlock()
		if len(got) != 2 || got[0] != "A" || got[1] != "C" {
			t.Fatalf("expected order [A C], got %v", order)
		}
	})

	t.Run("handler not found returns ErrHandlerNotFound", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("evt", h)
		other := newCallHandler(t)
		err := d.Remove("evt", other)
		if !errors.Is(err, event.ErrHandlerNotFound) {
			t.Fatalf("expected ErrHandlerNotFound, got %v", err)
		}
	})

	t.Run("event type not found returns ErrHandlerNotFound", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		err := d.Remove("nonexistent", h)
		if !errors.Is(err, event.ErrHandlerNotFound) {
			t.Fatalf("expected ErrHandlerNotFound, got %v", err)
		}
	})
}

type recorderHandler struct {
	id    string
	order *[]string
	mu    *sync.Mutex
}

func (h *recorderHandler) Handle(_ context.Context, _ event.Event) error {
	h.mu.Lock()
	*h.order = append(*h.order, h.id)
	h.mu.Unlock()
	return nil
}

func TestHas(t *testing.T) {
	t.Run("registered handler returns true", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("evt", h)
		if !d.Has("evt", h) {
			t.Fatal("expected Has to return true")
		}
	})

	t.Run("unregistered handler returns false", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		if d.Has("evt", h) {
			t.Fatal("expected Has to return false")
		}
	})

	t.Run("after Remove returns false", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("evt", h)
		if err := d.Remove("evt", h); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.Has("evt", h) {
			t.Fatal("expected Has to return false after Remove")
		}
	})
}

func TestClear(t *testing.T) {
	t.Run("dispatch after Clear does not invoke handlers", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("evt", h)
		d.Clear()
		if err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"}); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if h.callCount() != 0 {
			t.Fatal("handler should not be invoked after Clear")
		}
	})

	t.Run("Has returns false after Clear", func(t *testing.T) {
		d := event.NewDispatcher()
		h := newCallHandler(t)
		d.Register("evt", h)
		d.Clear()
		if d.Has("evt", h) {
			t.Fatal("expected Has to return false after Clear")
		}
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("Register and Dispatch concurrent", func(t *testing.T) {
		d := event.NewDispatcher()
		var wg sync.WaitGroup
		const goroutines = 50
		for i := 0; i < goroutines; i++ {
			wg.Add(2)
			go func(i int) {
				defer wg.Done()
				h := newCallHandler(t)
				d.Register(fmt.Sprintf("evt%d", i%5), h)
			}(i)
			go func(i int) {
				defer wg.Done()
				err := d.Dispatch(context.Background(), &mockEvent{eventType: fmt.Sprintf("evt%d", i%5)})
				if err != nil {
					t.Errorf("dispatch failed: %v", err)
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("Dispatch and Remove concurrent", func(t *testing.T) {
		d := event.NewDispatcher()
		handlers := make([]*callHandler, 20)
		for i := range handlers {
			handlers[i] = newCallHandler(t)
			d.Register("evt", handlers[i])
		}
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(2)
			go func(h *callHandler) {
				defer wg.Done()
				err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"})
				if err != nil {
					t.Errorf("dispatch failed: %v", err)
				}
			}(handlers[i])
			go func(h *callHandler) {
				defer wg.Done()
				_ = d.Remove("evt", h)
			}(handlers[i])
		}
		wg.Wait()
	})

	t.Run("Dispatch and Clear concurrent", func(t *testing.T) {
		d := event.NewDispatcher()
		for i := 0; i < 10; i++ {
			d.Register("evt", newCallHandler(t))
		}
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				err := d.Dispatch(context.Background(), &mockEvent{eventType: "evt"})
				if err != nil {
					t.Errorf("dispatch failed: %v", err)
				}
			}()
			go func() {
				defer wg.Done()
				d.Clear()
			}()
		}
		wg.Wait()
	})
}

func TestPanicError(t *testing.T) {
	t.Run("Error formats correctly", func(t *testing.T) {
		pe := &event.PanicError{Value: "test panic", Stack: []byte("stack")}
		got := pe.Error()
		expected := "panic recovered during handler execution"
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})
}

func TestDispatchAllocsPerRun(t *testing.T) {
	d := event.NewDispatcher()
	var counter atomic.Int64
	h := &countHandler{counter: &counter}
	d.Register("evt", h)
	e := &mockEvent{eventType: "evt"}
	ctx := context.Background()

	allocs := testing.AllocsPerRun(100, func() {
		if err := d.Dispatch(ctx, e); err != nil {
			t.Fatalf("dispatch failed: %v", err)
		}
	})
	if allocs > 2 {
		t.Fatalf("too many allocs per dispatch: %.1f (want <= 2)", allocs)
	}
	if got := counter.Load(); got < 100 {
		t.Fatalf("expected handler to run at least 100 times, got %d", got)
	}
}

type countHandler struct {
	counter *atomic.Int64
}

func (h *countHandler) Handle(_ context.Context, _ event.Event) error {
	h.counter.Add(1)
	return nil
}
