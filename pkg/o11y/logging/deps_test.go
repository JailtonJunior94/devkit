package logging_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestPackageDepsDoNotIncludeOTLPTransports(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("go", "list", "-deps", "devkit/pkg/o11y/logging")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps failed: %v\n%s", err, output)
	}

	deps := string(output)
	for _, forbidden := range []string{
		"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc",
		"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp",
		"google.golang.org/grpc",
		"google.golang.org/protobuf",
	} {
		if strings.Contains(deps, forbidden) {
			t.Fatalf("logging package unexpectedly depends on %q", forbidden)
		}
	}
}
