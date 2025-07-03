# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o igscraper ./cmd/igscraper

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S igscraper && \
    adduser -u 1000 -S igscraper -G igscraper

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/igscraper .

# Create directories for downloads and config
RUN mkdir -p /downloads /config && \
    chown -R igscraper:igscraper /app /downloads /config

# Switch to non-root user
USER igscraper

# Set environment variables
ENV IGSCRAPER_OUTPUT_BASE_DIRECTORY=/downloads

# Volume for downloads and config
VOLUME ["/downloads", "/config"]

# Default command
ENTRYPOINT ["./igscraper"]
CMD ["--help"]