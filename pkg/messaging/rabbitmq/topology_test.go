package rabbitmq

import (
	"testing"
	"time"
)

func TestRetryQueueName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		queue string
		want  string
	}{
		{"orders", "orders.retry"},
		{"payments.processed", "payments.processed.retry"},
		{"a", "a.retry"},
	}

	for _, tc := range cases {
		got := RetryQueueName(tc.queue)
		if got != tc.want {
			t.Errorf("RetryQueueName(%q) = %q, want %q", tc.queue, got, tc.want)
		}
	}
}

func TestDLQName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		queue string
		want  string
	}{
		{"orders", "orders.dlq"},
		{"payments.processed", "payments.processed.dlq"},
		{"a", "a.dlq"},
	}

	for _, tc := range cases {
		got := DLQName(tc.queue)
		if got != tc.want {
			t.Errorf("DLQName(%q) = %q, want %q", tc.queue, got, tc.want)
		}
	}
}

func TestValidateExchangeKind(t *testing.T) {
	t.Parallel()

	valid := []string{"direct", "topic", "fanout", "headers"}
	for _, kind := range valid {
		if err := validateExchangeKind(kind); err != nil {
			t.Errorf("validateExchangeKind(%q) unexpected error: %v", kind, err)
		}
	}

	invalid := []string{"", "unknown", "DIRECT", "Topic"}
	for _, kind := range invalid {
		if err := validateExchangeKind(kind); err == nil {
			t.Errorf("validateExchangeKind(%q) expected error, got nil", kind)
		}
	}
}

func TestTopologyBuilder_Options(t *testing.T) {
	t.Parallel()

	b := NewTopologyBuilder(
		WithExchangeDecl(&Exchange{Name: "events", Kind: "topic", Durable: true}),
		WithQueueDecl(&QueueDecl{Name: "orders", Durable: true}),
		WithBindingDecl(&Binding{Queue: "orders", Exchange: "events", RoutingKey: "order.*"}),
		WithRetryQueue("orders", "events", 5*time.Second),
		WithDLQQueue("orders"),
		WithTopologyPrefetch(10),
	)

	if len(b.exchanges) != 1 {
		t.Errorf("expected 1 exchange, got %d", len(b.exchanges))
	}
	if b.exchanges[0].Name != "events" {
		t.Errorf("unexpected exchange name: %s", b.exchanges[0].Name)
	}
	if len(b.queues) != 1 {
		t.Errorf("expected 1 queue, got %d", len(b.queues))
	}
	if len(b.bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(b.bindings))
	}
	if len(b.retryQueues) != 1 {
		t.Errorf("expected 1 retry queue, got %d", len(b.retryQueues))
	}
	if b.retryQueues[0].ttl != 5*time.Second {
		t.Errorf("unexpected retry ttl: %v", b.retryQueues[0].ttl)
	}
	if len(b.dlqQueues) != 1 {
		t.Errorf("expected 1 dlq queue, got %d", len(b.dlqQueues))
	}
	if b.prefetchCount != 10 {
		t.Errorf("unexpected prefetch: %d", b.prefetchCount)
	}
}

func TestTopologyBuilder_EmptyBuilderIsValid(t *testing.T) {
	t.Parallel()

	b := NewTopologyBuilder()
	if b == nil {
		t.Fatal("expected non-nil builder")
	}
	if b.prefetchCount != 0 {
		t.Errorf("expected 0 prefetch, got %d", b.prefetchCount)
	}
}
