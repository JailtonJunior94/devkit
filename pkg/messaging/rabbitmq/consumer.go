package rabbitmq

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"devkit/pkg/messaging"
)

// Consumer consumes messages from one or more RabbitMQ queues.
// It dispatches messages to registered handlers using a per-queue worker pool,
// with manual ack/nack based on handler result.
type Consumer struct {
	uri         string
	queues      []string
	prefetch    int
	prefetchSet bool
	tlsCfg      *tls.Config

	handlers        map[string]messaging.Handler
	mu              sync.RWMutex
	queueWorkers    map[string]int
	queueOrdered    map[string]bool
	eventTypeHeader string
	logger          *slog.Logger

	maxRetries    int
	retryEnabled  bool
	retryExchange string
	retryTTL      time.Duration
	dlqEnabled    bool

	connMgr   *connectionManager
	channels  map[string]*amqp.Channel
	pools     map[string]*workerPool
	publishCh *amqp.Channel
	publishMu sync.Mutex

	wg           sync.WaitGroup
	running      atomic.Bool
	stopOnce     sync.Once
	internalCtx  context.Context
	internalStop context.CancelFunc
}

// Compile-time assertion: Consumer implements messaging.Consumer.
var _ messaging.Consumer = (*Consumer)(nil)

// NewConsumer creates and validates a RabbitMQ consumer from the given options.
// Returns ErrNoURI, ErrNoQueues, or ErrNoPrefetch if required configuration is missing.
func NewConsumer(opts ...Option) (*Consumer, error) {
	c := &Consumer{
		handlers:        make(map[string]messaging.Handler),
		queueWorkers:    make(map[string]int),
		queueOrdered:    make(map[string]bool),
		channels:        make(map[string]*amqp.Channel),
		pools:           make(map[string]*workerPool),
		eventTypeHeader: "event_type",
		logger:          slog.Default(),
		maxRetries:      3,
	}

	for _, opt := range opts {
		opt(c)
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	c.connMgr = newConnectionManager(c.uri, c.tlsCfg, c.logger)
	c.connMgr.onReconnect = c.reregisterConsumers

	return c, nil
}

func (c *Consumer) validate() error {
	var errs []error
	if c.uri == "" {
		errs = append(errs, ErrNoURI)
	}
	if len(c.queues) == 0 {
		errs = append(errs, ErrNoQueues)
	}
	if !c.prefetchSet {
		errs = append(errs, ErrNoPrefetch)
	}
	return errors.Join(errs...)
}

// RegisterHandler registers a handler for a given event type. Thread-safe.
func (c *Consumer) RegisterHandler(eventType string, handler messaging.Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[eventType] = handler
}

// Consume connects to RabbitMQ and begins consuming all configured queues.
// Blocks until the context is cancelled. Returns ErrShutdown if called after Shutdown.
func (c *Consumer) Consume(ctx context.Context) error {
	if !c.running.CompareAndSwap(false, true) {
		return nil
	}

	if err := c.connMgr.dial(); err != nil {
		c.running.Store(false)
		return err
	}

	internalCtx, cancel := context.WithCancel(ctx)
	c.internalCtx = internalCtx
	c.internalStop = cancel

	c.logger.Info("consumer started", "queues", c.queues)

	if err := c.setupQueues(internalCtx); err != nil {
		cancel()
		c.running.Store(false)
		return err
	}

	<-internalCtx.Done()
	return nil
}

func (c *Consumer) setupPublishChannel() error {
	if !c.retryEnabled && !c.dlqEnabled {
		return nil
	}
	ch, err := c.connMgr.newChannel()
	if err != nil {
		return fmt.Errorf("open publish channel: %w", err)
	}
	c.publishMu.Lock()
	c.publishCh = ch
	c.publishMu.Unlock()
	return nil
}

func (c *Consumer) setupQueues(ctx context.Context) error {
	if err := c.setupPublishChannel(); err != nil {
		return err
	}

	for _, queue := range c.queues {
		ch, err := c.connMgr.newChannel()
		if err != nil {
			return fmt.Errorf("open channel for queue %q: %w", queue, err)
		}

		if err := ch.Qos(c.prefetch, 0, false); err != nil {
			_ = ch.Close()
			return fmt.Errorf("set qos for queue %q: %w", queue, err)
		}

		deliveries, err := ch.Consume(queue, "", false, false, false, false, nil)
		if err != nil {
			_ = ch.Close()
			return fmt.Errorf("start consuming queue %q: %w", queue, err)
		}

		c.channels[queue] = ch

		workers := c.workersForQueue(queue)
		pool := newWorkerPool(workers)
		c.pools[queue] = pool
		pool.start()

		c.wg.Add(1)
		go func(q string, pool *workerPool, deliveryCh <-chan amqp.Delivery) {
			defer c.wg.Done()
			c.consumeLoop(ctx, q, pool, deliveryCh)
			pool.stop()
		}(queue, pool, deliveries)
	}
	return nil
}

func (c *Consumer) consumeLoop(ctx context.Context, queue string, pool *workerPool, deliveries <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-deliveries:
			if !ok {
				// channel closed — reconnection will re-register consumers
				return
			}
			msg := normalizeMessage(d, queue, c.eventTypeHeader)
			c.logger.Debug("message received",
				"queue", queue,
				"delivery_tag", d.DeliveryTag,
				"event_type", msg.EventType,
			)
			pool.dispatch(c.handleDelivery(ctx, queue, d, msg))
		}
	}
}

