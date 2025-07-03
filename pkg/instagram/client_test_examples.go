package instagram

// This file contains examples of how to properly mock the Instagram client for testing.
// The key insight is that we should mock at the HTTP transport level rather than
// trying to override the baseURL, since the URL construction functions use constants.

/*
Example 1: Basic mocking with mockRoundTripper

func TestMyFeature(t *testing.T) {
    log := logger.NewTestLogger()
    
    // Create a mock HTTP client that intercepts requests
    mockClient := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
        // Check the URL and return appropriate responses
        if req.URL.String() == GetProfileURL("testuser") {
            response := &InstagramResponse{
                Status: "ok",
                Data: Data{
                    User: User{
                        ID: "123456",
                    },
                },
            }
            responseBody, _ := json.Marshal(response)
            return &http.Response{
                StatusCode: http.StatusOK,
                Body:       io.NopCloser(bytes.NewReader(responseBody)),
                Header:     make(http.Header),
            }, nil
        }
        return newResponse(http.StatusNotFound, ""), nil
    })
    
    // Create Instagram client and inject the mock HTTP client
    client := NewClient(30*time.Second, log)
    client.httpClient = mockClient
    
    // Now all requests will go through the mock
    result, err := client.FetchUserProfile("testuser")
    // ... assertions ...
}

Example 2: Using the helper function for cleaner tests

func TestMyFeature(t *testing.T) {
    log := logger.NewTestLogger()
    
    // Define expected responses for different URLs
    responses := map[string]interface{}{
        GetProfileURL("testuser"): &InstagramResponse{
            Status: "ok",
            Data: Data{
                User: User{
                    ID: "123456",
                },
            },
        },
        GetMediaURL("123456", ""): &InstagramResponse{
            Status: "ok",
            Data: Data{
                User: User{
                    EdgeOwnerToTimelineMedia: EdgeOwnerToTimelineMedia{
                        Edges: []Edge{
                            {
                                Node: Node{
                                    ID:         "media1",
                                    Shortcode:  "ABC123",
                                    DisplayURL: "https://example.com/photo.jpg",
                                },
                            },
                        },
                    },
                },
            },
        },
        // Return just a status code for specific URLs
        "https://example.com/photo.jpg": http.StatusOK,
        // Return an error for specific URLs
        "https://example.com/error": errors.New("network error"),
    }
    
    // Create client with predefined responses
    client := newTestClient(log, responses)
    
    // All requests matching the URLs above will return the mocked responses
    profile, _ := client.FetchUserProfile("testuser")
    media, _ := client.FetchUserMedia("123456", "")
    // ... assertions ...
}

Example 3: Testing error scenarios

func TestErrorHandling(t *testing.T) {
    log := logger.NewTestLogger()
    
    t.Run("rate limit error", func(t *testing.T) {
        client := newTestClient(log, map[string]interface{}{
            GetProfileURL("limited"): http.StatusTooManyRequests,
        })
        
        _, err := client.FetchUserProfile("limited")
        assert.Error(t, err)
        
        var igErr *errors.Error
        assert.ErrorAs(t, err, &igErr)
        assert.Equal(t, errors.ErrorTypeRateLimit, igErr.Type)
    })
    
    t.Run("authentication required", func(t *testing.T) {
        client := newTestClient(log, map[string]interface{}{
            GetProfileURL("private"): &InstagramResponse{
                RequiresToLogin: true,
            },
        })
        
        _, err := client.FetchUserProfile("private")
        assert.Error(t, err)
        
        var igErr *errors.Error
        assert.ErrorAs(t, err, &igErr)
        assert.Equal(t, errors.ErrorTypeAuth, igErr.Type)
    })
}

Key Benefits of This Approach:
1. No need to modify production code (endpoints.go remains unchanged)
2. Complete control over HTTP responses for testing
3. No real network calls are made
4. Easy to test error scenarios and edge cases
5. Tests are isolated and deterministic
*/