package worker

import (
	"log/slog"
	"time"
)

type Option func(*WorkerManager)

func WithWorkers(workers ...Worker) Option {
	return func(m *WorkerManager) {
		m.workers = append(m.workers, workers...)
	}
}

func WithJobs(jobs ...Job) Option {
	return func(m *WorkerManager) {
		m.jobs = append(m.jobs, jobs...)
	}
}

func WithTimezone(loc *time.Location) Option {
	return func(m *WorkerManager) {
		if loc != nil {
			m.timezone = loc
		}
	}
}

func WithOverlapStrategy(strategy OverlapStrategy) Option {
	return func(m *WorkerManager) {
		m.overlap = strategy
	}
}

func WithShutdownTimeout(d time.Duration) Option {
	return func(m *WorkerManager) {
		m.timeout = d
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(m *WorkerManager) {
		if logger != nil {
			m.logger = logger
		}
	}
}