func (c *Consumer) handleDelivery(ctx context.Context, queue string, d amqp.Delivery, msg messaging.Message) task {
	return func() {
		c.mu.RLock()
		h, ok := c.handlers[msg.EventType]
		c.mu.RUnlock()

		if !ok {
			c.logger.Warn("no handler registered",
				"queue", queue,
				"event_type", msg.EventType,
			)
			if err := d.Nack(false, false); err != nil {
				c.logger.Error("nack error (no handler)", "queue", queue, "error", err)
			}
			return
		}

		handlerErr := h.Handle(ctx, msg)
		if handlerErr == nil {
			if err := d.Ack(false); err != nil {
				c.logger.Error("ack error", "queue", queue, "error", err)
			}
			c.logger.Debug("handler success", "queue", queue, "event_type", msg.EventType)
			return
		}

		c.logger.Error("handler error",
			"queue", queue,
			"event_type", msg.EventType,
			"error", handlerErr,
		)

		retryCount := retryCountFromHeaders(d.Headers)

		if c.retryEnabled && retryCount < c.maxRetries {
			c.publishRetry(queue, d, retryCount+1)
			if err := d.Ack(false); err != nil {
				c.logger.Error("ack error after retry publish", "queue", queue, "error", err)
			}
			return
		}

		if c.dlqEnabled {
			c.publishDLQ(ctx, queue, d, msg, handlerErr, retryCount)
			if err := d.Ack(false); err != nil {
				c.logger.Error("ack error after dlq publish", "queue", queue, "error", err)
			}
			return
		}

		if err := d.Nack(false, false); err != nil {
			c.logger.Error("nack error", "queue", queue, "error", err)
		}
	}
}

// retryCountFromHeaders reads x-retry-count from AMQP headers (default 0).
func retryCountFromHeaders(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	v, ok := headers["x-retry-count"]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int32:
		return int(val)
	case int64:
		return int(val)
	case string:
		n, err := strconv.Atoi(val)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}

// publishRetry publishes the message to the retry queue with incremented x-retry-count.
func (c *Consumer) publishRetry(queue string, d amqp.Delivery, newCount int) {
	retryQueue := RetryQueueName(queue)
	headers := copyHeaders(d.Headers)
	headers["x-retry-count"] = newCount

	pub := amqp.Publishing{
		Headers:      amqp.Table(headers),
		ContentType:  d.ContentType,
		Body:         d.Body,
		DeliveryMode: d.DeliveryMode,
	}

	c.publishMu.Lock()
	ch := c.publishCh
	c.publishMu.Unlock()

	if ch == nil {
		c.logger.Error("retry publish channel not available", "queue", queue)
		return
	}

	if err := ch.PublishWithContext(context.Background(), "", retryQueue, false, false, pub); err != nil {
		c.logger.Error("retry publish error", "queue", queue, "retry_queue", retryQueue, "error", err)
	} else {
		c.logger.Debug("message sent to retry queue", "queue", queue, "retry_count", newCount)
	}
}

