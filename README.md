# Volume Syncer

A high-performance API server for synchronizing data from various sources (SSH, Git, S3) to local volumes. Built with Go and Gin framework, designed for containerized environments.

## Features

- **REST API**: Simple HTTP API for triggering sync operations
- **SSH Support**: Sync from remote servers using SSH with private key authentication
- **Git Support**: Clone and sync from Git repositories
- **HTTP Support**: Download files from HTTP/HTTPS URLs
- **S3 Support**: Sync data from AWS S3 or S3-compatible storage
- **Extensible Design**: Easy to add support for additional sources
- **Concurrent Safety**: Prevents multiple sync operations from running simultaneously
- **Health Checks**: Built-in health endpoint for monitoring
- **Dockerized**: Ready-to-deploy container with all dependencies
- **Timeout Support**: Configurable timeouts for sync operations

## API Endpoints

### Health Check
```
GET /health
```
Returns server health status.

### Sync Data
```
POST /api/1.0/sync
```

Request payload:
```json
{
  "source": {
    "type": "ssh",
    "details": {
      "host": "sshServer",
      "port": 22,
      "username": "pdm",
      "path": "/opt/data",
      "privateKey": "<base64 encoded private key>"
    }
  },
  "target": {
    "path": "/Users/bilgehan.nal/Desktop/test/cp/repo"
  },
  "timeout": "30s"
}
```

Response codes:
- `201`: Sync started successfully
- `400`: Invalid request format or parameters
- `503`: Sync already in progress

## Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone <repository-url>
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

### Using Docker

1. Build the image:
```bash
docker build -t volume-syncer .
```

2. Run the container:
```bash
docker run -p 8080:8080 -v $(pwd)/data:/mnt/shared-volume volume-syncer
```

### Local Development

1. Install dependencies:
```bash
go mod tidy
```

2. Run the server:
```bash
go run main.go
```

## Configuration

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

- `GIN_MODE`: Set to "release" for production
- `PORT`: Server port (default: 8080)

## Example Usage

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

## Security Considerations

- Private keys are handled securely in memory and temporary files
- Temporary SSH key files are created with restrictive permissions (600)
- SSH host key verification is disabled (configure properly for production)
- Container runs as non-root user
- All sensitive data is cleaned up after use

## Monitoring

The server provides a health check endpoint at `/health` that returns:
```json
{
  "status": "healthy",
  "timestamp": "2025-07-02T10:30:00Z"
}
```

## Development

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
└── README.md                 # This file
```

### Adding New Source Types

1. Create a new syncer implementation in `internal/syncer/<type>/`
2. Add the new type to the models in `internal/models/requests.go`
3. Add parsing logic in `internal/syncer/types.go`
4. Update the service validation in `internal/service/sync_service.go`

## Troubleshooting

### Common Issues

1. **SSH Connection Failed**: Verify host, port, username, and private key
2. **Permission Denied**: Ensure private key has correct permissions
3. **Timeout**: Increase timeout value for large data transfers
4. **Sync Already in Progress**: Wait for current sync to complete

### Logs

Check container logs for detailed error information:
```bash
docker-compose logs -f volume-syncer
```

## License

MIT License - see LICENSE file for details.