.PHONY: all build test clean install coverage lint fmt vet dev run e2e bench license-check help

# Binary name
BINARY := secure-backup

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -ldflags "\
	-s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)"

## all: Build the binary (default target)
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY) $(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY) .

## test: Run all tests
test:
	@echo "Running tests..."
	go test ./... -v -race -coverprofile=coverage.out

## coverage: Generate coverage report
coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf dist/

## install: Install binary to /usr/local/bin
install: build
	@echo "Installing $(BINARY) to /usr/local/bin..."
	sudo install -m 755 $(BINARY) /usr/local/bin/

## lint: Run linter (requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running linter..."; \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## dev: Development workflow (fmt, vet, test, build)
dev: clean fmt vet test build
	@echo "Development build complete!"

## run: Build and run
run: build
	./$(BINARY)

## e2e: Run end-to-end pipeline test
e2e: build
	sh test-scripts/e2e_test.sh

## bench: Run compression benchmarks
bench:
	@echo "Running compression benchmarks..."
	go test ./internal/compress/... -bench=. -benchmem -count=3

## license-check: Verify license headers are present
license-check:
	@echo "Checking license headers..."
	@go install github.com/google/addlicense@latest
	@addlicense -check -s -c "Marko Milivojevic" -y "2026" \
		-l apache \
		-ignore 'Makefile' \
		-ignore '**/*.yml' \
		-ignore '**/*.yaml' \
		-ignore '**/*.md' \
		-ignore 'test_data/**' \
		-ignore 'coverage.*' \
		.

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := build
