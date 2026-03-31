package worker_test

import (
	"context"
	"log"
	"time"

	"devkit/pkg/worker"
)

func ExampleNewWorkerManager() {
	w := &fakeWorker{
		startFunc: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
	}

	job, err := worker.NewJobAdapter("ping", "* * * * *", func(_ context.Context) {
	})
	if err != nil {
		log.Fatal(err)
	}

	m := worker.NewWorkerManager(
		worker.WithWorkers(w),
		worker.WithJobs(job),
		worker.WithOverlapStrategy(worker.SkipIfRunning),
		worker.WithShutdownTimeout(30*time.Second),
	)

	ctx, cancel := context.WithCancel(context.Background())

	if err := m.Start(ctx); err != nil {
		log.Fatal(err)
	}

	cancel()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	if err := m.Stop(stopCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

}
