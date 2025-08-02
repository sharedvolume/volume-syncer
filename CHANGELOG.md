# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Professional open source repository structure
- MIT License
- Comprehensive documentation
- GitHub Actions for CI/CD
- Automated Docker image builds on tag push
- Issue and PR templates
- Contributing guidelines
- Security policy
- Code of conduct

### Changed
- Enhanced README with detailed contribution guidelines
- Improved project documentation

### Fixed
- Enhanced .gitignore with better coverage
- Professional .dockerignore

## [0.0.8] - 2025-08-02

### Added
- Initial release of Volume Syncer
- Support for SSH, Git, HTTP, and S3 sync sources
- REST API for triggering sync operations
- Docker support with multi-architecture builds
- Health check endpoint
- Comprehensive API documentation

### Features
- SSH sync with private key and password authentication
- Git repository cloning and syncing
- HTTP/HTTPS file downloads
- S3 bucket synchronization
- Concurrent operation safety
- Configurable timeouts
- Extensive logging and monitoring

### Security
- Secure handling of SSH private keys
- Non-root container execution
- Input validation and sanitization
- Temporary file cleanup
