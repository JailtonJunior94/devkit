package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"devkit/pkg/messaging"
)

var errMaxRetriesExhausted = errors.New("kafka: max retries exhausted")

type retryConfig struct {
	maxRetries  int
	backoffInit time.Duration
	backoffMax  time.Duration
	backoffMul  float64
}

func retry(ctx context.Context, h messaging.Handler, msg messaging.Message, cfg retryConfig, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	var lastErr error
	for attempt := 0; attempt <= cfg.maxRetries; attempt++ {
		if err := h.Handle(ctx, msg); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt == cfg.maxRetries {
			break
		}

		backoff := backoffDuration(cfg.backoffInit, cfg.backoffMax, cfg.backoffMul, attempt)
		logger.Warn("handler error — retrying",
			"topic", msg.Topic,
			"event_type", msg.EventType,
			"attempt", attempt+1,
			"error", lastErr,
			"backoff", backoff,
		)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("%w: %w", errMaxRetriesExhausted, lastErr)
}

func backoffDuration(initial, max time.Duration, multiplier float64, attempt int) time.Duration {
	d := min(time.Duration(float64(initial) * math.Pow(multiplier, float64(attempt))), max)
	return d
}

func publishDLQ(ctx context.Context, w *kafkago.Writer, msg messaging.Message, raw kafkago.Message, handlerErr error, retryCount int) error {
	dlqTopic := msg.Topic + "-dlq"

	headers := []kafkago.Header{
		{Key: "error", Value: []byte(handlerErr.Error())},
		{Key: "event_type", Value: []byte(msg.EventType)},
		{Key: "retry_count", Value: []byte(strconv.Itoa(retryCount))},
		{Key: "origin_topic", Value: []byte(msg.Topic)},
		{Key: "timestamp", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
	}

	dlqMsg := kafkago.Message{
		Topic:   dlqTopic,
		Key:     raw.Key,
		Value:   raw.Value,
		Headers: headers,
	}

	return w.WriteMessages(ctx, dlqMsg)
}
