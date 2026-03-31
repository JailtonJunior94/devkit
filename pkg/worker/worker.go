package worker

import (
	"context"
)

type Worker interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Job interface {
	Name() string
	Schedule() string
	Execute(ctx context.Context)
}

type OverlapStrategy int

const (
	SkipIfRunning OverlapStrategy = iota
	DelayIfRunning
	AllowConcurrent
)

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrAlreadyRunning      Error = "worker: manager is already running"
	ErrAllWorkersFailed    Error = "worker: all workers failed to start"
	ErrJobNameRequired     Error = "worker: job name is required"
	ErrJobScheduleRequired Error = "worker: job schedule is required"
	ErrJobFuncRequired     Error = "worker: job function is required"
)
