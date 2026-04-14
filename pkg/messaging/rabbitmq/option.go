package rabbitmq

import (
	"crypto/tls"
	"log/slog"
	"time"
)

// Option configures a Consumer.
type Option func(*Consumer)

// WithURI sets the AMQP connection URI.
func WithURI(uri string) Option {
	return func(c *Consumer) {
		c.uri = uri
	}
}

// WithQueues adds queues to consume from.
func WithQueues(queues ...string) Option {
	return func(c *Consumer) {
		c.queues = append(c.queues, queues...)
	}
}

// WithPrefetch sets the prefetch count (QoS) per consumer channel.
// Must be explicitly configured; NewConsumer returns ErrNoPrefetch if absent.
func WithPrefetch(n int) Option {
	return func(c *Consumer) {
		c.prefetch = n
		c.prefetchSet = true
	}
}

// WithWorkers sets the number of concurrent workers for a specific queue.
func WithWorkers(queue string, n int) Option {
	return func(c *Consumer) {
		if n < 1 {
			n = 1
		}
		c.queueWorkers[queue] = n
		c.queueOrdered[queue] = false
	}
}

// WithOrderedProcessing forces sequential (single-worker) processing for a queue.
// It overrides any workers count set for the same queue.
func WithOrderedProcessing(queue string) Option {
	return func(c *Consumer) {
		c.queueOrdered[queue] = true
		c.queueWorkers[queue] = 1
	}
}

// WithEventTypeHeader sets the AMQP message header key used to extract the event type.
// Defaults to "event_type".
func WithEventTypeHeader(key string) Option {
	return func(c *Consumer) {
		if key != "" {
			c.eventTypeHeader = key
		}
	}
}

// WithConsumerLogger injects a custom logger into the consumer.
func WithConsumerLogger(logger *slog.Logger) Option {
	return func(c *Consumer) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// WithConsumerTLS configures TLS for the AMQP connection.
func WithConsumerTLS(cfg *tls.Config) Option {
	return func(c *Consumer) {
		c.tlsCfg = cfg
	}
}

// WithMaxRetries sets the maximum number of retry attempts before a message is sent to the DLQ.
// Defaults to 3 if not set and retry is enabled.
func WithMaxRetries(n int) Option {
	return func(c *Consumer) {
		if n >= 0 {
			c.maxRetries = n
		}
	}
}

// WithConsumerRetry enables retry via native RabbitMQ topology (retry queue with TTL + DLX).
// exchange is the dead-letter exchange that routes messages back to the original queue after TTL.
// ttl is the message time-to-live in the retry queue before redelivery.
func WithConsumerRetry(exchange string, ttl time.Duration) Option {
	return func(c *Consumer) {
		c.retryEnabled = true
		c.retryExchange = exchange
		c.retryTTL = ttl
	}
}

// WithDLQEnabled enables dead letter queue publishing when max retries are exhausted.
// Failed messages are published to "{queue}.dlq" with enriched metadata headers.
func WithDLQEnabled() Option {
	return func(c *Consumer) {
		c.dlqEnabled = true
	}
}
