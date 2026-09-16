# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability within ibkrapi4go, please report it
privately using [GitHub Security Advisories](https://github.com/shing1211/ibkrapi4go/security/advisories/new),
or by email to **shing1211@users.noreply.github.com**. All security
vulnerabilities will be promptly addressed.

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
- **HTTPS**: All API calls use HTTPS. The local gateway uses a self-signed certificate; enabling `WithInsecureSkipVerify` is only safe against `localhost`.
- **Tokens**: Authentication tokens are stored in memory only, never persisted to disk.
- **Logging**: Tokens, cookies, and account credentials are redacted from logs and errors.
- **Rate limiting**: Built-in rate limiting prevents accidental credential lockout.

See [docs/design/08-concurrency.md](./docs/design/08-concurrency.md) for the
concurrency and secrets model.
