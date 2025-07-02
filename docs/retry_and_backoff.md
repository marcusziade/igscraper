# Exponential Backoff and Retry Logic

The Instagram scraper now includes sophisticated retry logic with exponential backoff to handle transient failures gracefully.

## Features

### 1. Multiple Backoff Strategies
- **Exponential Backoff**: Delays increase exponentially with each retry
- **Linear Backoff**: Delays increase linearly 
- **Constant Backoff**: Fixed delay between retries
- **Jitter**: Random variation to avoid thundering herd problems

### 2. Error-Type Specific Handling
Different error types get different retry strategies:
- **Network Errors**: Quick retries (1s base, 5 attempts)
- **Rate Limit Errors**: Longer delays (30s base, 3 attempts)
- **Server Errors (5xx)**: Moderate delays (5s base, 3 attempts)
- **Auth/Not Found Errors**: No retry (fail immediately)

### 3. Context Support
- Cancellation support via Go contexts
- Graceful shutdown during retries

### 4. Configuration
The retry behavior is fully configurable via the config file:

```yaml
retry:
  enabled: true
  max_attempts: 3
  
  # General exponential backoff settings
  base_delay: 1s
  max_delay: 60s
  multiplier: 2.0
  jitter_factor: 0.1
  
  # Error-type specific settings
  network_retries: 5
  network_base_delay: 1s
  
  rate_limit_retries: 3
  rate_limit_base_delay: 30s
  
  server_error_retries: 3
  server_error_base_delay: 5s
```

## How It Works

### Instagram Client Integration
The Instagram client automatically retries failed requests based on the error type:

```go
// Client creation with retry config
client := instagram.NewClientWithConfig(timeout, &cfg.Retry, logger)

// Automatic retry on API calls
response, err := client.FetchUserProfile(username)
// This will automatically retry on network/server errors

// Photo downloads also retry
data, err := client.DownloadPhoto(url)
// Uses network-specific retry configuration
```

### Backoff Calculation
For exponential backoff:
1. Base delay starts at configured value (e.g., 1 second)
2. Each retry multiplies delay by multiplier (e.g., 2.0)
3. Jitter adds randomness: ±10% by default
4. Delay is capped at max_delay (e.g., 60 seconds)

Example progression with 2x multiplier:
- Attempt 1: 1s (±0.1s)
- Attempt 2: 2s (±0.2s)
- Attempt 3: 4s (±0.4s)
- Attempt 4: 8s (±0.8s)
- Attempt 5: 16s (±1.6s)

### Retry Decision Logic
The system determines if an error should be retried:

1. **HTTP Status Codes**:
   - 429 (Rate Limit): Yes, with longer delays
   - 500, 502, 503, 504: Yes, server errors
   - 401, 403, 404: No, client errors
   - Network errors: Yes

2. **Error Types**:
   - `ErrorTypeNetwork`: Retry
   - `ErrorTypeRateLimit`: Retry with backoff
   - `ErrorTypeServerError`: Retry
   - `ErrorTypeAuth`: Don't retry
   - `ErrorTypeNotFound`: Don't retry

## Usage Examples

### Basic Retry
```go
err := retry.Do(func() error {
    return someOperation()
}, nil) // Uses default config
```

### Custom Configuration
```go
cfg := &retry.Config{
    MaxAttempts: 5,
    Backoff: &retry.ExponentialBackoff{
        BaseDelay:    2 * time.Second,
        MaxDelay:     30 * time.Second,
        Multiplier:   1.5,
        JitterFactor: 0.2,
    },
    RetryIf: retry.DefaultRetryIf,
    Context: ctx,
    Logger:  logger,
}

err := retry.Do(operation, cfg)
```

### With Result
```go
result, err := retry.DoWithResult(func() (string, error) {
    return fetchData()
}, cfg)
```

## Benefits

1. **Improved Reliability**: Transient failures are handled automatically
2. **Rate Limit Handling**: Respects Instagram's rate limits with appropriate delays
3. **Network Resilience**: Handles temporary network issues
4. **Configurable**: Adjust retry behavior without code changes
5. **Observable**: All retry attempts are logged for debugging

## Best Practices

1. **Don't Retry Everything**: Auth errors and 404s shouldn't be retried
2. **Use Appropriate Delays**: Rate limits need longer delays than network errors
3. **Set Reasonable Limits**: Don't retry indefinitely
4. **Monitor Logs**: Watch for excessive retries which may indicate persistent issues
5. **Test Your Config**: Ensure retry delays align with your use case

## Monitoring

The system logs all retry attempts:
```
WARN retrying operation attempt=1 error="network error" delay_ms=1000
WARN retrying operation attempt=2 error="network error" delay_ms=2000
DEBUG operation succeeded after retry attempt=3
```

This helps identify patterns and optimize retry configuration.