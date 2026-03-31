//go:build integration

package kafka_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"

	"devkit/pkg/messaging"
	"devkit/pkg/messaging/kafka"
)

func setupKafka(t *testing.T) (string, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.5.0")
	if err != nil {
		t.Fatalf("start kafka container: %v", err)
	}

	broker, err := container.Brokers(ctx)
	if err != nil {
		t.Fatalf("get kafka brokers: %v", err)
	}

	return broker[0], func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminate kafka container: %v", err)
		}
	}
}

func produceMessage(t *testing.T, broker, topic, eventType string, body []byte) {
	t.Helper()
	w := kafkago.NewWriter(kafkago.WriterConfig{
		Brokers: []string{broker},
		Topic:   topic,
	})
	defer w.Close()

	msg := kafkago.Message{
		Value: body,
		Headers: []kafkago.Header{
			{Key: "event_type", Value: []byte(eventType)},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := w.WriteMessages(ctx, msg); err != nil {
		t.Fatalf("write message: %v", err)
	}
}

func TestIntegration_ConsumeAndCommit(t *testing.T) {
	broker, cleanup := setupKafka(t)
	defer cleanup()

	topic := "integration-test-consume"
	produceMessage(t, broker, topic, "test.event", []byte(`{"id":1}`))

	received := make(chan messaging.Message, 1)
	c, err := kafka.NewConsumer(
		kafka.WithBrokers(broker),
		kafka.WithGroupID("integration-grp"),
		kafka.WithTopics(topic),
	)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}

	c.RegisterHandler("test.event", messaging.HandlerFunc(func(_ context.Context, msg messaging.Message) error {
		received <- msg
		return nil
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() { _ = c.Consume(ctx) }()

	select {
	case msg := <-received:
		if msg.EventType != "test.event" {
			t.Errorf("EventType = %q, want %q", msg.EventType, "test.event")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for message")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := c.Shutdown(shutdownCtx); err != nil {
		t.Logf("shutdown: %v", err)
	}
}

func TestIntegration_DLQFlow(t *testing.T) {
	broker, cleanup := setupKafka(t)
	defer cleanup()

	topic := "integration-test-dlq"
	dlqTopic := topic + "-dlq"
	produceMessage(t, broker, topic, "fail.event", []byte(`{"fail":true}`))

	c, err := kafka.NewConsumer(
		kafka.WithBrokers(broker),
		kafka.WithGroupID("integration-dlq-grp"),
		kafka.WithTopics(topic),
		kafka.WithMaxRetries(2),
		kafka.WithBackoff(10*time.Millisecond, 50*time.Millisecond, 2.0),
		kafka.WithDLQ(true),
	)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}

	c.RegisterHandler("fail.event", messaging.HandlerFunc(func(_ context.Context, _ messaging.Message) error {
		return errors.New("intentional failure")
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() { _ = c.Consume(ctx) }()

	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    dlqTopic,
		GroupID:  "integration-dlq-checker",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer r.Close()

	dlqCtx, dlqCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dlqCancel()

	dlqMsg, err := r.FetchMessage(dlqCtx)
	if err != nil {
		t.Fatalf("fetch DLQ message: %v", err)
	}

	headers := make(map[string]string)
	for _, h := range dlqMsg.Headers {
		headers[h.Key] = string(h.Value)
	}
	for _, key := range []string{"error", "event_type", "origin_topic", "timestamp"} {
		if headers[key] == "" {
			t.Errorf("DLQ header %q missing", key)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = c.Shutdown(shutdownCtx)
}

func TestIntegration_GracefulShutdown(t *testing.T) {
	broker, cleanup := setupKafka(t)
	defer cleanup()

	topic := "integration-test-shutdown"
	produceMessage(t, broker, topic, "slow.event", []byte(`{}`))

	var completed atomic.Bool
	c, err := kafka.NewConsumer(
		kafka.WithBrokers(broker),
		kafka.WithGroupID("integration-shutdown-grp"),
		kafka.WithTopics(topic),
	)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}

	started := make(chan struct{})
	c.RegisterHandler("slow.event", messaging.HandlerFunc(func(_ context.Context, _ messaging.Message) error {
		close(started)
		time.Sleep(200 * time.Millisecond)
		completed.Store(true)
		return nil
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	go func() { _ = c.Consume(ctx) }()

	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("timed out waiting for handler to start")
	}

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = c.Shutdown(shutdownCtx)

	if !completed.Load() {
		t.Error("handler did not complete before shutdown returned")
	}
}

func TestIntegration_WorkerPoolThroughput(t *testing.T) {
	broker, cleanup := setupKafka(t)
	defer cleanup()

	topic := "integration-test-parallel"
	const numMessages = 8
	for i := 0; i < numMessages; i++ {
		produceMessage(t, broker, topic, "parallel.event", []byte(`{}`))
	}

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	c, err := kafka.NewConsumer(
		kafka.WithBrokers(broker),
		kafka.WithGroupID("integration-parallel-grp"),
		kafka.WithTopics(topic),
		kafka.WithWorkers(topic, 4),
	)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}

	var processed atomic.Int32
	done := make(chan struct{})

	c.RegisterHandler("parallel.event", messaging.HandlerFunc(func(_ context.Context, _ messaging.Message) error {
		n := concurrent.Add(1)
		for {
			cur := maxConcurrent.Load()
			if n <= cur || maxConcurrent.CompareAndSwap(cur, n) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		concurrent.Add(-1)
		if processed.Add(1) == numMessages {
			close(done)
		}
		return nil
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	go func() { _ = c.Consume(ctx) }()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timed out waiting for all messages")
	}

	if maxConcurrent.Load() < 2 {
		t.Errorf("expected concurrency ≥2, got %d", maxConcurrent.Load())
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = c.Shutdown(shutdownCtx)
}