// publishDLQ publishes the message to the DLQ with enriched metadata headers.
func (c *Consumer) publishDLQ(ctx context.Context, queue string, d amqp.Delivery, msg messaging.Message, handlerErr error, retryCount int) {
	dlqQueue := DLQName(queue)
	headers := copyHeaders(d.Headers)
	headers["error"] = handlerErr.Error()
	headers["event_type"] = msg.EventType
	headers["retry_count"] = strconv.Itoa(retryCount)
	headers["origin_queue"] = queue
	headers["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	pub := amqp.Publishing{
		Headers:      amqp.Table(headers),
		ContentType:  d.ContentType,
		Body:         d.Body,
		DeliveryMode: d.DeliveryMode,
	}

	c.publishMu.Lock()
	ch := c.publishCh
	c.publishMu.Unlock()

	if ch == nil {
		c.logger.Error("dlq publish channel not available", "queue", queue)
		return
	}

	if err := ch.PublishWithContext(ctx, "", dlqQueue, false, false, pub); err != nil {
		c.logger.Error("dlq publish error", "queue", queue, "dlq_queue", dlqQueue, "error", err)
	} else {
		c.logger.Debug("message sent to dlq", "queue", queue, "dlq_queue", dlqQueue)
	}
}

// copyHeaders creates a copy of AMQP headers as map[string]any.
func copyHeaders(src amqp.Table) map[string]any {
	dst := make(map[string]any, len(src)+5)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// reregisterConsumers is called by connectionManager after a reconnection.
// It reopens channels and restarts consumers for each queue.
func (c *Consumer) reregisterConsumers() error {
	if !c.running.Load() {
		return nil
	}

	// stop existing pools and close old channels
	for _, queue := range c.queues {
		if pool, ok := c.pools[queue]; ok {
			pool.stop()
			delete(c.pools, queue)
		}
		if ch, ok := c.channels[queue]; ok {
			_ = ch.Close()
			delete(c.channels, queue)
		}
	}

	// close and reset publish channel so setupPublishChannel reopens it
	c.publishMu.Lock()
	if c.publishCh != nil {
		_ = c.publishCh.Close()
		c.publishCh = nil
	}
	c.publishMu.Unlock()

	ctx := c.internalCtx
	if ctx == nil {
		ctx = context.Background()
	}

	return c.setupQueues(ctx)
}

// Shutdown gracefully stops the consumer: cancels consumption, drains workers,
// and closes the AMQP connection.
func (c *Consumer) Shutdown(ctx context.Context) error {
	var shutdownErr error
	c.stopOnce.Do(func() {
		c.logger.Info("consumer shutdown initiated")

		c.running.Store(false)

		if c.internalStop != nil {
			c.internalStop()
		}

		wgDone := make(chan struct{})
		go func() {
			c.wg.Wait()
			close(wgDone)
		}()

		var errs []error
		select {
		case <-wgDone:
		case <-ctx.Done():
			errs = append(errs, ctx.Err())
		}

		for _, ch := range c.channels {
			if err := ch.Close(); err != nil {
				errs = append(errs, err)
			}
		}

		c.publishMu.Lock()
		if c.publishCh != nil {
			if err := c.publishCh.Close(); err != nil {
				errs = append(errs, err)
			}
			c.publishCh = nil
		}
		c.publishMu.Unlock()

		c.connMgr.close()
		c.logger.Info("consumer shutdown complete")
		shutdownErr = errors.Join(errs...)
	})
	return shutdownErr
}

func (c *Consumer) workersForQueue(queue string) int {
	if n, ok := c.queueWorkers[queue]; ok {
		return n
	}
	return 1
}

// normalizeMessage maps an amqp.Delivery to a messaging.Message.
// queue becomes Topic, DeliveryTag becomes Offset, Partition is always 0.
// The event type is extracted from the configured header; falls back to the queue name.
func normalizeMessage(d amqp.Delivery, queue, eventTypeHeader string) messaging.Message {
	headers := make(map[string]string, len(d.Headers))
	for k, v := range d.Headers {
		if s, ok := v.(string); ok {
			headers[k] = s
		} else {
			headers[k] = fmt.Sprintf("%v", v)
		}
	}

	eventType, ok := headers[eventTypeHeader]
	if !ok || eventType == "" {
		eventType = queue
	}

	return messaging.Message{
		EventType: eventType,
		Headers:   headers,
		Body:      d.Body,
		Topic:     queue,
		Partition: 0,
		Offset:    int64(d.DeliveryTag),
	}
}
