package oteltest_test

import (
	"fmt"

	"devkit/pkg/o11y/oteltest"
)

func ExampleNewFakeLogger() {
	logger := oteltest.NewFakeLogger()
	logger.Logger().Info("hello")

	fmt.Println(len(logger.Records()))
	// Output: 1
}
