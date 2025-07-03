# Security Policy

## Security Best Practices

When using this Instagram scraper, please follow these security best practices:

### Credential Security
- Never hardcode credentials in your code
- Use environment variables or secure credential stores
- Rotate credentials regularly

### Network Security
- Use HTTPS connections only
- Consider using a VPN for additional privacy

## Security Features

This project implements several security features:

1. **Encrypted Credential Storage**: Credentials can be stored encrypted using various backends
2. **Non-Root Container**: Docker images run as non-root user
3. **Minimal Dependencies**: Only essential dependencies are included
4. **Input Validation**: All user inputs are validated and sanitized
5. **Secure Communication**: All API calls use HTTPS
