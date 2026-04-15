package metrics_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestPackageDepsDoNotIncludeOTLPTransports(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("go", "list", "-deps", "devkit/pkg/o11y/metrics")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps failed: %v\n%s", err, output)
	}

	deps := string(output)
	for _, forbidden := range []string{
		"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc",
		"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp",
		"google.golang.org/grpc",
		"google.golang.org/protobuf",
	} {
		if strings.Contains(deps, forbidden) {
			t.Fatalf("metrics package unexpectedly depends on %q", forbidden)
		}
	}
}
