package instagram

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"igscraper/pkg/config"
	"igscraper/pkg/errors"
	"igscraper/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRoundTripper allows us to intercept HTTP requests
type mockRoundTripper struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

// Helper function to create a mock HTTP client
func newMockHTTPClient(handler func(req *http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{
		Transport: &mockRoundTripper{handler: handler},
		Timeout:   30 * time.Second,
	}
}

// Helper function to create a response
func newResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

// Helper function to create a mock client with predefined responses
func newTestClient(log logger.Logger, responses map[string]interface{}) *Client {
	mockHTTPClient := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		if response, exists := responses[req.URL.String()]; exists {
			switch v := response.(type) {
			case error:
				return nil, v
			case int:
				// Just status code
				return newResponse(v, ""), nil
			default:
				// Assume it's a struct to be JSON encoded
				responseBody, _ := json.Marshal(v)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(responseBody)),
					Header:     make(http.Header),
				}, nil
			}
		}
		// Default to 404 for unmatched URLs
		return newResponse(http.StatusNotFound, ""), nil
	})
	
	client := NewClient(30*time.Second, log)
	client.httpClient = mockHTTPClient
	return client
}

func TestNewClient(t *testing.T) {
	log := logger.NewTestLogger()
	client := NewClient(30*time.Second, log)

	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.headers)
	assert.Equal(t, BaseURL, client.baseURL)
	assert.Equal(t, log, client.logger)
	assert.NotNil(t, client.retrier)
}

func TestNewClientWithConfig(t *testing.T) {
	log := logger.NewTestLogger()
	
	t.Run("with retry enabled", func(t *testing.T) {
		retryConfig := &config.RetryConfig{
			Enabled:     true,
			MaxAttempts: 5,
		}
		client := NewClientWithConfig(30*time.Second, retryConfig, log)
		
		assert.NotNil(t, client)
		assert.NotNil(t, client.retrier)
		assert.Equal(t, retryConfig, client.retryConfig)
	})

	t.Run("with retry disabled", func(t *testing.T) {
		retryConfig := &config.RetryConfig{
			Enabled:     false,
			MaxAttempts: 5,
		}
		client := NewClientWithConfig(30*time.Second, retryConfig, log)
		
		assert.NotNil(t, client)
		assert.NotNil(t, client.retrier)
	})

	t.Run("with nil config", func(t *testing.T) {
		client := NewClientWithConfig(30*time.Second, nil, log)
		
		assert.NotNil(t, client)
		assert.NotNil(t, client.retrier)
		assert.Nil(t, client.retryConfig)
	})
}

func TestSetHeaders(t *testing.T) {
	client := NewClient(30*time.Second, logger.NewTestLogger())
	
	t.Run("SetHeader", func(t *testing.T) {
		client.SetHeader("X-Custom-Header", "test-value")
		assert.Equal(t, "test-value", client.headers["X-Custom-Header"])
	})
	
	t.Run("SetHeaders", func(t *testing.T) {
		headers := map[string]string{
			"X-Header-1": "value1",
			"X-Header-2": "value2",
		}
		client.SetHeaders(headers)
		assert.Equal(t, "value1", client.headers["X-Header-1"])
		assert.Equal(t, "value2", client.headers["X-Header-2"])
	})
}

func TestDoRequest(t *testing.T) {
	log := logger.NewTestLogger()
	client := NewClient(30*time.Second, log)
	
	t.Run("successful request", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify headers are set
			assert.Contains(t, r.Header.Get("User-Agent"), "Mozilla")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))
		defer server.Close()
		
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		
		resp, err := client.doRequest(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "success", string(body))
		resp.Body.Close()
	})
	
	t.Run("network error", func(t *testing.T) {
		// Invalid URL to trigger network error
		req, err := http.NewRequest("GET", "http://invalid-domain-that-does-not-exist.com", nil)
		require.NoError(t, err)
		
		resp, err := client.doRequest(req)
		assert.Nil(t, resp)
		assert.Error(t, err)
		
		// Check error type
		var igErr *errors.Error
		assert.ErrorAs(t, err, &igErr)
		assert.Equal(t, errors.ErrorTypeNetwork, igErr.Type)
	})
}

