package event

import (
	"context"
	"errors"
	"runtime"
	"sync"
)

type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string][]EventHandler),
	}
}

func handlersEqual(a, b EventHandler) (equal bool) {
	defer func() {
		if recover() != nil {
			equal = false
		}
	}()
	return a == b
}

func (d *Dispatcher) Register(eventType string, handler EventHandler) {
	if eventType == "" || handler == nil {
		return
	}
	d.mu.Lock()
	d.handlers[eventType] = append(d.handlers[eventType], handler)
	d.mu.Unlock()
}

func (d *Dispatcher) Has(eventType string, handler EventHandler) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, h := range d.handlers[eventType] {
		if handlersEqual(h, handler) {
			return true
		}
	}
	return false
}

func (d *Dispatcher) Clear() {
	d.mu.Lock()
	d.handlers = make(map[string][]EventHandler)
	d.mu.Unlock()
}

func (d *Dispatcher) Remove(eventType string, handler EventHandler) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	handlers, ok := d.handlers[eventType]
	if !ok || len(handlers) == 0 {
		return ErrHandlerNotFound
	}

	for i, h := range handlers {
		if handlersEqual(h, handler) {
			copy(handlers[i:], handlers[i+1:])
			handlers[len(handlers)-1] = nil
			d.handlers[eventType] = handlers[:len(handlers)-1]
			return nil
		}
	}
	return ErrHandlerNotFound
}

func (d *Dispatcher) Dispatch(ctx context.Context, event Event) error {
	if ctx == nil {
		return ErrNilContext
	}
	if event == nil {
		return ErrNilEvent
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	d.mu.RLock()
	src := d.handlers[event.EventType()]
	if len(src) == 0 {
		d.mu.RUnlock()
		return nil
	}
	snapshot := make([]EventHandler, len(src))
	copy(snapshot, src)
	d.mu.RUnlock()

	var errs []error
	for _, h := range snapshot {
		if err := safeHandle(ctx, h, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func safeHandle(ctx context.Context, h EventHandler, event Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			err = &PanicError{Value: r, Stack: buf[:n]}
		}
	}()
	return h.Handle(ctx, event)
}
