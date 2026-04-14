package rabbitmq

import "sync"

// task is a unit of work dispatched to a worker pool.
type task func()

type workerPool struct {
	size  int
	tasks chan task
	wg    sync.WaitGroup
}

func newWorkerPool(size int) *workerPool {
	if size < 1 {
		size = 1
	}
	return &workerPool{
		size:  size,
		tasks: make(chan task, size*4),
	}
}

// start launches size worker goroutines that process tasks from the internal channel.
func (p *workerPool) start() {
	for range p.size {
		p.wg.Go(func() {
			for t := range p.tasks {
				t()
			}
		})
	}
}

// dispatch sends a task to the worker pool. Blocks if the internal buffer is full.
func (p *workerPool) dispatch(t task) {
	p.tasks <- t
}

// stop closes the task channel and waits for all in-flight tasks to complete.
func (p *workerPool) stop() {
	close(p.tasks)
	p.wg.Wait()
}
