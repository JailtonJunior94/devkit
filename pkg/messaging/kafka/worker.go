package kafka

import (
	"context"
	"sync"

	kafkago "github.com/segmentio/kafka-go"

	"devkit/pkg/messaging"
)

type workerJob struct {
	msg    messaging.Message
	reader *kafkago.Reader
	raw    kafkago.Message
}

type workerPool struct {
	size int
	jobs chan workerJob
	wg   sync.WaitGroup
}

func newWorkerPool(size int) *workerPool {
	if size < 1 {
		size = 1
	}
	return &workerPool{
		size: size,
		jobs: make(chan workerJob, size*4),
	}
}

func (p *workerPool) start(ctx context.Context, fn func(context.Context, workerJob)) {
	for i := 0; i < p.size; i++ {
		p.wg.Go(func() {
			for job := range p.jobs {
				fn(ctx, job)
			}
		})
	}
}

func (p *workerPool) dispatch(job workerJob) {
	p.jobs <- job
}

func (p *workerPool) stop() {
	close(p.jobs)
	p.wg.Wait()
}
