.PHONY: all build test lint fmt vet check clean codegen

# Default target
all: check

# Build the SDK (verification only — library has no main)
build:
	go build ./...

# Run all tests
test:
	go test ./... -count=1

# Run tests with race detector
test-race:
	go test ./... -race -count=1

# Format code
fmt:
	gofmt -s -w .

# Vet code
vet:
	go vet ./...

# Lint (go vet + staticcheck if available)
lint: vet
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

# Run all checks (CI target)
check: fmt vet test

# Generate types from OpenAPI spec
codegen:
	./scripts/codegen.sh

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -f *.test *.out coverage.out coverage.html

# Download dependencies
deps:
	go mod download

# Verify dependencies
verify:
	go mod verify
