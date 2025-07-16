.PHONY: build run test docker-build docker-run clean help

# Default target
help:
	@echo "Available targets:"
	@echo "  build       - Build the Go binary"
	@echo "  run         - Run the application locally"
	@echo "  test        - Run tests"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run  - Run with Docker Compose"
	@echo "  clean       - Clean build artifacts"
	@echo "  help        - Show this help"

# Build the Go binary
build:
	go build -o volume-syncer ./cmd/server

# Run the application locally
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Build Docker image
docker-build:
	docker build -t volume-syncer .

# Run with Docker Compose
docker-run:
	docker-compose up --build

# Stop Docker Compose
docker-stop:
	docker-compose down

# Clean build artifacts
clean:
	rm -f volume-syncer
	docker-compose down --rmi local --volumes --remove-orphans

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy

# Generate base64 private key for testing
gen-key:
	@echo "To generate a base64 private key for testing:"
	@echo "ssh-keygen -t rsa -b 2048 -f test_key -N ''"
	@echo "base64 -w 0 test_key"
