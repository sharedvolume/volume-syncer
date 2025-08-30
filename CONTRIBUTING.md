# Contributing to Volume Syncer

We love your input! We want to make contributing to Volume Syncer as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

## Pull Requests

Pull requests are the best way to propose changes to the codebase. We actively welcome your pull requests:

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Code Standards

### Go Code Style

- Follow standard Go formatting (`go fmt`)
- Use `golint` and `go vet` to check your code
- Write meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions small and focused

### Project Structure

Please follow the existing project structure:

```
├── cmd/                    # Application entry points
├── internal/              # Private application code
│   ├── config/           # Configuration management
│   ├── handler/          # HTTP handlers
│   ├── models/           # Data models
│   ├── server/           # Server setup
│   ├── service/          # Business logic
│   ├── syncer/           # Sync implementations
│   └── utils/            # Utility functions
├── pkg/                   # Public libraries
└── docs/                  # Documentation
```

### Testing

- Write unit tests for new functionality
- Maintain or improve test coverage
- Use table-driven tests where appropriate
- Mock external dependencies
- Test error conditions

Example test structure:
```go
func TestSyncService_ValidateRequest(t *testing.T) {
    tests := []struct {
        name    string
        request models.SyncRequest
        wantErr bool
    }{
        // test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Getting Started

### Prerequisites

- Go 1.21 or later
- Docker (for running tests with containers)
- Git

### Setup Development Environment

1. Clone your fork:
```bash
git clone https://github.com/yourusername/volume-syncer.git
cd volume-syncer
```

2. Install dependencies:
```bash
go mod download
```

3. Run tests:
```bash
go test ./...
```

4. Build the application:
```bash
go build -o volume-syncer ./cmd/server
```

5. Run the application:
```bash
./volume-syncer
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/service/

# Run with race detection
go test -race ./...
```

### Building Docker Image

```bash
docker build -t volume-syncer:dev .
```

## Adding New Sync Sources

To add a new sync source type:

1. Create a new directory under `internal/syncer/`
2. Implement the `Syncer` interface:
```go
type Syncer interface {
    Sync(source SourceDetails, target TargetDetails) error
}
```
3. Add the new source type to `internal/models/requests.go`
4. Update the factory function in `internal/syncer/types.go`
5. Add validation logic in `internal/service/sync_service.go`
6. Write comprehensive tests
7. Update documentation

## Reporting Bugs

We use GitHub issues to track bugs. Report a bug by [opening a new issue](https://github.com/sharedvolume/volume-syncer/issues).

**Great Bug Reports** tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

### Bug Report Template

```markdown
**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Send request with payload for SSH sync to 'example-server.com'
2. Check logs for connection timeout
3. See error "connection refused"

**Expected behavior**
SSH connection should establish successfully and sync files.

**Actual behavior**
Connection times out after 30 seconds with "connection refused" error.

**Environment:**
- OS: [e.g. macOS, Linux, Windows]
- Go version: [e.g. 1.21.5]
- Volume Syncer version: [e.g. 0.1.0]
- Container runtime: [e.g. Docker 24.0.7]

**Additional context**
Server is reachable via ping and SSH manually works with same credentials.
```

## Feature Requests

We welcome feature requests! Please open an issue with:

- Clear description of the feature
- Use case and motivation
- Possible implementation approach
- Any alternatives you've considered

## Security Issues

Please do not report security issues in public GitHub issues. Instead, send an email to security@example.com with details.

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

## Code of Conduct

### Our Pledge

We pledge to make participation in our project a harassment-free experience for everyone, regardless of age, body size, disability, ethnicity, gender identity and expression, level of experience, nationality, personal appearance, race, religion, or sexual identity and orientation.

### Our Standards

Examples of behavior that contributes to creating a positive environment include:

- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

### Enforcement

Project maintainers are responsible for clarifying standards and are expected to take appropriate action in response to any instances of unacceptable behavior.

## Questions?

Feel free to open an issue for any questions about contributing!