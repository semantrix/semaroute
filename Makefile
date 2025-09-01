# semaroute Makefile
# Common development and build tasks

.PHONY: help build run test clean deps lint format

# Default target
help:
	@echo "semaroute - Available targets:"
	@echo "  build    - Build the binary"
	@echo "  run      - Run the server"
	@echo "  test     - Run tests"
	@echo "  clean    - Clean build artifacts"
	@echo "  deps     - Download and tidy dependencies"
	@echo "  lint     - Run linter"
	@echo "  format   - Format code"
	@echo "  docker   - Build Docker image"

# Build the binary
build:
	@echo "Building semaroute-server..."
	@mkdir -p bin
	go build -o bin/semaroute-server ./cmd/semaroute-server
	@echo "Build complete: bin/semaroute-server"

# Run the server
run:
	@echo "Starting semaroute-server..."
	go run ./cmd/semaroute-server

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean -cache
	@echo "Clean complete"

# Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies updated"

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
format:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t semaroute:latest .

# Install development tools
tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Create logs directory
logs:
	@mkdir -p logs
	@echo "Logs directory created"

# Development setup
dev: deps logs
	@echo "Development environment ready"

# Production build
prod: clean
	@echo "Building production binary..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o bin/semaroute-server ./cmd/semaroute-server
	@echo "Production build complete: bin/semaroute-server"
