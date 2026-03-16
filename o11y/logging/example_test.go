package logging_test

import (
	"context"
	"fmt"

	"devkit/o11y/logging"
)

func ExampleNew_fallback() {
	p, err := logging.New(context.Background(), logging.Config{
		ServiceName: "my-service",
	})
	if err != nil {
		panic(err)
	}
	defer p.Shutdown(context.Background()) //nolint:errcheck

	// Logger falls back to slog.Default() when no exporter is configured.
	_ = p.Logger()

	fmt.Println("logging initialized")
	// Output: logging initialized
}
