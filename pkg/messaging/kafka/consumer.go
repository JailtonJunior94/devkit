package kafka

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"devkit/pkg/messaging"
)

type Consumer struct {
	brokers  []string
	groupID  string
	topics   []string
	handlers map[string]messaging.Handler
	mu       sync.RWMutex
	logger   *slog.Logger

	auth   authConfig
	dialer *kafkago.Dialer

	topicWorkers map[string]int
	topicOrdered map[string]bool

	maxRetries  int
	backoffInit time.Duration
	backoffMax  time.Duration
	backoffMul  float64
	dlqEnabled  bool

	eventTypeHeader string

	readers      []*kafkago.Reader
	dlqWriter    *kafkago.Writer
	wg           sync.WaitGroup
	running      atomic.Bool
	stopOnce     sync.Once
	internalStop context.CancelFunc
}

var _ messaging.Consumer = (*Consumer)(nil)

func NewConsumer(opts ...Option) (*Consumer, error) {
	c := &Consumer{
		handlers:        make(map[string]messaging.Handler),
		topicWorkers:    make(map[string]int),
		topicOrdered:    make(map[string]bool),
		logger:          slog.Default(),
		maxRetries:      3,
		backoffInit:     time.Second,
		backoffMax:      30 * time.Second,
		backoffMul:      2.0,
		eventTypeHeader: "event_type",
	}

	for _, opt := range opts {
		opt(c)
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	dialer, err := buildDialer(c.auth)
	if err != nil {
		return nil, err
	}
	c.dialer = dialer

	if c.dlqEnabled {
		c.dlqWriter = &kafkago.Writer{
			Addr:      kafkago.TCP(c.brokers...),
			Transport: dialerTransport(c.dialer),
		}
	}

	return c, nil
}

func (c *Consumer) validate() error {
	var errs []error
	if len(c.brokers) == 0 {
		errs = append(errs, ErrNoBrokers)
	}
	if c.groupID == "" {
		errs = append(errs, ErrNoGroupID)
	}
	if len(c.topics) == 0 {
		errs = append(errs, ErrNoTopics)
	}
	return errors.Join(errs...)
}

func (c *Consumer) RegisterHandler(eventType string, handler messaging.Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[eventType] = handler
}

func (c *Consumer) Consume(ctx context.Context) error {
	if !c.running.CompareAndSwap(false, true) {
		return nil
	}

	internalCtx, cancel := context.WithCancel(ctx)
	c.internalStop = cancel

	c.logger.Info("consumer started",
		"topics", c.topics,
		"group_id", c.groupID,
		"brokers", c.brokers,
	)

	for _, topic := range c.topics {
		r := kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:  c.brokers,
			GroupID:  c.groupID,
			Topic:    topic,
			Dialer:   c.dialer,
			MinBytes: 1,
			MaxBytes: 10e6,
		})
		c.readers = append(c.readers, r)

		workers := c.workersForTopic(topic)
		pool := newWorkerPool(workers)

		c.wg.Add(1)
		go func(reader *kafkago.Reader, tp string, pool *workerPool) {
			defer c.wg.Done()
			pool.start(internalCtx, c.dispatchFn(tp))
			c.fetchLoop(internalCtx, reader, tp, pool)
			pool.stop()
		}(r, topic, pool)
	}

	<-internalCtx.Done()
	return nil
}

func (c *Consumer) workersForTopic(topic string) int {
	if n, ok := c.topicWorkers[topic]; ok {
		return n
	}
	return 1
}

func (c *Consumer) fetchLoop(ctx context.Context, reader *kafkago.Reader, topic string, pool *workerPool) {
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("fetch message error", "topic", topic, "error", err)
			continue
		}

		normalized := normalizeMessage(msg, c.eventTypeHeader)
		c.logger.Debug("message received",
			"topic", topic,
			"partition", msg.Partition,
			"offset", msg.Offset,
			"event_type", normalized.EventType,
		)

		pool.dispatch(workerJob{
			msg:    normalized,
			reader: reader,
			raw:    msg,
		})
	}
}

