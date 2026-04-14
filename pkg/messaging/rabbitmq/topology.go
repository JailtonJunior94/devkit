package rabbitmq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// TopologyBuilder declares exchanges, queues, bindings, retry queues and DLQ queues.
type TopologyBuilder struct {
	exchanges     []*Exchange
	queues        []*QueueDecl
	bindings      []*Binding
	retryQueues   []retryQueueCfg
	dlqQueues     []string
	prefetchCount int
}

type retryQueueCfg struct {
	queue    string
	exchange string
	ttl      time.Duration
}

// TopologyOption configures the TopologyBuilder.
type TopologyOption func(*TopologyBuilder)

// NewTopologyBuilder creates a TopologyBuilder with the given options.
func NewTopologyBuilder(opts ...TopologyOption) *TopologyBuilder {
	b := &TopologyBuilder{}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// WithExchange adds an exchange declaration.
func WithExchangeDecl(e *Exchange) TopologyOption {
	return func(b *TopologyBuilder) {
		b.exchanges = append(b.exchanges, e)
	}
}

// WithQueueDecl adds a queue declaration.
func WithQueueDecl(q *QueueDecl) TopologyOption {
	return func(b *TopologyBuilder) {
		b.queues = append(b.queues, q)
	}
}

// WithBindingDecl adds a binding declaration.
func WithBindingDecl(bind *Binding) TopologyOption {
	return func(b *TopologyBuilder) {
		b.bindings = append(b.bindings, bind)
	}
}

// WithRetryQueue configures a retry queue for the given queue using TTL+DLX pattern.
// The retry queue is named "{queue}.retry" and the dead letter exchange is the given exchange.
func WithRetryQueue(queue, exchange string, ttl time.Duration) TopologyOption {
	return func(b *TopologyBuilder) {
		b.retryQueues = append(b.retryQueues, retryQueueCfg{
			queue:    queue,
			exchange: exchange,
			ttl:      ttl,
		})
	}
}

// WithDLQQueue configures a dead letter queue for the given queue.
// The DLQ queue is named "{queue}.dlq".
func WithDLQQueue(queue string) TopologyOption {
	return func(b *TopologyBuilder) {
		b.dlqQueues = append(b.dlqQueues, queue)
	}
}

// WithTopologyPrefetch sets QoS prefetch count on the channel after applying topology.
func WithTopologyPrefetch(n int) TopologyOption {
	return func(b *TopologyBuilder) {
		b.prefetchCount = n
	}
}

// Apply declares all configured exchanges, queues, and bindings on the given channel.
// Operations are idempotent — declaring an existing topology with the same arguments
// does not return an error.
func (b *TopologyBuilder) Apply(ctx context.Context, ch *amqp.Channel) error {
	for _, e := range b.exchanges {
		if err := validateExchangeKind(e.Kind); err != nil {
			return err
		}
		if err := ch.ExchangeDeclare(
			e.Name,
			e.Kind,
			e.Durable,
			e.AutoDelete,
			false, // internal
			false, // noWait
			amqp.Table(e.Arguments),
		); err != nil {
			return fmt.Errorf("declare exchange %q: %w", e.Name, err)
		}
	}

	for _, q := range b.queues {
		if _, err := ch.QueueDeclare(
			q.Name,
			q.Durable,
			q.AutoDelete,
			q.Exclusive,
			false, // noWait
			amqp.Table(q.Arguments),
		); err != nil {
			return fmt.Errorf("declare queue %q: %w", q.Name, err)
		}
	}

	for _, r := range b.retryQueues {
		retryName := RetryQueueName(r.queue)
		args := amqp.Table{
			"x-message-ttl":             int64(r.ttl.Milliseconds()),
			"x-dead-letter-exchange":    r.exchange,
			"x-dead-letter-routing-key": r.queue,
		}
		if _, err := ch.QueueDeclare(
			retryName,
			true,  // durable
			false, // autoDelete
			false, // exclusive
			false, // noWait
			args,
		); err != nil {
			return fmt.Errorf("declare retry queue %q: %w", retryName, err)
		}
	}

	for _, queue := range b.dlqQueues {
		dlqName := DLQName(queue)
		if _, err := ch.QueueDeclare(
			dlqName,
			true,  // durable
			false, // autoDelete
			false, // exclusive
			false, // noWait
			nil,
		); err != nil {
			return fmt.Errorf("declare dlq queue %q: %w", dlqName, err)
		}
	}

	for _, bind := range b.bindings {
		if err := ch.QueueBind(
			bind.Queue,
			bind.RoutingKey,
			bind.Exchange,
			false, // noWait
			amqp.Table(bind.Arguments),
		); err != nil {
			return fmt.Errorf("bind queue %q to exchange %q: %w", bind.Queue, bind.Exchange, err)
		}
	}

	if b.prefetchCount > 0 {
		if err := ch.Qos(b.prefetchCount, 0, false); err != nil {
			return fmt.Errorf("set qos prefetch %d: %w", b.prefetchCount, err)
		}
	}

	return nil
}

// RetryQueueName returns the retry queue name for the given queue.
func RetryQueueName(queue string) string {
	return queue + ".retry"
}

// DLQName returns the dead letter queue name for the given queue.
func DLQName(queue string) string {
	return queue + ".dlq"
}

var validExchangeKinds = map[string]bool{
	"direct":  true,
	"topic":   true,
	"fanout":  true,
	"headers": true,
}

func validateExchangeKind(kind string) error {
	if !validExchangeKinds[kind] {
		return fmt.Errorf("invalid exchange kind %q: must be one of direct, topic, fanout, headers", kind)
	}
	return nil
}
