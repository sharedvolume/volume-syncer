# Volume Syncer

A high-performance API server for synchronizing data from various sources (SSH, Git, S3) to local volumes. Built with Go and Gin framework, designed for containerized environments.

## Features

- **REST API**: Simple HTTP API for triggering sync operations
- **SSH Support**: Sync from remote servers using SSH with private key authentication
- **Extensible Design**: Easy to add support for Git, S3, and other sources
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
    "path": "/Users/bilgehan.nal/Desktop/test/cp"
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

Planned:
- **Git**: Clone/pull from Git repositories
- **S3**: Sync from AWS S3 buckets
- **HTTP**: Download from HTTP/HTTPS endpoints

### SSH Configuration

- `host`: SSH server hostname or IP (required)
- `port`: SSH port (default: 22)
- `username`: SSH username (default: "root")
- `privateKey`: Base64 encoded private key (required)

### Environment Variables

- `GIN_MODE`: Set to "release" for production
- `PORT`: Server port (default: 8080)

## Example Usage

### Sync from SSH Server

```bash
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
    },
    "timeout": "60s"
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
├── main.go                 # Server entry point
├── internal/syncer/        # Sync logic
│   ├── manager.go         # Request handling and validation
│   └── ssh.go             # SSH synchronization implementation
├── Dockerfile             # Container definition
├── docker-compose.yml     # Docker Compose configuration
└── README.md              # This file
```

### Adding New Source Types

1. Create a new syncer implementation in `internal/syncer/`
2. Add the new type to the validation logic in `manager.go`
3. Update the `StartSync` method to handle the new type

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