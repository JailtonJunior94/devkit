package worker

import "context"

type jobAdapter struct {
	name     string
	schedule string
	fn       func(ctx context.Context)
}

func NewJobAdapter(name, schedule string, fn func(ctx context.Context)) (Job, error) {
	if name == "" {
		return nil, ErrJobNameRequired
	}
	if schedule == "" {
		return nil, ErrJobScheduleRequired
	}
	if fn == nil {
		return nil, ErrJobFuncRequired
	}
	return &jobAdapter{name: name, schedule: schedule, fn: fn}, nil
}

func (j *jobAdapter) Name() string { return j.name }

func (j *jobAdapter) Schedule() string { return j.schedule }

func (j *jobAdapter) Execute(ctx context.Context) { j.fn(ctx) }
