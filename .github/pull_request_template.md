## Description

Brief description of what this PR does. For example: "Adds support for FTP sync source" or "Fixes timeout handling in SSH connections".

## Type of Change

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Code refactoring
- [ ] Performance improvement
- [ ] Test improvements

## Changes Made

- Added FTP syncer implementation in `internal/syncer/ftp/`
- Updated request models to support FTP configuration
- Added comprehensive tests for FTP functionality
- Updated documentation with FTP examples

## Testing

- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] I have tested the changes manually

### Manual Testing Done
```bash
# Example manual test commands used
curl -X POST http://localhost:8080/api/1.0/sync \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "ftp",
      "details": {
        "host": "test-ftp.example.com",
        "username": "testuser",
        "password": "***"
      }
    },
    "target": {
      "path": "/mnt/test-volume"
    }
  }'
```

## Checklist

- [ ] My code follows the code style of this project
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] Any dependent changes have been merged and published
- [ ] I have updated the CHANGELOG.md with my changes

## Related Issues

Fixes #123
Closes #456
Related to #789

## Screenshots (if applicable)

If this change affects the UI or has visual impact, include before/after screenshots.

## Additional Notes

- This implementation uses passive FTP mode by default
- Added connection pooling for better performance
- Backwards compatible with existing API
