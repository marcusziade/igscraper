# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Which versions are eligible for receiving such patches depends on the CVSS v3.0 Rating:

| Version | Supported          |
| ------- | ------------------ |
| latest  | ✅                |
| < latest| ❌                |

## Reporting a Vulnerability

If you discover a security vulnerability within this project, please follow these steps:

1. **DO NOT** open a public issue.
2. Email us at security@example.com with:
   - A description of the vulnerability
   - Steps to reproduce the issue
   - Potential impact
   - Any suggested fixes (if available)

### What to expect:

- **Acknowledgment**: We will acknowledge receipt of your vulnerability report within 48 hours.
- **Initial Assessment**: Within 7 days, we will provide an initial assessment of the vulnerability and an expected timeline for a fix.
- **Progress Updates**: We will keep you informed about the progress of addressing the vulnerability.
- **Fix and Disclosure**: Once the vulnerability is fixed, we will coordinate with you on the disclosure timeline.

## Security Best Practices

When using this Instagram scraper, please follow these security best practices:

### Credential Security
- Never hardcode credentials in your code
- Use environment variables or secure credential stores
- Rotate credentials regularly
- Use strong, unique passwords

### Data Protection
- Store downloaded content securely
- Implement proper access controls
- Encrypt sensitive data at rest
- Regularly audit access logs

### Network Security
- Use HTTPS connections only
- Consider using a VPN for additional privacy
- Implement rate limiting to avoid detection
- Monitor for unusual network activity

### Container Security
- Run containers with non-root users
- Use minimal base images
- Regularly update base images
- Scan images for vulnerabilities

## Security Features

This project implements several security features:

1. **Encrypted Credential Storage**: Credentials can be stored encrypted using various backends
2. **Non-Root Container**: Docker images run as non-root user
3. **Minimal Dependencies**: Only essential dependencies are included
4. **Input Validation**: All user inputs are validated and sanitized
5. **Secure Communication**: All API calls use HTTPS

## Responsible Disclosure

We believe in responsible disclosure and appreciate security researchers who help us maintain the security of our project. We will publicly acknowledge your contribution after the vulnerability has been fixed (unless you prefer to remain anonymous).

## Updates and Patches

Security updates will be released as soon as possible after a vulnerability is confirmed. Users are encouraged to update to the latest version promptly.

Subscribe to our security announcements by:
- Watching this repository
- Following our security advisory feed
- Joining our mailing list (if available)