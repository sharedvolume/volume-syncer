.PHONY: build run test docker-build docker-run clean help coverage lint fmt deps vet security

# Variables
BINARY_NAME=volume-syncer
DOCKER_IMAGE=sharedvolume/volume-syncer
GO_VERSION=1.21

# Default target
help:
	@echo "Available targets:"
	@echo "  build       - Build the Go binary"
	@echo "  run         - Run the application locally"
	@echo "  test        - Run tests"
	@echo "  coverage    - Run tests with coverage"
	@echo "  lint        - Run linting"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  security    - Run security checks"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-build-tag - Build Docker image with version tag"
	@echo "  docker-run  - Run with Docker Compose"
	@echo "  docker-stop - Stop Docker Compose"
	@echo "  docker-inspect - Inspect Docker image labels"
	@echo "  docker-push - Push Docker image to registry"
	@echo "  clean       - Clean build artifacts"
	@echo "  release     - Build release binaries"
	@echo "  gen-key     - Generate test SSH key"
	@echo "  help        - Show this help"

# Build the Go binary
build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BINARY_NAME) ./cmd/server

# Build release binaries for multiple platforms
release:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/server
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/server

# Run the application locally
run:
	go run ./cmd/server

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Run go vet
vet:
	go vet ./...

# Lint code
lint:
	golangci-lint run

# Run security checks
security:
	gosec ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Verify dependencies
verify:
	go mod verify

# Build Docker image
docker-build:
	docker build \
		--build-arg BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
		--build-arg GIT_COMMIT=$(shell git rev-parse HEAD) \
		--build-arg VERSION=latest \
		-t $(DOCKER_IMAGE):latest .

# Build Docker image with version tag
docker-build-tag:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make docker-build-tag VERSION=x.x.x"; exit 1; fi
	docker build \
		--build-arg BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
		--build-arg GIT_COMMIT=$(shell git rev-parse HEAD) \
		--build-arg VERSION=$(VERSION) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest .

# Run with Docker Compose
docker-run:
	docker-compose up --build

# Stop Docker Compose
docker-stop:
	docker-compose down

# Inspect Docker image labels
docker-inspect:
	@echo "Inspecting Docker image labels..."
	docker inspect $(DOCKER_IMAGE):latest --format='{{json .Config.Labels}}' | jq .

# Push Docker image (requires login)
docker-push:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make docker-push VERSION=x.x.x"; exit 1; fi
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/
	rm -f coverage.out coverage.html
	docker-compose down --rmi local --volumes --remove-orphans

# Install development tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Generate base64 private key for testing
gen-key:
	@echo "Generating test SSH key..."
	ssh-keygen -t rsa -b 2048 -f test_key -N '' -C "test-key-for-volume-syncer"
	@echo "Base64 encoded private key:"
	@base64 -w 0 test_key
	@echo ""
	@echo "Remember to delete test_key and test_key.pub after testing"

# Run all checks (for CI)
ci: deps fmt vet lint security test

# Quick development check
check: fmt vet test
