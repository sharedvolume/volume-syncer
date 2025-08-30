# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Which versions are eligible for receiving such patches depends on the CVSS v3.0 Rating:

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability within Volume Syncer, please send an email to security@example.com. All security vulnerabilities will be promptly addressed.

Please include the following information in your report:

- Type of issue (e.g. buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit the issue

## Security Considerations

### Authentication & Authorization

- The API currently does not implement authentication or authorization
- It is recommended to run Volume Syncer behind a reverse proxy with proper authentication
- Consider implementing API keys or JWT tokens for production deployments

### SSH Security

- SSH private keys are handled securely in memory
- Temporary SSH key files are created with restrictive permissions (600)
- SSH host key verification is disabled by default - configure properly for production
- Private keys are cleaned up after use

### Data Handling

- Sensitive configuration data should be passed via environment variables
- Avoid logging sensitive information
- Container runs as non-root user by default

### Network Security

- The service listens on all interfaces (0.0.0.0) by default
- Consider binding to localhost or specific interfaces in production
- Use HTTPS/TLS when exposing the service publicly
- Implement proper firewall rules

### Container Security

- The Docker image runs as a non-root user
- Only necessary packages are included in the image
- Regular base image updates are recommended

### Input Validation

- All API inputs are validated before processing
- File paths are sanitized to prevent directory traversal
- URLs are validated to prevent SSRF attacks

## Best Practices for Deployment

1. **Network Isolation**: Deploy in a private network when possible
2. **Access Control**: Implement proper authentication and authorization
3. **Monitoring**: Set up logging and monitoring for security events
4. **Updates**: Keep the application and its dependencies up to date
5. **Secrets Management**: Use proper secrets management for sensitive data
6. **Backup**: Implement secure backup procedures for critical data

## Acknowledgments

We would like to thank the following individuals for their responsible disclosure of security vulnerabilities:

- None yet, but we appreciate security researchers who help keep our project secure!

## Security Updates

Security updates will be announced through:

- GitHub Security Advisories
- Release notes
- Project documentation

## Contact

For security-related questions that are not vulnerabilities, please open a GitHub issue or contact the maintainers.