# Build stage
FROM golang:1.24-alpine AS builder

# Build arguments for metadata
ARG BUILD_DATE
ARG GIT_COMMIT
ARG VERSION=0.0.8

# Add metadata labels
LABEL maintainer="Bilgehan NAL <bilgehan.nal@gmail.com>"
LABEL org.opencontainers.image.title="Volume Syncer"
LABEL org.opencontainers.image.description="A high-performance API server for synchronizing data from various sources (SSH, Git, S3) to local volumes"
LABEL org.opencontainers.image.url="https://github.com/sharedvolume/volume-syncer"
LABEL org.opencontainers.image.source="https://github.com/sharedvolume/volume-syncer"
LABEL org.opencontainers.image.vendor="SharedVolume"
LABEL org.opencontainers.image.licenses="MIT"

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o volume-syncer ./cmd/server

# Runtime stage
FROM alpine:3.18

# Build arguments for metadata
ARG BUILD_DATE
ARG GIT_COMMIT
ARG VERSION=0.0.8

# Add runtime labels
LABEL maintainer="Bilgehan NAL <bilgehan.nal@gmail.com>"
LABEL org.opencontainers.image.title="Volume Syncer"
LABEL org.opencontainers.image.description="A high-performance API server for synchronizing data from various sources (SSH, Git, S3) to local volumes"
LABEL org.opencontainers.image.url="https://github.com/sharedvolume/volume-syncer"
LABEL org.opencontainers.image.source="https://github.com/sharedvolume/volume-syncer"
LABEL org.opencontainers.image.vendor="SharedVolume"
LABEL org.opencontainers.image.licenses="MIT"

# Install runtime dependencies and create user in single layer
RUN apk add --no-cache \
    ca-certificates \
    git \
    openssh-client \
    rsync \
    sshpass \
    wget \
    && rm -rf /var/cache/apk/* \
    && addgroup -g 1001 -S appgroup \
    && adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/volume-syncer .

# Create volume mount point
RUN mkdir -p /mnt/shared-volume && \
    chown -R appuser:appgroup /mnt/shared-volume

# Change to non-root user
USER root:root

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./volume-syncer"]
