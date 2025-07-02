# Integration Tests for Instagram Scraper

This directory contains comprehensive integration tests for the Instagram Scraper, including a mock Instagram API server and realistic test scenarios.

## Structure

```
integration/
├── mock_server.go      # Mock Instagram API server implementation
├── test_utils.go       # Common test utilities and helpers
├── scraper_test.go     # Main integration test suite
├── run_tests.sh        # Test runner script
├── fixtures/           # Test data fixtures
│   ├── profile_*.json  # Profile response fixtures
│   └── media_*.json    # Media pagination fixtures
└── README.md          # This file
```

## Features Tested

### 1. Full Download Flow
- Complete user profile scraping
- Photo downloads with proper file naming
- Directory structure creation
- Metadata saving

### 2. Rate Limit Handling
- Proper retry behavior on 429 responses
- Exponential backoff implementation
- Recovery after rate limits

### 3. Error Recovery
- 401 Unauthorized handling
- 404 Not Found handling
- 500 Server Error with retries
- Network failure recovery

### 4. Checkpoint/Resume
- Progress saving during downloads
- Resume from interrupted state
- Avoiding duplicate downloads

### 5. Concurrent Downloads
- Parallel photo downloads
- Worker pool management
- Performance optimization

### 6. Authentication Flow
- Session cookie handling
- Private profile detection
- Login simulation

### 7. Edge Cases
- Empty/invalid usernames
- Large accounts with pagination
- Video skipping
- Multiple user downloads

## Running Tests

### Quick Run
```bash
./run_tests.sh
```

### With Benchmarks
```bash
./run_tests.sh --bench
```

### Specific Test
```bash
go test -v -run TestFullDownloadFlow
```

### With Coverage
```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Mock Server Features

The mock Instagram API server (`mock_server.go`) provides:

1. **Realistic Endpoints**
   - `/api/v1/users/web_profile_info/` - User profile data
   - `/graphql/query/` - Media pagination
   - `/photos/` - Photo downloads
   - `/accounts/login/` - Authentication

2. **Error Simulation**
   - Configurable error responses per endpoint
   - Rate limiting (every 10th request)
   - Network delays
   - Server errors

3. **State Tracking**
   - Request counting
   - Rate limit hit tracking
   - Pagination state (checkpoints)

## Test Fixtures

Fixtures simulate various Instagram API responses:

- `profile_default.json` - Generic user profile
- `profile_testuser.json` - Test user with 25 photos
- `profile_private.json` - Private account requiring auth
- `media_*.json` - Pagination responses

## Writing New Tests

To add new integration tests:

1. Create test fixtures if needed:
```go
// In fixtures/profile_newuser.json
{
  "data": {
    "user": {
      "id": "newuser123",
      // ... rest of response
    }
  }
}
```

2. Use the test helper:
```go
func TestNewFeature(t *testing.T) {
    helper := NewTestHelper(t)
    defer helper.Cleanup()
    
    mockServer := helper.SetupMockServer()
    cfg := helper.CreateTestConfig()
    
    // Your test logic here
}
```

3. Configure mock server behavior:
```go
// Simulate errors
mockServer.SetErrorResponse("/api/v1/users/web_profile_info/erroruser", 500)

// Add delays
mockServer.SetDelay("/photos/slow.jpg", 2*time.Second)

// Check counters
requests := mockServer.GetRequestCount()
rateLimits := mockServer.GetRateLimitHits()
```

## Test Utilities

The `test_utils.go` file provides helpful assertion methods:

- `AssertFileExists(path)` - Verify file creation
- `AssertDirContainsFiles(dir, count)` - Check file count
- `AssertError(err)` - Verify error occurred
- `AssertNoError(err)` - Verify success
- `CreateCheckpoint()` - Create test checkpoints
- `WaitForCondition()` - Wait for async operations

## Continuous Integration

These tests are designed to run in CI environments:

```yaml
# Example GitHub Actions workflow
- name: Run Integration Tests
  run: |
    cd tests/integration
    ./run_tests.sh
```

## Troubleshooting

### Tests Failing

1. Check fixture files exist in `fixtures/`
2. Ensure proper Go module setup (`go mod tidy`)
3. Verify no port conflicts for mock server

### Debugging

Set verbose logging:
```go
log := logger.NewLogger(logger.LevelDebug)
```

### Performance

For performance testing:
```bash
go test -bench=. -benchmem -cpuprofile=cpu.prof
go tool pprof cpu.prof
```