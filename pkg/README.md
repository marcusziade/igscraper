# Instagram Scraper Packages

This directory contains the core packages that power the Instagram scraper application.

## Package Structure

### `/pkg/scraper`
The main orchestration package that coordinates the download process.

- **scraper.go**: Core scraper implementation
- **doc.go**: Package documentation
- **example_test.go**: Usage examples

Key features:
- Orchestrates photo downloads from Instagram profiles
- Integrates with Instagram API client
- Manages rate limiting and retries
- Provides progress tracking

### `/pkg/storage`
Handles file system operations and duplicate detection.

- **manager.go**: Storage manager implementation
- **doc.go**: Package documentation
- **manager_test.go**: Unit tests

Key features:
- Atomic file writes to prevent corruption
- Duplicate detection with in-memory cache
- Thread-safe operations
- Automatic directory creation

### `/pkg/ratelimit`
Provides rate limiting algorithms to prevent API abuse.

- **limiter.go**: Rate limiter implementations
- **doc.go**: Package documentation
- **limiter_test.go**: Unit tests

Implementations:
- **Token Bucket**: Fixed capacity with periodic refill
- **Sliding Window**: Request tracking over time window

### `/pkg/instagram`
Instagram API models and client (existing package).

- **models.go**: API response models
- **client.go**: HTTP client wrapper
- **endpoints.go**: API endpoint definitions

### `/pkg/ui`
User interface components (existing package).

- **terminal.go**: Terminal output formatting
- **progress.go**: Progress tracking
- **notifications.go**: System notifications

## Usage Example

```go
package main

import (
    "net/http"
    "time"
    "igscraper/pkg/scraper"
)

func main() {
    client := &http.Client{
        Timeout: 30 * time.Second,
    }
    
    headers := http.Header{
        // Configure headers with authentication
    }
    
    s, err := scraper.New(client, headers, "output_dir")
    if err != nil {
        panic(err)
    }
    
    err = s.DownloadUserPhotos("instagram_username")
    if err != nil {
        panic(err)
    }
}
```

## Testing

Run tests for individual packages:

```bash
go test ./pkg/ratelimit -v
go test ./pkg/storage -v
go test ./pkg/scraper -v
```

Run all package tests:

```bash
go test ./pkg/... -v
```