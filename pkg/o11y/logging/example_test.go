package logging_test

import (
	"context"
	"fmt"

	"devkit/pkg/o11y/logging"
)

func ExampleNew() {
	provider, err := logging.New(context.Background(), logging.Config{
		ServiceName: "checkout",
	})
	if err != nil {
		panic(err)
	}
	defer provider.Shutdown(context.Background()) //nolint:errcheck

	_ = provider.Logger()

	fmt.Println("logging initialized")
	// Output: logging initialized
}
