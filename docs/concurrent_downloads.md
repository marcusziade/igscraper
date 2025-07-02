# Concurrent Downloads

The Instagram Scraper now supports concurrent downloads with a configurable worker pool, significantly improving download performance while respecting rate limits.

## Features

### Worker Pool Implementation
- **Configurable Workers**: Set the number of concurrent download workers (default: 3)
- **Job Queue**: Efficient job distribution with buffered channels
- **Result Processing**: Asynchronous result handling with progress tracking
- **Graceful Shutdown**: Ensures all queued jobs complete before shutdown

### Rate Limiting Integration
- **Shared Rate Limiter**: All workers share the same rate limiter
- **Automatic Throttling**: Workers wait when rate limit is reached
- **API vs Download Limits**: Separate handling for API calls and downloads

### Error Handling
- **Per-Job Error Handling**: Failed downloads don't affect other jobs
- **Retry Logic**: Built-in retry mechanism for transient failures
- **Detailed Logging**: Track each download's status and duration

### Duplicate Detection
- **Concurrent-Safe**: Thread-safe duplicate detection
- **Skip Downloaded**: Already downloaded photos are skipped efficiently
- **Memory Efficient**: In-memory cache with file system verification

## Configuration

```yaml
download:
  concurrent_downloads: 5  # Number of worker threads
  download_timeout: 30s    # Timeout per download
  retry_attempts: 3        # Retries for failed downloads

rate_limit:
  requests_per_minute: 60  # Shared across all workers
```

## Performance Benefits

With concurrent downloads enabled:
- **5x Faster**: With 5 workers, download time reduced by up to 80%
- **Efficient**: Better utilization of network bandwidth
- **Scalable**: Adjust workers based on your connection speed
- **Safe**: Respects Instagram's rate limits

## Example Usage

```go
// Configure for optimal performance
cfg := config.DefaultConfig()
cfg.Download.ConcurrentDownloads = 5  // Use 5 workers
cfg.RateLimit.RequestsPerMinute = 60  // Stay within limits

// Create scraper
scraper, _ := scraper.New(cfg)

// Downloads now use concurrent workers automatically
scraper.DownloadUserPhotos("username")
```

## Technical Details

### Worker Pool Architecture
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Scraper   │────▶│  Job Queue  │────▶│  Worker 1   │
└─────────────┘     └─────────────┘     └─────────────┘
                            │            ┌─────────────┐
                            ├───────────▶│  Worker 2   │
                            │            └─────────────┘
                            │            ┌─────────────┐
                            └───────────▶│  Worker N   │
                                         └─────────────┘
                                                │
                                                ▼
                                         ┌─────────────┐
                                         │   Results   │
                                         └─────────────┘
```

### Best Practices

1. **Worker Count**: Start with 3-5 workers, adjust based on performance
2. **Rate Limits**: Keep requests per minute reasonable (60-100)
3. **Timeout**: Set appropriate timeout based on file sizes
4. **Monitoring**: Watch logs for rate limit warnings

### Troubleshooting

**High Memory Usage**
- Reduce concurrent downloads
- Check for very large photos

**Rate Limit Errors**
- Reduce workers or requests per minute
- Add longer delays between batches

**Connection Errors**
- Check network stability
- Reduce concurrent connections
- Increase timeout values