func TestCheckResponseStatus(t *testing.T) {
	client := NewClient(30*time.Second, logger.NewTestLogger())
	
	tests := []struct {
		name         string
		statusCode   int
		expectedErr  error
		expectedType errors.ErrorType
	}{
		{
			name:        "200 OK",
			statusCode:  http.StatusOK,
			expectedErr: nil,
		},
		{
			name:         "401 Unauthorized",
			statusCode:   http.StatusUnauthorized,
			expectedType: errors.ErrorTypeAuth,
		},
		{
			name:         "404 Not Found",
			statusCode:   http.StatusNotFound,
			expectedType: errors.ErrorTypeNotFound,
		},
		{
			name:         "429 Too Many Requests",
			statusCode:   http.StatusTooManyRequests,
			expectedType: errors.ErrorTypeRateLimit,
		},
		{
			name:         "500 Internal Server Error",
			statusCode:   http.StatusInternalServerError,
			expectedType: errors.ErrorTypeServerError,
		},
		{
			name:         "503 Service Unavailable",
			statusCode:   http.StatusServiceUnavailable,
			expectedType: errors.ErrorTypeServerError,
		},
		{
			name:         "400 Bad Request",
			statusCode:   http.StatusBadRequest,
			expectedType: errors.ErrorTypeUnknown,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Request:    req,
			}
			
			err := client.checkResponseStatus(resp)
			if tt.expectedType == "" {
				// Expecting no error
				assert.NoError(t, err)
			} else {
				// Expecting an error
				assert.Error(t, err)
				var igErr *errors.Error
				assert.ErrorAs(t, err, &igErr)
				assert.Equal(t, tt.expectedType, igErr.Type)
				assert.Equal(t, tt.statusCode, igErr.Code)
			}
		})
	}
}

func TestGet(t *testing.T) {
	log := logger.NewTestLogger()
	client := NewClient(30*time.Second, log)
	
	t.Run("successful GET", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test response"))
		}))
		defer server.Close()
		
		resp, err := client.Get(server.URL)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "test response", string(body))
		resp.Body.Close()
	})
	
	t.Run("invalid URL", func(t *testing.T) {
		resp, err := client.Get("://invalid-url")
		assert.Nil(t, resp)
		assert.Error(t, err)
		
		var igErr *errors.Error
		assert.ErrorAs(t, err, &igErr)
		assert.Equal(t, errors.ErrorTypeUnknown, igErr.Type)
	})
}

func TestGetJSON(t *testing.T) {
	log := logger.NewTestLogger()
	client := NewClient(30*time.Second, log)
	
	type testData struct {
		Message string `json:"message"`
		Value   int    `json:"value"`
	}
	
	t.Run("successful JSON decode", func(t *testing.T) {
		expected := testData{Message: "test", Value: 42}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(expected)
		}))
		defer server.Close()
		
		var result testData
		err := client.GetJSON(server.URL, &result)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})
	
	t.Run("invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()
		
		var result testData
		err := client.GetJSON(server.URL, &result)
		assert.Error(t, err)
		
		var igErr *errors.Error
		assert.ErrorAs(t, err, &igErr)
		assert.Equal(t, errors.ErrorTypeParsing, igErr.Type)
	})
	
	t.Run("non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()
		
		var result testData
		err := client.GetJSON(server.URL, &result)
		assert.Error(t, err)
		
		var igErr *errors.Error
		assert.ErrorAs(t, err, &igErr)
		assert.Equal(t, errors.ErrorTypeNotFound, igErr.Type)
	})
}

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
	
	t.Run("requires login", func(t *testing.T) {
		response := &InstagramResponse{
			RequiresToLogin: true,
		}
		
		// Create a mock HTTP client
		mockClient := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			expectedURL := GetProfileURL("privateuser")
			if req.URL.String() == expectedURL {
				responseBody, _ := json.Marshal(response)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(responseBody)),
					Header:     make(http.Header),
				}, nil
			}
			return newResponse(http.StatusBadRequest, ""), nil
		})
		
		// Create client with mock HTTP client
		client := NewClient(30*time.Second, log)
		client.httpClient = mockClient
		
		result, err := client.FetchUserProfile("privateuser")
		assert.Nil(t, result)
		assert.Error(t, err)
		
		var igErr *errors.Error
		assert.ErrorAs(t, err, &igErr)
		assert.Equal(t, errors.ErrorTypeAuth, igErr.Type)
	})
}

