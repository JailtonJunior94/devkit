package worker_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"devkit/pkg/worker"
)

type fakeWorker struct {
	startFunc func(ctx context.Context) error
	stopFunc  func(ctx context.Context) error
}

func (f *fakeWorker) Start(ctx context.Context) error {
	if f.startFunc != nil {
		return f.startFunc(ctx)
	}
	<-ctx.Done()
	return nil
}

func (f *fakeWorker) Stop(ctx context.Context) error {
	if f.stopFunc != nil {
		return f.stopFunc(ctx)
	}
	return nil
}

type fakeJob struct {
	name     string
	schedule string
	execFunc func(ctx context.Context)
}

func (j *fakeJob) Name() string     { return j.name }
func (j *fakeJob) Schedule() string { return j.schedule }
func (j *fakeJob) Execute(ctx context.Context) {
	if j.execFunc != nil {
		j.execFunc(ctx)
	}
}

func newBlockingWorker() *fakeWorker {
	return &fakeWorker{
		startFunc: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
	}
}

func newFailingWorker(err error) *fakeWorker {
	return &fakeWorker{
		startFunc: func(_ context.Context) error {
			return err
		},
	}
}

func TestNewWorkerManager_defaults(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager()
	if m == nil {
		t.Fatal("NewWorkerManager() returned nil")
	}
	if m.IsRunning() {
		t.Error("IsRunning() should be false before Start()")
	}
}

func TestWithLogger_nil_is_noop(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager(worker.WithLogger(nil))
	if m == nil {
		t.Fatal("nil logger option should not cause nil manager")
	}
}

func TestWithTimezone_nil_is_noop(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager(worker.WithTimezone(nil))
	if m == nil {
		t.Fatal("nil timezone option should not cause nil manager")
	}
}

func TestWithShutdownTimeout(t *testing.T) {
	t.Parallel()
	_ = worker.NewWorkerManager(worker.WithShutdownTimeout(5 * time.Second))
}

