# Contributing to IGScraper

Thank you for your interest in contributing to IGScraper! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Security](#security)

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to [conduct@example.com].

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally
3. Create a new branch for your feature or bug fix
4. Make your changes
5. Push to your fork and submit a pull request

## Development Setup

### Prerequisites

- Go 1.23 or higher
- Docker and Docker Compose (optional, for containerized development)
- Make (optional but recommended)
- Pre-commit (optional but recommended)

### Setting Up Your Development Environment

```bash
# Clone your fork
git clone https://github.com/yourusername/igscraper.git
cd igscraper

# Install dependencies
make deps

# Set up development tools
make setup

# Run tests to ensure everything is working
make test
```

### Using Docker for Development

```bash
# Build the Docker image
make docker-build

# Run with docker-compose
docker-compose up

# Or run a specific user
docker-compose run --rm igscraper username

# Run tests in Docker
docker run --rm -v $(pwd):/app -w /app golang:1.23-alpine go test ./...
```

## How to Contribute

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- A clear and descriptive title
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- System information (OS, Go version, etc.)
- Relevant logs or error messages

### Suggesting Enhancements

Enhancement suggestions are welcome! Please provide:

- A clear and descriptive title
- A detailed description of the proposed enhancement
- Why this enhancement would be useful
- Possible implementation approach

### Submitting Code

1. **Small Changes**: For small changes (typos, minor bug fixes), you can submit a PR directly.

2. **Large Changes**: For significant changes, please open an issue first to discuss the proposed changes.

## Pull Request Process

1. **Branch Naming**: Use descriptive branch names:
   - `feature/add-proxy-support`
   - `fix/rate-limit-bug`
   - `docs/update-readme`
   - `chore/update-dependencies`

2. **Commit Messages**: Follow conventional commits:
   ```
   type(scope): subject
   
   body
   
   footer
   ```
   
   Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`

3. **Pre-commit Checks**: Ensure all pre-commit hooks pass:
   ```bash
   pre-commit run --all-files
   ```

4. **Testing**: Add tests for new features and ensure all tests pass:
   ```bash
   make test
   make test-coverage
   ```

5. **Documentation**: Update documentation for any user-facing changes.

6. **PR Description**: Provide a clear description of what the PR does and why.

## Coding Standards

### Go Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` and `goimports` for formatting
- Run `golangci-lint` before submitting:
  ```bash
  make lint
  ```

### Code Organization

- Keep packages focused and cohesive
- Use meaningful package and file names
- Follow the project's existing structure

### Error Handling

- Always handle errors explicitly
- Use custom error types when appropriate
- Provide context with errors using `fmt.Errorf` with `%w`

### Logging

- Use the project's logger (zerolog)
- Use appropriate log levels
- Include relevant context in log messages

### Comments and Documentation

- Write clear, concise comments
- Document all exported functions, types, and packages
- Use examples in documentation when helpful

## Testing

### Unit Tests

- Write unit tests for all new functionality
- Aim for high test coverage (>80%)
- Use table-driven tests where appropriate
- Mock external dependencies

### Integration Tests

- Write integration tests for critical paths
- Use build tags for integration tests:
  ```go
  //go:build integration
  ```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run integration tests
make test-integration

# Run benchmarks
make benchmark
```

## Documentation

- Update README.md for user-facing changes
- Add/update code comments and package documentation
- Include examples for complex features
- Update configuration documentation

## Security

- Never commit credentials or sensitive data
- Report security vulnerabilities privately
- Follow secure coding practices
- Use the latest dependencies

## Release Process

1. Ensure all tests pass
2. Update CHANGELOG.md
3. Create a new tag following semantic versioning
4. The CI/CD pipeline will handle the rest

## Getting Help

- Check the documentation first
- Search existing issues
- Ask in discussions
- Join our community chat (if available)

## Recognition

Contributors will be recognized in:
- The project's contributors list
- Release notes for significant contributions
- The project README (for major contributors)

Thank you for contributing to IGScraper!