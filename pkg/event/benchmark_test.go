package event_test

import (
	"context"
	"testing"

	"devkit/pkg/event"
)

func BenchmarkDispatch(b *testing.B) {
	d := event.NewDispatcher()
	h := &noopHandler{}
	d.Register("evt", h)
	e := &mockEvent{eventType: "evt"}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := d.Dispatch(ctx, e); err != nil {
			b.Fatalf("dispatch failed: %v", err)
		}
	}
}

func BenchmarkDispatchParallel(b *testing.B) {
	d := event.NewDispatcher()
	h := &noopHandler{}
	d.Register("evt", h)
	e := &mockEvent{eventType: "evt"}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := d.Dispatch(ctx, e); err != nil {
				b.Fatalf("dispatch failed: %v", err)
			}
		}
	})
}

type noopHandler struct{}

func (h *noopHandler) Handle(_ context.Context, _ event.Event) error { return nil }
