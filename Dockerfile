# Multi-stage build for smaller image
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with proper flags for static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o igscraper ./cmd/igscraper

# Final stage - minimal image
FROM scratch

# Copy timezone data for proper time handling
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy SSL certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy user from builder
COPY --from=builder /etc/passwd /etc/passwd

# Copy the binary
COPY --from=builder /build/igscraper /igscraper

# Create directories for downloads and config
VOLUME ["/downloads", "/config"]

# Switch to non-root user
USER appuser

# Set environment variables
ENV IGSCRAPER_DOWNLOAD_DIR=/downloads
ENV IGSCRAPER_CONFIG_DIR=/config

# Expose any ports if needed (not required for this CLI tool)
# EXPOSE 8080

ENTRYPOINT ["/igscraper"]