package worker

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
)

type WorkerManager struct {
	workers  []Worker
	jobs     []Job
	logger   *slog.Logger
	timezone *time.Location
	overlap  OverlapStrategy
	timeout  time.Duration

	lifecycleMu sync.Mutex
	cron        *cron.Cron
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	running     atomic.Bool
	stopOnce    sync.Once
	stopErr     error
}

func NewWorkerManager(opts ...Option) *WorkerManager {
	m := &WorkerManager{
		logger:   slog.Default(),
		timezone: time.UTC,
		overlap:  SkipIfRunning,
		timeout:  30 * time.Second,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *WorkerManager) Start(ctx context.Context) error {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()

	if m.running.Load() {
		m.logger.Error("start called on already-running manager")
		return ErrAlreadyRunning
	}

	validWorkers := make([]Worker, 0, len(m.workers))
	for i, w := range m.workers {
		if w == nil {
			m.logger.Warn("nil worker skipped", "index", i)
			continue
		}
		validWorkers = append(validWorkers, w)
	}
	validJobs := make([]Job, 0, len(m.jobs))
	for i, j := range m.jobs {
		if j == nil {
			m.logger.Warn("nil job skipped", "index", i)
			continue
		}
		validJobs = append(validJobs, j)
	}

	derivedCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	cronLogger := &cronLogAdapter{logger: m.logger}
	cronOpts := []cron.Option{cron.WithLocation(m.timezone)}

	switch m.overlap {
	case SkipIfRunning:
		cronOpts = append(cronOpts, cron.WithChain(cron.SkipIfStillRunning(cronLogger)))
	case DelayIfRunning:
		cronOpts = append(cronOpts, cron.WithChain(cron.DelayIfStillRunning(cronLogger)))
	}
	m.cron = cron.New(cronOpts...)

	registeredJobs := 0
	for _, j := range validJobs {
		job := j
		m.logger.Info("registering job", "job_name", job.Name(), "schedule", job.Schedule())
		_, err := m.cron.AddFunc(job.Schedule(), func() {
			defer func() {
				if r := recover(); r != nil {
					m.logger.Error("panic recovered in job",
						"source", "job",
						"job_name", job.Name(),
						"panic", r,
						"stack", string(debug.Stack()),
					)
				}
			}()
			m.logger.Debug("executing job", "job_name", job.Name())
			job.Execute(derivedCtx)
		})
		if err != nil {
			m.logger.Error("failed to register job", "job_name", job.Name(), "error", err)
			continue
		}
		registeredJobs++
	}

	if len(validWorkers) == 0 {
		m.cron.Start()
		m.running.Store(true)
		return nil
	}

	startedCh := make(chan struct{}, len(validWorkers))
	doneCh := make(chan error, len(validWorkers))

	for i, w := range validWorkers {
		m.wg.Add(1)
		go func(idx int, worker Worker) {
			defer m.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					m.logger.Error("panic recovered in worker",
						"source", "worker",
						"worker_index", idx,
						"panic", r,
						"stack", string(debug.Stack()),
					)
					doneCh <- errors.New("worker panicked")
				}
			}()

			startedCh <- struct{}{}

			m.logger.Info("starting worker", "worker_index", idx)
			err := worker.Start(derivedCtx)
			if err != nil {
				m.logger.Error("worker exited with error", "worker_index", idx, "error", err)
			} else {
				m.logger.Info("worker finished", "worker_index", idx)
			}
			doneCh <- err
		}(i, w)
	}

	for range validWorkers {
		<-startedCh
	}

	failCount := 0
	completedCount := 0
	deadline := time.NewTimer(10 * time.Millisecond)
	defer deadline.Stop()

collect:
	for completedCount < len(validWorkers) {
		select {
		case err := <-doneCh:
			completedCount++
			if err != nil {
				failCount++
			}
			if completedCount == len(validWorkers) {
				break collect
			}
		case <-deadline.C:
			break collect
		}
	}

	if completedCount == len(validWorkers) && failCount == len(validWorkers) && registeredJobs == 0 {
		cancel()
		m.cron.Stop()
		return ErrAllWorkersFailed
	}

	m.cron.Start()
	m.running.Store(true)
	return nil
}

func (m *WorkerManager) Stop(ctx context.Context) error {
	m.stopOnce.Do(func() {
		m.lifecycleMu.Lock()
		defer m.lifecycleMu.Unlock()

		m.running.Store(false)
		m.logger.Info("shutdown initiated")

		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, m.timeout)
			defer cancel()
		}

		start := time.Now()

		var cronCtx context.Context
		if m.cron != nil {
			cronCtx = m.cron.Stop()
		} else {
			done := make(chan struct{})
			close(done)
			cronCtx = &closedContext{done: done}
		}

		if m.cancel != nil {
			m.cancel()
		}

		var errs []error
		timedOut := false
		for i, w := range m.workers {
			if w == nil {
				continue
			}
			if err := w.Stop(ctx); err != nil {
				errs = append(errs, err)
				m.logger.Error("worker stop error", "worker_index", i, "error", err)
			}
		}

		select {
		case <-cronCtx.Done():
		case <-ctx.Done():
			if !timedOut {
				m.logger.Error("timeout waiting for cron jobs to finish")
				errs = append(errs, ctx.Err())
				timedOut = true
			}
		}

		wgDone := make(chan struct{})
		go func() {
			m.wg.Wait()
			close(wgDone)
		}()

		select {
		case <-wgDone:
		case <-ctx.Done():
			if !timedOut {
				m.logger.Error("timeout waiting for worker goroutines")
				errs = append(errs, ctx.Err())
			}
		}

		m.logger.Info("shutdown complete", "duration", time.Since(start))
		m.stopErr = errors.Join(errs...)
	})
	return m.stopErr
}

type closedContext struct {
	done chan struct{}
}

func (c *closedContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *closedContext) Done() <-chan struct{}       { return c.done }
func (c *closedContext) Err() error                  { return context.Canceled }
func (c *closedContext) Value(_ any) any             { return nil }

func (m *WorkerManager) IsRunning() bool {
	return m.running.Load()
}
