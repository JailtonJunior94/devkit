package rabbitmq

import (
	"crypto/tls"
	"log/slog"
	"time"
)

// ProducerOption configures a Producer.
type ProducerOption func(*Producer)

// WithProducerURI sets the AMQP connection URI.
func WithProducerURI(uri string) ProducerOption {
	return func(p *Producer) {
		p.uri = uri
	}
}

// WithProducerExchange sets the default exchange for publishing.
func WithProducerExchange(exchange string) ProducerOption {
	return func(p *Producer) {
		p.exchange = exchange
	}
}

// WithProducerRoutingKey sets the default routing key for publishing.
func WithProducerRoutingKey(key string) ProducerOption {
	return func(p *Producer) {
		p.routingKey = key
	}
}

// WithProducerMandatory sets the mandatory flag. When true, the broker returns
// the message if it cannot be routed to any queue.
func WithProducerMandatory(m bool) ProducerOption {
	return func(p *Producer) {
		p.mandatory = m
	}
}

// WithProducerLogger injects a custom logger.
func WithProducerLogger(logger *slog.Logger) ProducerOption {
	return func(p *Producer) {
		if logger != nil {
			p.logger = logger
		}
	}
}

// WithProducerTLS configures TLS for the AMQP connection.
func WithProducerTLS(cfg *tls.Config) ProducerOption {
	return func(p *Producer) {
		p.tlsCfg = cfg
	}
}

// WithProducerConfirm enables publisher confirms (at-least-once delivery guarantee).
func WithProducerConfirm() ProducerOption {
	return func(p *Producer) {
		p.confirmMode = true
	}
}

// WithProducerMaxRetries sets the maximum number of publish retries on transient errors.
// Default is 3.
func WithProducerMaxRetries(n int) ProducerOption {
	return func(p *Producer) {
		if n >= 0 {
			p.maxRetries = n
		}
	}
}

// WithProducerBackoff configures exponential backoff for publish retries.
// Default: initial=100ms, max=5s, multiplier=2.0.
func WithProducerBackoff(initial, max time.Duration, multiplier float64) ProducerOption {
	return func(p *Producer) {
		p.backoffInit = initial
		p.backoffMax = max
		p.backoffMul = multiplier
	}
}