func (c *Consumer) dispatchFn(topic string) func(context.Context, workerJob) {
	return func(ctx context.Context, job workerJob) {
		c.mu.RLock()
		h, ok := c.handlers[job.msg.EventType]
		c.mu.RUnlock()

		if !ok {
			c.logger.Warn("no handler registered",
				"topic", topic,
				"event_type", job.msg.EventType,
			)
			if err := job.reader.CommitMessages(ctx, job.raw); err != nil {
				c.logger.Error("commit error (no handler)", "topic", topic, "error", err)
			}
			return
		}

		err := retry(ctx, h, job.msg, retryConfig{
			maxRetries:  c.maxRetries,
			backoffInit: c.backoffInit,
			backoffMax:  c.backoffMax,
			backoffMul:  c.backoffMul,
		}, c.logger)

		switch {
		case err == nil:
			if commitErr := job.reader.CommitMessages(ctx, job.raw); commitErr != nil {
				c.logger.Error("commit error", "topic", topic, "error", commitErr)
			} else {
				c.logger.Debug("handler success",
					"topic", topic,
					"event_type", job.msg.EventType,
				)
			}

		case errors.Is(err, errMaxRetriesExhausted):
			c.handleExhausted(ctx, job, err, c.maxRetries)

		default:
			c.logger.Error("handler error (no retry)",
				"topic", topic,
				"event_type", job.msg.EventType,
				"error", err,
			)
		}
	}
}

func (c *Consumer) handleExhausted(ctx context.Context, job workerJob, err error, retryCount int) {
	if c.dlqEnabled && c.dlqWriter != nil {
		if dlqErr := publishDLQ(ctx, c.dlqWriter, job.msg, job.raw, err, retryCount); dlqErr != nil {
			c.logger.Error("dlq publish failed — offset not committed",
				"topic", job.msg.Topic,
				"event_type", job.msg.EventType,
				"error", dlqErr,
			)
			return
		}
		c.logger.Error("message sent to dlq",
			"topic", job.msg.Topic,
			"dlq_topic", job.msg.Topic+"-dlq",
			"event_type", job.msg.EventType,
			"retry_count", retryCount,
			"error", err,
		)
	} else {
		c.logger.Error("retries exhausted and dlq disabled — committing",
			"topic", job.msg.Topic,
			"event_type", job.msg.EventType,
			"retry_count", retryCount,
			"error", err,
		)
	}

	if commitErr := job.reader.CommitMessages(ctx, job.raw); commitErr != nil {
		c.logger.Error("commit error after exhaustion",
			"topic", job.msg.Topic,
			"error", commitErr,
		)
	}
}

func (c *Consumer) Shutdown(ctx context.Context) error {
	var shutdownErr error
	c.stopOnce.Do(func() {
		c.logger.Info("shutdown initiated")
		start := time.Now()

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
			errs = append(errs, ErrShutdownTimeout)
		}

		for _, r := range c.readers {
			if err := r.Close(); err != nil {
				errs = append(errs, err)
			}
		}

		if c.dlqWriter != nil {
			if err := c.dlqWriter.Close(); err != nil {
				errs = append(errs, err)
			}
		}

		c.logger.Info("shutdown complete", "duration", time.Since(start))
		shutdownErr = errors.Join(errs...)
	})
	return shutdownErr
}

func normalizeMessage(msg kafkago.Message, eventTypeHeader string) messaging.Message {
	headers := make(map[string]string, len(msg.Headers))
	for _, h := range msg.Headers {
		headers[h.Key] = string(h.Value)
	}

	eventType, ok := headers[eventTypeHeader]
	if !ok || eventType == "" {
		eventType = msg.Topic
	}

	return messaging.Message{
		EventType: eventType,
		Headers:   headers,
		Body:      msg.Value,
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
	}
}

func dialerTransport(d *kafkago.Dialer) *kafkago.Transport {
	if d == nil {
		return nil
	}
	return &kafkago.Transport{
		SASL: d.SASLMechanism,
		TLS:  d.TLS,
	}
}
