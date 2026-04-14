package rabbitmq

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"devkit/pkg/messaging"
)

// Producer publishes messages to a RabbitMQ exchange with optional publisher confirms
// and exponential-backoff retry on transient errors.
type Producer struct {
	uri        string
	exchange   string
	routingKey string
	mandatory  bool
	tlsCfg     *tls.Config
	logger     *slog.Logger

	confirmMode bool
	maxRetries  int
	backoffInit time.Duration
	backoffMax  time.Duration
	backoffMul  float64

	connMgr  *connectionManager
	chMu     sync.Mutex
	ch       *amqp.Channel
	confirms <-chan amqp.Confirmation

	running  atomic.Bool
	stopOnce sync.Once
}

// Compile-time assertion: Producer implements messaging.Producer.
var _ messaging.Producer = (*Producer)(nil)

// NewProducer creates and validates a RabbitMQ producer from the given options.
// Returns ErrNoURI if the URI is not configured.
func NewProducer(opts ...ProducerOption) (*Producer, error) {
	p := &Producer{
		logger:      slog.Default(),
		maxRetries:  3,
		backoffInit: 100 * time.Millisecond,
		backoffMax:  5 * time.Second,
		backoffMul:  2.0,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.uri == "" {
		return nil, ErrNoURI
	}

	p.connMgr = newConnectionManager(p.uri, p.tlsCfg, p.logger)
	return p, nil
}

// Publish serializes msg and sends it to the broker.
// Per-call PublishOptions override the producer's default exchange and routing key.
// Returns an error if the connection is not established or publish fails after retries.
func (p *Producer) Publish(ctx context.Context, msg messaging.Message, opts ...messaging.PublishOption) error {
	if !p.running.Load() {
		if err := p.connect(); err != nil {
			return err
		}
	}

	cfg := messaging.PublishConfig{
		Exchange:   p.exchange,
		RoutingKey: p.routingKey,
		Mandatory:  p.mandatory,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	pub := serializeMessage(msg)

	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			delay := calculateBackoff(p.backoffInit, p.backoffMax, p.backoffMul, attempt-1)
			p.logger.Warn("retrying publish",
				"exchange", cfg.Exchange,
				"routing_key", cfg.RoutingKey,
				"attempt", attempt,
				"delay", delay,
			)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := p.publishOnce(ctx, cfg, pub); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("publish failed after %d attempts: %w", p.maxRetries+1, lastErr)
}

func (p *Producer) publishOnce(ctx context.Context, cfg messaging.PublishConfig, pub amqp.Publishing) error {
	p.chMu.Lock()
	ch := p.ch
	var confirms <-chan amqp.Confirmation
	if p.confirmMode {
		confirms = p.confirms
	}
	p.chMu.Unlock()

	if ch == nil {
		if err := p.connect(); err != nil {
			return err
		}
		p.chMu.Lock()
		ch = p.ch
		if p.confirmMode {
			confirms = p.confirms
		}
		p.chMu.Unlock()
	}

	if err := ch.PublishWithContext(ctx, cfg.Exchange, cfg.RoutingKey, cfg.Mandatory, false, pub); err != nil {
		p.chMu.Lock()
		p.ch = nil
		p.chMu.Unlock()
		return fmt.Errorf("publish: %w", err)
	}

	if p.confirmMode && confirms != nil {
		select {
		case confirm, ok := <-confirms:
			if !ok {
				return errors.New("confirm channel closed")
			}
			if !confirm.Ack {
				return errors.New("broker nacked message")
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	p.logger.Debug("message published",
		"exchange", cfg.Exchange,
		"routing_key", cfg.RoutingKey,
	)
	return nil
}

func (p *Producer) connect() error {
	p.chMu.Lock()
	defer p.chMu.Unlock()

	if p.ch != nil && !p.ch.IsClosed() {
		return nil
	}

	if err := p.connMgr.dial(); err != nil {
		return err
	}

	ch, err := p.connMgr.newChannel()
	if err != nil {
		return err
	}

	if p.confirmMode {
		if err := ch.Confirm(false); err != nil {
			_ = ch.Close()
			return fmt.Errorf("enable confirm mode: %w", err)
		}
		notifyCh := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
		p.confirms = notifyCh
	}

	p.ch = ch
	p.running.Store(true)
	return nil
}

// Shutdown gracefully drains in-flight publishes and closes the connection.
func (p *Producer) Shutdown(ctx context.Context) error {
	var shutdownErr error
	p.stopOnce.Do(func() {
		p.running.Store(false)

		p.chMu.Lock()
		if p.ch != nil {
			if err := p.ch.Close(); err != nil {
				shutdownErr = err
			}
			p.ch = nil
		}
		p.chMu.Unlock()

		p.connMgr.close()
		p.logger.Info("producer shutdown complete")
	})
	return shutdownErr
}

// serializeMessage maps a messaging.Message to an amqp.Publishing.
func serializeMessage(msg messaging.Message) amqp.Publishing {
	headers := make(amqp.Table, len(msg.Headers))
	for k, v := range msg.Headers {
		headers[k] = v
	}
	if msg.EventType != "" {
		headers["event_type"] = msg.EventType
	}
	return amqp.Publishing{
		Headers:     headers,
		Body:        msg.Body,
		ContentType: "application/octet-stream",
	}
}
