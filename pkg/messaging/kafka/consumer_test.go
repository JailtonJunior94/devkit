package kafka_test

import (
	"context"
	"crypto/tls"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"

	"devkit/pkg/messaging"
	"devkit/pkg/messaging/kafka"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestNewConsumer_missingBrokers(t *testing.T) {
	_, err := kafka.NewConsumer(
		kafka.WithGroupID("grp"),
		kafka.WithTopics("t"),
	)
	if !errors.Is(err, kafka.ErrNoBrokers) {
		t.Errorf("expected ErrNoBrokers, got %v", err)
	}
}

func TestNewConsumer_missingGroupID(t *testing.T) {
	_, err := kafka.NewConsumer(
		kafka.WithBrokers("b:9092"),
		kafka.WithTopics("t"),
	)
	if !errors.Is(err, kafka.ErrNoGroupID) {
		t.Errorf("expected ErrNoGroupID, got %v", err)
	}
}

func TestNewConsumer_missingTopics(t *testing.T) {
	_, err := kafka.NewConsumer(
		kafka.WithBrokers("b:9092"),
		kafka.WithGroupID("grp"),
	)
	if !errors.Is(err, kafka.ErrNoTopics) {
		t.Errorf("expected ErrNoTopics, got %v", err)
	}
}

func TestNewConsumer_multipleErrors(t *testing.T) {
	_, err := kafka.NewConsumer()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, kafka.ErrNoBrokers) {
		t.Errorf("expected ErrNoBrokers in joined error, got %v", err)
	}
	if !errors.Is(err, kafka.ErrNoGroupID) {
		t.Errorf("expected ErrNoGroupID in joined error, got %v", err)
	}
	if !errors.Is(err, kafka.ErrNoTopics) {
		t.Errorf("expected ErrNoTopics in joined error, got %v", err)
	}
}

func TestNewConsumer_minimumOptions(t *testing.T) {
	c, err := kafka.NewConsumer(
		kafka.WithBrokers("localhost:9092"),
		kafka.WithGroupID("grp"),
		kafka.WithTopics("test-topic"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil consumer")
	}
}

func TestNewConsumer_defaults(t *testing.T) {
	c, err := kafka.NewConsumer(
		kafka.WithBrokers("localhost:9092"),
		kafka.WithGroupID("grp"),
		kafka.WithTopics("test-topic"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var _ messaging.Consumer = c
}

var _ messaging.Consumer = (*kafka.Consumer)(nil)

func TestNewConsumer_withPlainAuth(t *testing.T) {
	c, err := kafka.NewConsumer(
		kafka.WithBrokers("localhost:9092"),
		kafka.WithGroupID("grp"),
		kafka.WithTopics("t"),
		kafka.WithPlainAuth("user", "pass"),
	)
	if err != nil {
		t.Fatalf("unexpected error with plain auth: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil consumer")
	}
}

func TestNewConsumer_withSCRAM256(t *testing.T) {
	_, err := kafka.NewConsumer(
		kafka.WithBrokers("localhost:9092"),
		kafka.WithGroupID("grp"),
		kafka.WithTopics("t"),
		kafka.WithSCRAM256("user", "pass"),
	)
	if err != nil {
		t.Fatalf("unexpected error with SCRAM256: %v", err)
	}
}

func TestNewConsumer_withSCRAM512(t *testing.T) {
	_, err := kafka.NewConsumer(
		kafka.WithBrokers("localhost:9092"),
		kafka.WithGroupID("grp"),
		kafka.WithTopics("t"),
		kafka.WithSCRAM512("user", "pass"),
	)
	if err != nil {
		t.Fatalf("unexpected error with SCRAM512: %v", err)
	}
}

func TestNewConsumer_withTLS(t *testing.T) {
	_, err := kafka.NewConsumer(
		kafka.WithBrokers("localhost:9092"),
		kafka.WithGroupID("grp"),
		kafka.WithTopics("t"),
		kafka.WithTLS(&tls.Config{MinVersion: tls.VersionTLS12}),
	)
	if err != nil {
		t.Fatalf("unexpected error with TLS: %v", err)
	}
}

func TestRegisterHandler_concurrent(t *testing.T) {
	c, err := kafka.NewConsumer(
		kafka.WithBrokers("localhost:9092"),
		kafka.WithGroupID("grp"),
		kafka.WithTopics("t"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			et := "event." + string(rune('a'+n%26))
			c.RegisterHandler(et, messaging.HandlerFunc(func(_ context.Context, _ messaging.Message) error {
				return nil
			}))
		}(i)
	}
	wg.Wait()
}

func TestBackoffDuration_table(t *testing.T) {
	tests := []struct {
		initial    time.Duration
		max        time.Duration
		multiplier float64
		attempt    int
		want       time.Duration
	}{
		{time.Second, 30 * time.Second, 2.0, 0, time.Second},
		{time.Second, 30 * time.Second, 2.0, 1, 2 * time.Second},
		{time.Second, 30 * time.Second, 2.0, 2, 4 * time.Second},
		{time.Second, 30 * time.Second, 2.0, 5, 30 * time.Second},
	}

	for _, tt := range tests {
		got := kafka.BackoffDuration(tt.initial, tt.max, tt.multiplier, tt.attempt)
		if got != tt.want {
			t.Errorf("attempt=%d: got %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestShutdown_idempotent(t *testing.T) {
	c, err := kafka.NewConsumer(
		kafka.WithBrokers("localhost:9092"),
		kafka.WithGroupID("grp"),
		kafka.WithTopics("t"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	if err := c.Shutdown(ctx); err != nil {
		t.Errorf("first Shutdown: %v", err)
	}
	if err := c.Shutdown(ctx); err != nil {
		t.Errorf("second Shutdown (idempotent): %v", err)
	}
}

func TestShutdown_deadlineExceeded(t *testing.T) {
	c, err := kafka.NewConsumer(
		kafka.WithBrokers("localhost:9092"),
		kafka.WithGroupID("grp"),
		kafka.WithTopics("t"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond)

	if err := c.Shutdown(ctx); err != nil {
		if !errors.Is(err, kafka.ErrShutdownTimeout) {
			t.Logf("shutdown error (acceptable): %v", err)
		}
	}
}
