package worker_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"devkit/pkg/worker"
)

func benchmarkLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func BenchmarkWorkerManager_IsRunning(b *testing.B) {
	b.ReportAllocs()

	manager := worker.NewWorkerManager(
		worker.WithLogger(benchmarkLogger()),
	)

	for i := 0; i < b.N; i++ {
		_ = manager.IsRunning()
	}
}

func BenchmarkWorkerManager_Stop_NoWorkers(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		manager := worker.NewWorkerManager(
			worker.WithLogger(benchmarkLogger()),
		)

		startCtx, startCancel := context.WithCancel(context.Background())
		if err := manager.Start(startCtx); err != nil {
			b.Fatalf("Start() error = %v", err)
		}

		startCancel()

		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
		if err := manager.Stop(stopCtx); err != nil {
			stopCancel()
			b.Fatalf("Stop() error = %v", err)
		}
		stopCancel()
	}
}
