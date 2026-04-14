package rabbitmq

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestWorkerPool_ProcessesTasks(t *testing.T) {
	t.Parallel()

	pool := newWorkerPool(3)
	pool.start()

	var count atomic.Int64
	var wg sync.WaitGroup

	const n = 30
	wg.Add(n)
	for range n {
		pool.dispatch(func() {
			defer wg.Done()
			count.Add(1)
		})
	}

	wg.Wait()
	pool.stop()

	if got := count.Load(); got != n {
		t.Errorf("expected %d tasks processed, got %d", n, got)
	}
}

func TestWorkerPool_StopDrainsInFlight(t *testing.T) {
	t.Parallel()

	pool := newWorkerPool(2)
	pool.start()

	var count atomic.Int64
	const n = 20
	for range n {
		pool.dispatch(func() {
			count.Add(1)
		})
	}

	pool.stop() // must wait for all tasks

	if got := count.Load(); got != n {
		t.Errorf("expected all %d tasks after stop, got %d", n, got)
	}
}

func TestWorkerPool_SizeNormalization(t *testing.T) {
	t.Parallel()

	pool := newWorkerPool(0)
	if pool.size != 1 {
		t.Errorf("expected size=1 for input 0, got %d", pool.size)
	}

	pool = newWorkerPool(-5)
	if pool.size != 1 {
		t.Errorf("expected size=1 for input -5, got %d", pool.size)
	}
}

func TestWorkerPool_Race(t *testing.T) {
	t.Parallel()

	pool := newWorkerPool(5)
	pool.start()

	var count atomic.Int64
	const n = 100
	for range n {
		pool.dispatch(func() {
			count.Add(1)
		})
	}

	pool.stop()

	if got := count.Load(); got != n {
		t.Errorf("race test: expected %d, got %d", n, got)
	}
}
