# Instagram Client Testing Approach

## Problem

The Instagram client tests were failing because:
1. The `GetProfileURL`, `GetMediaURL`, and other URL construction functions in `endpoints.go` use the `BaseURL` constant directly
2. Tests were trying to override `client.baseURL` to point to a test server
3. Since the URL functions don't use `client.baseURL`, they were still constructing URLs pointing to the real Instagram API

## Solution

Instead of modifying the production code (which would be a breaking change), we mock at the HTTP transport level using Go's `http.RoundTripper` interface.

### Key Components

1. **mockRoundTripper**: A custom implementation of `http.RoundTripper` that intercepts HTTP requests
2. **newMockHTTPClient**: Helper function that creates an `http.Client` with our mock transport
3. **newTestClient**: Higher-level helper that creates a fully configured test client with predefined responses

### Implementation

```go
// Mock HTTP transport
type mockRoundTripper struct {
    handler func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
    return m.handler(req)
}

// Create test client with mocked responses
client := newTestClient(log, map[string]interface{}{
    GetProfileURL("testuser"): expectedResponse,
})

// The client will intercept requests to the real Instagram URLs
// and return our mocked responses
result, err := client.FetchUserProfile("testuser")
```

### Benefits

1. **No Breaking Changes**: The production code remains unchanged
2. **Complete Control**: Tests can simulate any HTTP response, including errors
3. **No Network Calls**: Tests are fast and reliable
4. **Realistic Testing**: Tests use the actual URLs that would be used in production
5. **Easy Error Simulation**: Can easily test rate limits, auth errors, network failures, etc.

### Example Test

```go
func TestFetchUserProfile(t *testing.T) {
    log := logger.NewTestLogger()
    
    t.Run("successful profile fetch", func(t *testing.T) {
        expectedResponse := &InstagramResponse{
            Status: "ok",
            Data: Data{
                User: User{
                    ID: "123456",
                },
            },
        }
        
        // Create client with mocked responses
        client := newTestClient(log, map[string]interface{}{
            GetProfileURL("testuser"): expectedResponse,
        })
        
        result, err := client.FetchUserProfile("testuser")
        require.NoError(t, err)
        assert.Equal(t, "123456", result.Data.User.ID)
    })
}
```

## Alternative Approaches Considered

1. **Make URL functions use client.baseURL**: Would require changing all URL functions to be methods on Client, breaking the API
2. **Add baseURL parameter to URL functions**: Would make the API cumbersome and still be a breaking change
3. **Use dependency injection for URL builder**: Would add unnecessary complexity

The HTTP transport mocking approach is the most elegant solution that maintains backward compatibility while providing complete test isolation.