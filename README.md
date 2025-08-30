# Volume Syncer

A professional-grade, high-performance API server for synchronizing data from various sources (SSH, Git, S3) to local volumes. Built specifically for Kubernetes SharedVolume operators with Go and Gin framework, designed for containerized environments and enterprise workloads.

**Repository**: [https://github.com/sharedvolume/volume-syncer](https://github.com/sharedvolume/volume-syncer)  
**Releases**: [https://github.com/sharedvolume/volume-syncer/releases](https://github.com/sharedvolume/volume-syncer/releases)

## 🚀 Features

- **📡 REST API**: Simple HTTP API for triggering sync operations with comprehensive error handling
- **🔐 SSH Support**: Sync from remote servers using SSH with private key and password authentication
- **📚 Git Support**: Clone and sync from Git repositories with full authentication support
- **🌐 HTTP Support**: Download files from HTTP/HTTPS URLs with robust error handling
- **☁️ S3 Support**: Sync data from AWS S3 or S3-compatible storage systems
- **🔧 Extensible Design**: Clean architecture for adding support for additional sources
- **⚡ Concurrent Safety**: Prevents multiple sync operations with mutex protection
- **💚 Health Checks**: Kubernetes-ready health endpoints for readiness and liveness probes
- **🐳 Kubernetes Ready**: Optimized for Kubernetes deployments with proper signal handling
- **⏱️ Timeout Support**: Configurable timeouts for sync operations
- **🔒 Security First**: Enterprise-grade security with Apache 2.0 license

## 📋 API Endpoints

### Health Check
```
GET /health
```
Returns server health status for Kubernetes readiness and liveness probes.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-08-30T10:30:00Z"
}
```

### Sync Data
```
POST /api/1.0/sync
```

**Request payload:**
```json
{
  "source": {
    "type": "ssh",
    "details": {
      "host": "example-server.com",
      "port": 22,
      "username": "deploy",
      "path": "/opt/data",
      "privateKey": "<base64 encoded private key>"
    }
  },
  "target": {
    "path": "/mnt/shared-volume"
  },
  "timeout": "30s"
}
```

**Response codes:**
- `201`: Sync started successfully
- `400`: Invalid request format or parameters
- `503`: Sync already in progress

## 🚀 Quick Start

### Using Docker Compose (Recommended for Development)

1. Clone the repository:
```bash
git clone https://github.com/sharedvolume/volume-syncer.git
cd volume-syncer
```

2. Build and run:
```bash
docker-compose up --build
```

3. Test the health endpoint:
```bash
curl http://localhost:8080/health
```

### Using Docker (Production Ready)

1. Build the image:
```bash
docker build -t volume-syncer .
```

2. Run the container:
```bash
docker run -p 8080:8080 -v $(pwd)/data:/mnt/shared-volume volume-syncer
```

### Kubernetes Deployment

For production Kubernetes deployments, use the provided manifests or integrate with your SharedVolume operator:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: volume-syncer
  labels:
    app: volume-syncer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: volume-syncer
  template:
    metadata:
      labels:
        app: volume-syncer
    spec:
      containers:
      - name: volume-syncer
        image: volume-syncer:latest
        ports:
        - containerPort: 8080
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 20
        volumeMounts:
        - name: shared-volume
          mountPath: /mnt/shared-volume
      volumes:
      - name: shared-volume
        persistentVolumeClaim:
          claimName: shared-volume-pvc
```

### Local Development

1. Install dependencies:
```bash
go mod tidy
```

2. Run the server:
```bash
go run cmd/server/main.go
```

## ⚙️ Configuration

### Source Types

Currently supported:
- **SSH**: Sync from remote servers via SSH
- **Git**: Clone/pull from Git repositories  
- **HTTP**: Download from HTTP/HTTPS endpoints
- **S3**: Sync from AWS S3 or S3-compatible storage

### SSH Configuration

- `host`: SSH server hostname or IP (required)
- `port`: SSH port (default: 22)
- `username`: SSH username (default: "root")
- `privateKey`: Base64-encoded SSH private key (optional)
- `password`: SSH password (optional)

**Note**: `privateKey` and `password` cannot be provided at the same time.

### Git Configuration

- `url`: Git repository URL (required)
- `branch`: Branch to clone (optional, if not specified, uses repository's default branch)
- `depth`: Clone depth (optional, default: 1 for shallow clone)
- `username`: Username for HTTP authentication (optional, requires password)
- `password`: Password for HTTP authentication (optional, requires username)
- `privateKey`: Base64-encoded SSH private key for SSH authentication (optional)

**Note**: `username`/`password` and `privateKey` cannot be provided at the same time.

### HTTP Configuration

- `url`: HTTP/HTTPS URL to download (required)

### S3 Configuration

- `endpointUrl`: S3 endpoint URL (required, e.g., "https://s3.amazonaws.com")
- `bucketName`: S3 bucket name (required)
- `path`: Path/prefix in the bucket (required)
- `accessKey`: AWS access key (required)
- `secretKey`: AWS secret key (required)
- `region`: AWS region (required)

### Environment Variables

- `GIN_MODE`: Set to "release" for production deployments
- `PORT`: Server port (default: 8080)
- `LOG_LEVEL`: Logging level (default: "info", options: "debug", "info", "warn", "error")

## 💡 Example Usage

### Sync from SSH Server

```bash
# Using private key (base64 encoded)
curl -X POST http://localhost:8080/api/1.0/sync \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "ssh",
      "details": {
        "host": "example.com",
        "port": 22,
        "username": "deploy",
        "privateKey": "LS0tLS1CRUdJTi..."
      }
    },
    "target": {
      "path": "/mnt/shared-volume"
    }
  }'

# Using password authentication
curl -X POST http://localhost:8080/api/1.0/sync \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "ssh",
      "details": {
        "host": "example.com",
        "port": 22,
        "username": "deploy",
        "password": "your-password"
      }
    },
    "target": {
      "path": "/mnt/shared-volume"
    }
  }'
```

### Sync from Git Repository

```bash
# Clone with username/password authentication
curl -X POST http://localhost:8080/api/1.0/sync \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "git",
      "details": {
        "url": "https://github.com/user/repo.git",
        "branch": "main",
        "depth": 1,
        "username": "your-username",
        "password": "your-password"
      }
    },
    "target": {
      "path": "/mnt/shared-volume"
    }
  }'

# Clone with SSH private key authentication
curl -X POST http://localhost:8080/api/1.0/sync \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "git",
      "details": {
        "url": "git@github.com:user/repo.git",
        "branch": "main",
        "depth": 1,
        "privateKey": "LS0tLS1CRUdJTi..."
      }
    },
    "target": {
      "path": "/mnt/shared-volume"
    }
  }'

# Clone using repository default branch (no branch specified)
curl -X POST http://localhost:8080/api/1.0/sync \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "git",
      "details": {
        "url": "https://github.com/user/repo.git",
        "depth": 1
      }
    },
    "target": {
      "path": "/mnt/shared-volume"
    }
  }'
```

### Download from HTTP URL

```bash
curl -X POST http://localhost:8080/api/1.0/sync \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "http",
      "details": {
        "url": "https://example.com/file.zip"
      }
    },
    "target": {
      "path": "/mnt/shared-volume"
    }
  }'
```

### Sync from S3 Bucket

```bash
curl -X POST http://localhost:8080/api/1.0/sync \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "s3",
      "details": {
        "endpointUrl": "https://s3.amazonaws.com",
        "bucketName": "my-aws-bucket",
        "path": "project/data/",
        "accessKey": "AKIA....",
        "secretKey": "abcd1234...",
        "region": "us-east-1"
      }
    },
    "target": {
      "path": "/mnt/shared-volume"
    }
  }'
```

### Generate Base64 Private Key

```bash
base64 -w 0 ~/.ssh/id_rsa
```

## 🔐 Security Considerations

- **Enterprise Ready**: Apache 2.0 license for commercial and enterprise usage
- **Secure Credential Handling**: Private keys and credentials are handled securely in memory
- **Temporary File Security**: SSH key files created with restrictive permissions (600)
- **Host Verification**: SSH host key verification configurable for production environments
- **Container Security**: Runs as non-root user following security best practices
- **Data Cleanup**: All sensitive data and temporary files cleaned up after use
- **Input Validation**: Comprehensive validation and sanitization of all API inputs

## 📊 Monitoring & Health Checks

The server provides comprehensive health check endpoints optimized for Kubernetes:

**Health Check Endpoint:** `/health`
```json
{
  "status": "healthy",
  "timestamp": "2025-08-30T10:30:00Z"
}
```

**Kubernetes Integration:**
- **Readiness Probe**: Use `/health` endpoint to determine when pod is ready to receive traffic
- **Liveness Probe**: Use `/health` endpoint to determine when pod should be restarted
- **Graceful Shutdown**: Proper signal handling for Kubernetes pod lifecycle

## 🛠️ Development

### Project Structure
```
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── handler/
│   │   └── sync_handler.go   # HTTP request handlers
│   ├── models/
│   │   └── requests.go       # Request/response models
│   ├── server/
│   │   └── server.go         # HTTP server setup
│   ├── service/
│   │   └── sync_service.go   # Business logic
│   ├── syncer/
│   │   ├── git/
│   │   │   └── git_syncer.go # Git synchronization
│   │   ├── http/
│   │   │   └── http_syncer.go # HTTP download
│   │   ├── s3/
│   │   │   └── s3_syncer.go  # S3 synchronization
│   │   ├── ssh/
│   │   │   └── ssh_syncer.go # SSH synchronization
│   │   └── types.go          # Common types and factory
│   └── utils/
│       └── fs.go             # File system utilities
├── pkg/
│   └── errors/
│       └── errors.go         # Custom error types
├── Dockerfile                # Container definition
├── docker-compose.yml        # Docker Compose configuration
├── kubernetes/               # Kubernetes manifests
│   ├── deployment.yaml       # Production deployment
│   ├── service.yaml          # Service definition
│   └── configmap.yaml        # Configuration
└── README.md                 # This documentation
```

### Adding New Source Types

1. Create a new syncer implementation in `internal/syncer/<type>/`
2. Add the new type to the models in `internal/models/requests.go`
3. Add parsing logic in `internal/syncer/types.go`
4. Update the service validation in `internal/service/sync_service.go`
5. Add comprehensive tests for the new source type
6. Update documentation and examples

## 🔧 Troubleshooting

### Common Issues

1. **SSH Connection Failed**: Verify host, port, username, and private key format
2. **Permission Denied**: Ensure private key has correct permissions and format
3. **Timeout Errors**: Increase timeout value for large data transfers
4. **Sync Already in Progress**: Wait for current sync to complete or check logs
5. **Kubernetes Health Check Failures**: Verify pod readiness and resource allocation

### Kubernetes Debugging

```bash
# Check pod status
kubectl get pods -l app=volume-syncer

# View pod logs
kubectl logs -f deployment/volume-syncer

# Check health endpoint
kubectl port-forward deployment/volume-syncer 8080:8080
curl http://localhost:8080/health

# Debug volume mounts
kubectl exec -it deployment/volume-syncer -- ls -la /mnt/shared-volume
```

### Logs

Check container logs for detailed error information:
```bash
# Docker Compose
docker-compose logs -f volume-syncer

# Docker
docker logs -f <container-id>

# Kubernetes
kubectl logs -f deployment/volume-syncer
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on how to contribute to this project.

### Quick Start for Contributors

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and add tests
4. Run tests: `go test ./...`
5. Run linting: `golangci-lint run`
6. Commit your changes: `git commit -m 'Add amazing feature'`
7. Push to the branch: `git push origin feature/amazing-feature`
8. Open a Pull Request

### Development Setup

```bash
# Clone your fork
git clone https://github.com/yourusername/volume-syncer.git
cd volume-syncer

# Install dependencies
go mod download

# Run tests
go test ./...

# Build and run
go build -o volume-syncer ./cmd/server
./volume-syncer
```

For more detailed contribution guidelines, code standards, and development practices, please read our [Contributing Guide](CONTRIBUTING.md).

### Pull Request Guidelines

When submitting a pull request, please include:

**Description & Type of Change**
- Clear description of what the PR does
- Specify if it's a bug fix, new feature, breaking change, documentation update, etc.

**Changes Made**
- List specific changes made
- Reference affected files or modules
- Include any new dependencies or configuration changes

**Testing**
- Add tests for new functionality
- Ensure all existing tests pass
- Include manual testing steps if applicable

**Example Manual Testing:**
```bash
# Test new sync source
curl -X POST http://localhost:8080/api/1.0/sync \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "your-new-source",
      "details": {
        "host": "test.example.com",
        "username": "testuser"
      }
    },
    "target": {
      "path": "/mnt/test-volume"
    }
  }'
```

**Checklist**
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings generated
- [ ] CHANGELOG.md updated

**Related Issues**
- Reference any related issues (e.g., "Fixes #123", "Closes #456")

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

**Why Apache 2.0?**
- Enterprise-friendly license allowing commercial use
- Compatible with most open source and proprietary projects
- Provides patent protection for users
- Industry standard for enterprise Kubernetes operators and tools