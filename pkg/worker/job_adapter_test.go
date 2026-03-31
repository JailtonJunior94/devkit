package worker_test

import (
	"context"
	"errors"
	"testing"

	"devkit/pkg/worker"
)

func TestNewJobAdapter_validation(t *testing.T) {
	t.Parallel()

	fn := func(_ context.Context) {}

	tests := []struct {
		name     string
		jobName  string
		schedule string
		fn       func(context.Context)
		wantErr  error
	}{
		{
			name:     "empty name",
			jobName:  "",
			schedule: "* * * * *",
			fn:       fn,
			wantErr:  worker.ErrJobNameRequired,
		},
		{
			name:     "empty schedule",
			jobName:  "cleanup",
			schedule: "",
			fn:       fn,
			wantErr:  worker.ErrJobScheduleRequired,
		},
		{
			name:     "nil fn",
			jobName:  "cleanup",
			schedule: "* * * * *",
			fn:       nil,
			wantErr:  worker.ErrJobFuncRequired,
		},
		{
			name:     "valid inputs",
			jobName:  "cleanup",
			schedule: "0 0 * * *",
			fn:       fn,
			wantErr:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := worker.NewJobAdapter(tc.jobName, tc.schedule, tc.fn)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("NewJobAdapter() error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil && got != nil {
				t.Error("NewJobAdapter() should return nil job on error")
			}
			if tc.wantErr == nil && got == nil {
				t.Error("NewJobAdapter() should return non-nil job on success")
			}
		})
	}
}

func TestNewJobAdapter_methods(t *testing.T) {
	t.Parallel()

	const wantName = "report"
	const wantSchedule = "0 9 * * 1"
	executed := make(chan context.Context, 1)

	job, err := worker.NewJobAdapter(wantName, wantSchedule, func(ctx context.Context) {
		executed <- ctx
	})
	if err != nil {
		t.Fatalf("NewJobAdapter() unexpected error: %v", err)
	}

	if got := job.Name(); got != wantName {
		t.Errorf("Name() = %q, want %q", got, wantName)
	}
	if got := job.Schedule(); got != wantSchedule {
		t.Errorf("Schedule() = %q, want %q", got, wantSchedule)
	}

	ctx := context.Background()
	job.Execute(ctx)

	select {
	case got := <-executed:
		if got != ctx {
			t.Error("Execute() passed unexpected context")
		}
	default:
		t.Error("Execute() did not call the underlying function")
	}
}
