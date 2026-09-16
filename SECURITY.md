# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability within ibkr-sdk, please send an email to the project maintainer. All security vulnerabilities will be promptly addressed.

**Please do not report security vulnerabilities through public GitHub issues.**

### What to include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response timeline

- **Acknowledgment**: within 48 hours
- **Initial assessment**: within 1 week
- **Fix or mitigation**: depends on severity

## Security Considerations

This SDK connects to IBKR's Client Portal Gateway. Key security notes:

- **Credentials**: Never hardcode API secrets or passwords. Use environment variables.
- **HTTPS**: All API calls use HTTPS. The local gateway uses a self-signed certificate.
- **Tokens**: Authentication tokens are stored in memory only, never persisted to disk.
- **Rate limiting**: Built-in rate limiting prevents accidental credential lockout.
