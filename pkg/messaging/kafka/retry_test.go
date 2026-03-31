package kafka_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"devkit/pkg/messaging"
	"devkit/pkg/messaging/kafka"
)

type countingHandler struct {
	calls   int
	failFor int
	err     error
}

func (h *countingHandler) Handle(_ context.Context, _ messaging.Message) error {
	h.calls++
	if h.calls <= h.failFor {
		return h.err
	}
	return nil
}

func TestRetry_successOnFirstAttempt(t *testing.T) {
	h := &countingHandler{failFor: 0}
	err := kafka.RetryForTest(context.Background(), h, messaging.Message{Topic: "t"}, kafka.RetryConfigForTest{
		MaxRetries:  3,
		BackoffInit: time.Millisecond,
		BackoffMax:  10 * time.Millisecond,
		BackoffMul:  2.0,
	})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if h.calls != 1 {
		t.Errorf("expected 1 call, got %d", h.calls)
	}
}

func TestRetry_successAfterRetry(t *testing.T) {
	wantErr := errors.New("transient")
	h := &countingHandler{failFor: 2, err: wantErr}
	err := kafka.RetryForTest(context.Background(), h, messaging.Message{Topic: "t"}, kafka.RetryConfigForTest{
		MaxRetries:  3,
		BackoffInit: time.Millisecond,
		BackoffMax:  10 * time.Millisecond,
		BackoffMul:  2.0,
	})
	if err != nil {
		t.Errorf("expected nil after retry, got %v", err)
	}
	if h.calls != 3 {
		t.Errorf("expected 3 calls, got %d", h.calls)
	}
}

func TestRetry_exhausted(t *testing.T) {
	wantErr := errors.New("permanent")
	h := &countingHandler{failFor: 99, err: wantErr}
	err := kafka.RetryForTest(context.Background(), h, messaging.Message{Topic: "t"}, kafka.RetryConfigForTest{
		MaxRetries:  3,
		BackoffInit: time.Millisecond,
		BackoffMax:  10 * time.Millisecond,
		BackoffMul:  2.0,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, kafka.ErrMaxRetriesExhausted) {
		t.Errorf("expected ErrMaxRetriesExhausted, got %v", err)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped wantErr, got %v", err)
	}
	if h.calls != 4 {
		t.Errorf("expected 4 calls, got %d", h.calls)
	}
}

func TestRetry_contextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	h := &countingHandler{failFor: 99, err: errors.New("err")}
	err := kafka.RetryForTest(ctx, h, messaging.Message{Topic: "t"}, kafka.RetryConfigForTest{
		MaxRetries:  3,
		BackoffInit: 10 * time.Millisecond,
		BackoffMax:  time.Second,
		BackoffMul:  2.0,
	})
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
}
