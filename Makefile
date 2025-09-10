.PHONY: build test mock-test capture clean help lint fmt fmt-check unit-test integration-test coverage check ci

# Default target
help:
	@echo "Available targets:"
	@echo "  build       - Build the redfish_exporter binary"
	@echo "  test        - Run all tests"
	@echo "  unit-test   - Run unit tests only"
	@echo "  integration-test - Run integration tests"
	@echo "  lint        - Run golangci-lint"
	@echo "  fmt         - Format code with gofmt"
	@echo "  fmt-check   - Check if code is formatted"
	@echo "  coverage    - Generate and show test coverage"
	@echo "  check       - Run fmt, lint, and test (dev workflow)"
	@echo "  ci          - Run all CI checks"
	@echo "  mock-test   - Run mock server with exporter for testing"
	@echo "  capture     - Capture Redfish data from a BMC"
	@echo "  clean       - Clean build artifacts"
	@echo ""
	@echo "For mock-test, you can specify TESTDATA:"
	@echo "  make mock-test TESTDATA=gb300/gb300_host.txt"
	@echo ""
	@echo "For capture, specify HOST, USER, PASS, and OUTPUT:"
	@echo "  make capture HOST=10.0.0.1 USER=admin PASS=password OUTPUT=mysystem"

# Build the exporter
build:
	go build -o redfish_exporter

# Run tests
test:
	go test -v ./...

# Run mock server and exporter for testing
mock-test:
	@if [ -z "$(TESTDATA)" ]; then \
		echo "Using default test data: gb300/gb300_host.txt"; \
		echo "You can specify different data with: make mock-test TESTDATA=<path>"; \
		./tools/mock-server/test-local.sh gb300/gb300_host.txt; \
	else \
		./tools/mock-server/test-local.sh $(TESTDATA); \
	fi

# Build and run the capture tool
capture:
	@if [ -z "$(HOST)" ] || [ -z "$(USER)" ] || [ -z "$(PASS)" ] || [ -z "$(OUTPUT)" ]; then \
		echo "Error: Required parameters missing"; \
		echo "Usage: make capture HOST=<bmc-ip> USER=<username> PASS=<password> OUTPUT=<name>"; \
		echo ""; \
		echo "Optional parameters:"; \
		echo "  TIMEOUT=30s            - Request timeout (default: 10s)"; \
		echo "  SLEEP=100              - Sleep between requests in ms (default: 0)"; \
		echo "  MAX=100                - Max endpoints to capture (default: unlimited)"; \
		echo "  INSECURE=false         - Skip TLS verification (default: true)"; \
		echo "  SKIP_PATHS=Actions,Logs - Comma-separated paths to skip (default: Actions)"; \
		exit 1; \
	fi
	@echo "Building capture tool..."
	@cd tools/capture && go build -o capture main.go
	@echo "Capturing from $(HOST)..."
	@cd tools/capture && ./capture \
		-host $(HOST) \
		-user $(USER) \
		-pass $(PASS) \
		-output $(OUTPUT) \
		$(if $(TIMEOUT),-timeout $(TIMEOUT)) \
		$(if $(SLEEP),-sleep $(SLEEP)) \
		$(if $(MAX),-max $(MAX)) \
		$(if $(filter false,$(INSECURE)),-insecure=false) \
		$(if $(SKIP_PATHS),-skipPaths "$(SKIP_PATHS)")
	@echo ""
	@echo "Capture complete! To test with mock server:"
	@echo "  make mock-test TESTDATA=$(OUTPUT)/capture.txt"

# Clean build artifacts
clean:
	rm -f redfish_exporter
	rm -f tools/mock-server/exporter-config.yml
	rm -f tools/capture/capture
	rm -f coverage.out

# Run unit tests only (exclude integration)
unit-test:
	go test -v -race -short ./...

# Run integration tests
integration-test:
	go test -v -run Integration ./collector

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	gofmt -w -s .
	go mod tidy

# Check formatting
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Please run 'make fmt'" && exit 1)

# Generate test coverage
coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Development workflow - format, lint, and test
check: fmt lint test

# CI workflow - stricter checks
ci: fmt-check lint test build