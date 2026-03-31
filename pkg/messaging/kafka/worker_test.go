package kafka_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"devkit/pkg/messaging"
	"devkit/pkg/messaging/kafka"
)

func TestWorkerPool_sequential(t *testing.T) {
	pool := kafka.NewWorkerPoolForTest(1)

	var (
		mu      sync.Mutex
		results []int
	)

	pool.Start(context.Background(), func(_ context.Context, job kafka.WorkerJobForTest) {
		mu.Lock()
		results = append(results, job.N)
		mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	})

	for i := 0; i < 5; i++ {
		pool.DispatchForTest(kafka.WorkerJobForTest{N: i})
	}
	pool.Stop()

	mu.Lock()
	defer mu.Unlock()
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
}

func TestWorkerPool_parallel(t *testing.T) {
	const numWorkers = 4
	pool := kafka.NewWorkerPoolForTest(numWorkers)

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	pool.Start(context.Background(), func(_ context.Context, _ kafka.WorkerJobForTest) {
		n := concurrent.Add(1)
		for {
			cur := maxConcurrent.Load()
			if n <= cur || maxConcurrent.CompareAndSwap(cur, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		concurrent.Add(-1)
	})

	for i := 0; i < numWorkers*2; i++ {
		pool.DispatchForTest(kafka.WorkerJobForTest{N: i})
	}
	pool.Stop()

	if maxConcurrent.Load() < 2 {
		t.Errorf("expected concurrency ≥2, got %d", maxConcurrent.Load())
	}
}

func TestWorkerPool_stopDrainsJobs(t *testing.T) {
	pool := kafka.NewWorkerPoolForTest(2)

	var count atomic.Int32
	pool.Start(context.Background(), func(_ context.Context, _ kafka.WorkerJobForTest) {
		count.Add(1)
	})

	const total = 20
	for i := 0; i < total; i++ {
		pool.DispatchForTest(kafka.WorkerJobForTest{N: i})
	}
	pool.Stop()

	if got := count.Load(); got != total {
		t.Errorf("expected %d processed, got %d", total, got)
	}
}

func TestWorkerPool_normalizeMessage(t *testing.T) {
	tests := []struct {
		name            string
		headers         []kafkago.Header
		topic           string
		eventTypeHeader string
		wantEventType   string
	}{
		{
			name:            "header present",
			headers:         []kafkago.Header{{Key: "event_type", Value: []byte("order.created")}},
			topic:           "orders",
			eventTypeHeader: "event_type",
			wantEventType:   "order.created",
		},
		{
			name:            "header absent fallback to topic",
			headers:         nil,
			topic:           "orders",
			eventTypeHeader: "event_type",
			wantEventType:   "orders",
		},
		{
			name:            "custom header key",
			headers:         []kafkago.Header{{Key: "x-event", Value: []byte("payment.processed")}},
			topic:           "payments",
			eventTypeHeader: "x-event",
			wantEventType:   "payment.processed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := kafkago.Message{
				Topic:   tt.topic,
				Headers: tt.headers,
				Value:   []byte("body"),
			}
			got := kafka.NormalizeMessage(raw, tt.eventTypeHeader)
			if got.EventType != tt.wantEventType {
				t.Errorf("EventType = %q, want %q", got.EventType, tt.wantEventType)
			}
			if got.Topic != tt.topic {
				t.Errorf("Topic = %q, want %q", got.Topic, tt.topic)
			}
		})
	}
}

func TestHandlerFuncAdapter(t *testing.T) {
	called := false
	fn := messaging.HandlerFunc(func(_ context.Context, _ messaging.Message) error {
		called = true
		return nil
	})
	_ = fn.Handle(context.Background(), messaging.Message{})
	if !called {
		t.Error("HandlerFunc not called")
	}
}