func TestStart_nil_workers_are_skipped(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager(worker.WithWorkers(nil, newBlockingWorker()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}

func TestStart_with_jobs_registered(t *testing.T) {
	t.Parallel()

	executed := make(chan struct{}, 1)
	j := &fakeJob{
		name:     "test-job",
		schedule: "* * * * *",
		execFunc: func(_ context.Context) {
			select {
			case executed <- struct{}{}:
			default:
			}
		},
	}

	m := worker.NewWorkerManager(worker.WithJobs(j))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if !m.IsRunning() {
		t.Error("IsRunning() should be true after Start() with only jobs")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}

func TestStart_nil_jobs_are_skipped(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager(worker.WithJobs(nil))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() with nil jobs unexpected error: %v", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}

func TestStart_returns_error_if_already_running(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager(worker.WithWorkers(newBlockingWorker()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("first Start() unexpected error: %v", err)
	}

	if err := m.Start(ctx); !errors.Is(err, worker.ErrAlreadyRunning) {
		t.Errorf("second Start() error = %v, want ErrAlreadyRunning", err)
	}

	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}

func TestStart_no_workers_starts_cron_successfully(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if !m.IsRunning() {
		t.Error("IsRunning() should be true after Start()")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}

func TestStart_all_workers_fail_returns_ErrAllWorkersFailed(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("startup error")
	m := worker.NewWorkerManager(
		worker.WithWorkers(
			newFailingWorker(errBoom),
			newFailingWorker(errBoom),
		),
	)

	err := m.Start(context.Background())
	if !errors.Is(err, worker.ErrAllWorkersFailed) {
		t.Errorf("Start() error = %v, want ErrAllWorkersFailed", err)
	}
	if m.IsRunning() {
		t.Error("IsRunning() should be false when all workers fail")
	}
}

func TestStart_some_workers_fail_continues_running(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("startup error")
	m := worker.NewWorkerManager(
		worker.WithWorkers(
			newFailingWorker(errBoom),
			newBlockingWorker(),
		),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if !m.IsRunning() {
		t.Error("IsRunning() should be true when at least one worker is running")
	}

	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}

func TestStart_all_workers_fail_with_jobs_continues_running(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("startup error")
	job := &fakeJob{
		name:     "cleanup",
		schedule: "* * * * *",
	}
	m := worker.NewWorkerManager(
		worker.WithWorkers(
			newFailingWorker(errBoom),
			newFailingWorker(errBoom),
		),
		worker.WithJobs(job),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if !m.IsRunning() {
		t.Error("IsRunning() should be true when jobs are registered successfully")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}

func TestIsRunning_lifecycle(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager(worker.WithWorkers(newBlockingWorker()))

	if m.IsRunning() {
		t.Error("IsRunning() should be false before Start()")
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if !m.IsRunning() {
		t.Error("IsRunning() should be true after Start()")
	}

	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck

	if m.IsRunning() {
		t.Error("IsRunning() should be false after Stop()")
	}
}

func TestStop_before_start_does_not_panic(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager()
	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := m.Stop(stopCtx)
	_ = err
}

func TestStop_without_deadline_uses_configured_timeout(t *testing.T) {
	t.Parallel()
	w := &fakeWorker{
		startFunc: func(_ context.Context) error {
			select {}
		},
		stopFunc: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	m := worker.NewWorkerManager(
		worker.WithWorkers(w),
		worker.WithShutdownTimeout(100*time.Millisecond),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	start := time.Now()
	err := m.Stop(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Stop() expected an error due to configured timeout")
	}
	if elapsed > 3*time.Second {
		t.Errorf("Stop() took %v — configured timeout of 100ms was not applied", elapsed)
	}
}

func TestStop_is_idempotent(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager(worker.WithWorkers(newBlockingWorker()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck

	err1 := m.Stop(stopCtx)
	err2 := m.Stop(stopCtx)

	if err1 != err2 {
		t.Errorf("Stop() idempotency: first=%v, second=%v", err1, err2)
	}
}

func TestStop_collects_worker_stop_errors(t *testing.T) {
	t.Parallel()
	stopErr := errors.New("stop failed")
	w := &fakeWorker{
		startFunc: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
		stopFunc: func(_ context.Context) error {
			return stopErr
		},
	}

	m := worker.NewWorkerManager(worker.WithWorkers(w))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck

	err := m.Stop(stopCtx)
	if !errors.Is(err, stopErr) {
		t.Errorf("Stop() error = %v, want to contain %v", err, stopErr)
	}
}

func TestStop_timeout_exceeded(t *testing.T) {
	t.Parallel()

	var stopCalled atomic.Bool
	w := &fakeWorker{
		startFunc: func(_ context.Context) error {
			select {}
		},
		stopFunc: func(_ context.Context) error {
			stopCalled.Store(true)
			return nil
		},
	}

	m := worker.NewWorkerManager(worker.WithWorkers(w))
	startCtx, startCancel := context.WithCancel(context.Background())
	defer startCancel()

	if err := m.Start(startCtx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck

	err := m.Stop(stopCtx)
	if err == nil {
		t.Error("Stop() expected an error due to timeout")
	}
	if !stopCalled.Load() {
		t.Error("Stop() should invoke worker Stop even when shutdown times out")
	}
}

func TestStart_worker_panic_is_recovered(t *testing.T) {
	t.Parallel()

	panicWorker := &fakeWorker{
		startFunc: func(_ context.Context) error {
			panic("unexpected panic")
		},
	}

	m := worker.NewWorkerManager(worker.WithWorkers(panicWorker))
	err := m.Start(context.Background())
	if !errors.Is(err, worker.ErrAllWorkersFailed) {
		t.Fatalf("Start() error = %v, want ErrAllWorkersFailed", err)
	}
}

func TestConcurrent_start_stop_isrunning(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager(worker.WithWorkers(newBlockingWorker()))

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	const goroutines = 10

	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = m.Start(ctx)
	}()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.IsRunning()
		}()
	}

	wg.Wait()

	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck

	var stopWg sync.WaitGroup
	for i := 0; i < 3; i++ {
		stopWg.Add(1)
		go func() {
			defer stopWg.Done()
			_ = m.Stop(stopCtx)
		}()
	}
	stopWg.Wait()
}

func TestConcurrent_start_only_one_succeeds(t *testing.T) {
	t.Parallel()

	m := worker.NewWorkerManager(worker.WithWorkers(newBlockingWorker()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const starters = 8
	results := make(chan error, starters)

	var wg sync.WaitGroup
	for i := 0; i < starters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- m.Start(ctx)
		}()
	}
	wg.Wait()
	close(results)

	var nilCount int
	var alreadyRunningCount int
	for err := range results {
		switch {
		case err == nil:
			nilCount++
		case errors.Is(err, worker.ErrAlreadyRunning):
			alreadyRunningCount++
		default:
			t.Fatalf("Start() error = %v, want nil or ErrAlreadyRunning", err)
		}
	}

	if nilCount != 1 {
		t.Fatalf("successful starts = %d, want 1", nilCount)
	}
	if alreadyRunningCount != starters-1 {
		t.Fatalf("ErrAlreadyRunning count = %d, want %d", alreadyRunningCount, starters-1)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}

func TestWithOverlapStrategy_skipIfRunning(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager(
		worker.WithOverlapStrategy(worker.SkipIfRunning),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}

func TestWithOverlapStrategy_delayIfRunning(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager(
		worker.WithOverlapStrategy(worker.DelayIfRunning),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}

func TestWithOverlapStrategy_allowConcurrent(t *testing.T) {
	t.Parallel()
	m := worker.NewWorkerManager(
		worker.WithOverlapStrategy(worker.AllowConcurrent),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}

func TestWithTimezone_non_nil(t *testing.T) {
	t.Parallel()
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Skip("timezone America/Sao_Paulo not available:", err)
	}
	m := worker.NewWorkerManager(worker.WithTimezone(loc))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	m.Stop(stopCtx) //nolint:errcheck
}
