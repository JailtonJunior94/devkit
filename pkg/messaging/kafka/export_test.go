package kafka

import (
	"context"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"devkit/pkg/messaging"
)

func BackoffDuration(initial, max time.Duration, multiplier float64, attempt int) time.Duration {
	return backoffDuration(initial, max, multiplier, attempt)
}

var ErrMaxRetriesExhausted = errMaxRetriesExhausted

type RetryConfigForTest struct {
	MaxRetries  int
	BackoffInit time.Duration
	BackoffMax  time.Duration
	BackoffMul  float64
}

func RetryForTest(ctx context.Context, h messaging.Handler, msg messaging.Message, cfg RetryConfigForTest) error {
	return retry(ctx, h, msg, retryConfig{
		maxRetries:  cfg.MaxRetries,
		backoffInit: cfg.BackoffInit,
		backoffMax:  cfg.BackoffMax,
		backoffMul:  cfg.BackoffMul,
	}, nil)
}

func NormalizeMessage(msg kafkago.Message, eventTypeHeader string) messaging.Message {
	return normalizeMessage(msg, eventTypeHeader)
}

type WorkerJobForTest struct {
	N int
}

type testWorkerPool struct {
	inner *workerPool
}

func NewWorkerPoolForTest(size int) *testWorkerPool {
	return &testWorkerPool{inner: newWorkerPool(size)}
}

func (tp *testWorkerPool) Start(ctx context.Context, fn func(context.Context, WorkerJobForTest)) {
	tp.inner.start(ctx, func(ctx context.Context, job workerJob) {
		fn(ctx, WorkerJobForTest{N: int(job.raw.Offset)})
	})
}

func (tp *testWorkerPool) DispatchForTest(job WorkerJobForTest) {
	tp.inner.dispatch(workerJob{
		raw: kafkago.Message{Offset: int64(job.N)},
	})
}

func (tp *testWorkerPool) Stop() {
	tp.inner.stop()
}
