# Logger Package

A comprehensive structured logging package for the Instagram Scraper using zerolog.

## Features

- **Multiple Log Levels**: Debug, Info, Warn, Error, Fatal
- **Structured Logging**: Log with fields for better searchability
- **Pretty Console Output**: Colored output with clean formatting
- **File Output Support**: Optional file logging with rotation
- **Context Support**: Integrate with Go's context for request tracing
- **Global Logger**: Easy access throughout the application
- **Type-Safe**: Strong typing for log fields
- **Performance**: Zero-allocation logging from zerolog

## Quick Start

```go
import "igscraper/pkg/logger"

// Initialize the logger
cfg := &config.LoggingConfig{
    Level: "info",
    File: "/var/log/igscraper.log",
}
err := logger.Initialize(cfg)

// Use the logger
logger.Info("Application started")
logger.WithField("username", "john_doe").Info("Processing user")
logger.WithError(err).Error("Operation failed")
```

## Configuration

The logger is configured through the `LoggingConfig` struct:

```go
type LoggingConfig struct {
    Level      string // Log level: debug, info, warn, error, fatal
    File       string // Path to log file (empty for console only)
    MaxSize    int    // Maximum size in MB before rotation
    MaxBackups int    // Number of old files to keep
    MaxAge     int    // Maximum age in days
    Compress   bool   // Compress rotated files
}
```

## Usage Examples

### Basic Logging

```go
logger.Debug("Debug message")
logger.Info("Info message")
logger.Warn("Warning message")
logger.Error("Error message")
logger.Fatal("Fatal message") // This will exit the application
```

### Logging with Fields

```go
// Single field
logger.WithField("username", "john_doe").Info("User logged in")

// Multiple fields
logger.WithFields(map[string]interface{}{
    "username": "jane_doe",
    "action": "download",
    "count": 10,
}).Info("Download completed")

// Chaining fields
logger.
    WithField("component", "downloader").
    WithField("session_id", "abc123").
    Info("Component initialized")
```

### Error Logging

```go
err := doSomething()
if err != nil {
    logger.WithError(err).Error("Operation failed")
    
    // With additional context
    logger.
        WithError(err).
        WithField("username", username).
        WithField("retry_count", 3).
        Error("Download failed after retries")
}
```

### Structured Logging

```go
logger.InfoWithFields("Operation completed", map[string]interface{}{
    "duration": time.Second * 5,
    "bytes_processed": 1048576,
    "success_rate": 0.95,
})
```

### Component Loggers

Create loggers for specific components:

```go
scraperLog := logger.WithField("component", "scraper")
downloaderLog := logger.WithField("component", "downloader")

scraperLog.Info("Starting scrape")
downloaderLog.Info("Download queued")
```

### Performance Logging

```go
start := time.Now()
// ... do work ...
duration := time.Since(start)

logger.WithFields(map[string]interface{}{
    "operation": "fetch_posts",
    "duration": duration,
    "items_processed": 100,
}).Info("Operation completed")
```

## Helper Functions

The package includes helper functions for common logging scenarios:

```go
// Log HTTP requests
logger.LogRequest("GET", "/api/users", 200, 125.5)

// Log downloads
logger.LogDownload("john_doe", "ABC123", "image", true, nil)

// Log rate limiting
logger.LogRateLimit("/api/v1/media", 30)

// Log scraping progress
logger.LogScrapeProgress("john_doe", 50, 100)

// Log component lifecycle
logger.LogComponentStart("scraper", map[string]interface{}{
    "mode": "photos",
    "concurrent": 3,
})
logger.LogComponentStop("scraper", "completed")

// Log metrics
logger.LogMetrics("download_batch", map[string]interface{}{
    "total_files": 50,
    "success": 48,
    "failed": 2,
    "duration_ms": 5000,
})
```

## Integration Guide

### In main.go

```go
// Initialize logger early in main()
if err := logger.Initialize(&cfg.Logging); err != nil {
    fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
    os.Exit(1)
}

logger.Info("Application starting")
defer logger.Info("Application shutting down")
```

### In packages

```go
import "igscraper/pkg/logger"

func SomeFunction() {
    log := logger.GetLogger().WithField("function", "SomeFunction")
    log.Debug("Starting operation")
    
    // ... do work ...
    
    log.Info("Operation completed")
}
```

### In HTTP handlers

```go
func handler(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    
    // ... handle request ...
    
    logger.LogRequest(r.Method, r.URL.Path, 200, time.Since(start).Milliseconds())
}
```

## Best Practices

1. **Initialize Early**: Initialize the logger as early as possible in your application
2. **Use Structured Fields**: Prefer structured fields over string concatenation
3. **Log Levels**: Use appropriate log levels (Debug for development, Info for important events, Error for errors)
4. **Avoid Sensitive Data**: Never log passwords, tokens, or other sensitive information
5. **Performance**: Use `WithFields` for multiple fields instead of multiple `WithField` calls
6. **Context**: Add relevant context to help with debugging
7. **Consistency**: Use consistent field names across your application

## Testing

For testing, use the no-op logger:

```go
func TestSomething(t *testing.T) {
    // Use a no-op logger for tests
    log := logger.NewNopLogger()
    
    // Your test code that uses the logger
    result := functionThatLogs(log)
    
    // Assert results
}
```

## Performance Considerations

- The logger is built on zerolog, which provides zero-allocation logging
- File output is buffered for better performance
- Use `Debug` level only in development to reduce log volume
- Consider log sampling for high-frequency events

## Troubleshooting

### Logger not initialized
If you see "logger not initialized" errors, ensure you call `logger.Initialize()` before using the logger.

### File permissions
If file logging fails, check that the application has write permissions to the log directory.

### Log level not working
Ensure the log level is set correctly in the configuration. Valid levels are: debug, info, warn, error, fatal.