func TestFetchUserMedia(t *testing.T) {
	log := logger.NewTestLogger()
	
	t.Run("successful media fetch", func(t *testing.T) {
		expectedResponse := &InstagramResponse{
			Status: "ok",
			Data: Data{
				User: User{
					EdgeOwnerToTimelineMedia: EdgeOwnerToTimelineMedia{
						Edges: []Edge{
							{
								Node: Node{
									ID:         "media1",
									Shortcode:  "ABC123",
									DisplayURL: "https://example.com/photo1.jpg",
									IsVideo:    false,
								},
							},
						},
						PageInfo: PageInfo{
							HasNextPage: true,
							EndCursor:   "cursor123",
						},
					},
				},
			},
		}
		
		// Create a mock HTTP client
		mockClient := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			expectedURL := GetMediaURL("123456", "")
			if req.URL.String() == expectedURL {
				responseBody, _ := json.Marshal(expectedResponse)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(responseBody)),
					Header:     make(http.Header),
				}, nil
			}
			return newResponse(http.StatusBadRequest, ""), nil
		})
		
		// Create client with mock HTTP client
		client := NewClient(30*time.Second, log)
		client.httpClient = mockClient
		
		result, err := client.FetchUserMedia("123456", "")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Data.User.EdgeOwnerToTimelineMedia.Edges, 1)
		assert.Equal(t, "ABC123", result.Data.User.EdgeOwnerToTimelineMedia.Edges[0].Node.Shortcode)
	})
}

func TestDownloadPhoto(t *testing.T) {
	log := logger.NewTestLogger()
	client := NewClient(30*time.Second, log)
	
	t.Run("successful download", func(t *testing.T) {
		expectedData := []byte("fake image data")
		
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			w.Write(expectedData)
		}))
		defer server.Close()
		
		data, err := client.DownloadPhoto(server.URL + "/photo.jpg")
		require.NoError(t, err)
		assert.Equal(t, expectedData, data)
	})
	
	t.Run("download error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()
		
		data, err := client.DownloadPhoto(server.URL + "/notfound.jpg")
		assert.Nil(t, data)
		assert.Error(t, err)
		
		var igErr *errors.Error
		assert.ErrorAs(t, err, &igErr)
		assert.Equal(t, errors.ErrorTypeNotFound, igErr.Type)
	})
}

func TestDoRequestWithRetry(t *testing.T) {
	log := logger.NewTestLogger()
	
	t.Run("retry on server error", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success after retries"))
			}
		}))
		defer server.Close()
		
		retryConfig := &config.RetryConfig{
			Enabled:     true,
			MaxAttempts: 3,
			BaseDelay:   10 * time.Millisecond,
		}
		client := NewClientWithConfig(30*time.Second, retryConfig, log)
		
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		
		resp, err := client.doRequestWithRetry(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 3, attempts)
		resp.Body.Close()
	})
	
	t.Run("retry on rate limit", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				w.WriteHeader(http.StatusTooManyRequests)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()
		
		retryConfig := &config.RetryConfig{
			Enabled:     true,
			MaxAttempts: 3,
			BaseDelay:   10 * time.Millisecond,
		}
		client := NewClientWithConfig(30*time.Second, retryConfig, log)
		
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		
		resp, err := client.doRequestWithRetry(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 2, attempts)
		resp.Body.Close()
	})
	
	t.Run("no retry on auth error", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()
		
		retryConfig := &config.RetryConfig{
			Enabled:     true,
			MaxAttempts: 3,
			BaseDelay:   10 * time.Millisecond,
		}
		client := NewClientWithConfig(30*time.Second, retryConfig, log)
		
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		
		_, err = client.doRequestWithRetry(req)
		assert.Error(t, err)
		assert.Equal(t, 1, attempts) // Should not retry auth errors
		
		var igErr *errors.Error
		assert.ErrorAs(t, err, &igErr)
		assert.Equal(t, errors.ErrorTypeAuth, igErr.Type)
	})
}

func TestDownloadPhotoWithRetry(t *testing.T) {
	log := logger.NewTestLogger()
	
	t.Run("successful download with retries", func(t *testing.T) {
		attempts := 0
		expectedData := []byte("image data after retries")
		
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				// Simulate network error
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				w.Header().Set("Content-Type", "image/jpeg")
				w.WriteHeader(http.StatusOK)
				w.Write(expectedData)
			}
		}))
		defer server.Close()
		
		retryConfig := &config.RetryConfig{
			Enabled:          true,
			NetworkRetries:   3,
			NetworkBaseDelay: 10 * time.Millisecond,
			MaxDelay:         100 * time.Millisecond,
			Multiplier:       2.0,
			JitterFactor:     0.1,
		}
		client := NewClientWithConfig(30*time.Second, retryConfig, log)
		
		data, err := client.DownloadPhoto(server.URL + "/photo.jpg")
		require.NoError(t, err)
		assert.Equal(t, expectedData, data)
		assert.Equal(t, 2, attempts)
	})
}