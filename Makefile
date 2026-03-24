GOLANGCI_LINT_VERSION := v2.11.3
GOSEC_VERSION := v2.24.7

.PHONY: lint test test-integration security ci tools

tools:
	@command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	@command -v gosec >/dev/null 2>&1 || go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)

lint: tools
	golangci-lint run --config .github/golangci.yml ./...

test:
	go test -race -coverprofile=coverage.out ./...

test-integration:
	go test -race -tags integration -coverprofile=coverage-integration.out ./pkg/database/...

security: tools
	govulncheck ./...
	gosec ./...

ci: lint test security
