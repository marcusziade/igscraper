package instagram

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"igscraper/pkg/config"
	"igscraper/pkg/logger"
)

// TestClientLogging demonstrates the logging functionality of the Instagram client
func TestClientLogging(t *testing.T) {
	// Initialize logger with debug level to see all logs
	cfg := &config.LoggingConfig{
		Level: "debug",
	}
	log, err := logger.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create a test server to simulate Instagram API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the incoming request on server side
		t.Logf("Test server received: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/api/v1/users/web_profile_info/":
			// Simulate successful profile response
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"user":{"id":"123","username":"test"}}}`))
		case "/api/v1/users/123/media/":
			// Simulate rate limit response
			w.WriteHeader(http.StatusTooManyRequests)
		case "/api/v1/error/":
			// Simulate server error
			w.WriteHeader(http.StatusInternalServerError)
		case "/api/v1/auth/":
			// Simulate auth required
			w.WriteHeader(http.StatusUnauthorized)
		case "/api/v1/notfound/":
			// Simulate not found
			w.WriteHeader(http.StatusNotFound)
		case "/api/v1/invalid/":
			// Simulate invalid JSON response
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{invalid json`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create client with test server URL and short retry delays for testing
	retryConfig := &config.RetryConfig{
		Enabled:     true,
		MaxAttempts: 2,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    50 * time.Millisecond,
		Multiplier:  1.5,
	}
	client := NewClientWithConfig(5*time.Second, retryConfig, log)
	client.baseURL = server.URL

	t.Run("Successful Request", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/api/v1/users/web_profile_info/")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if resp != nil {
			resp.Body.Close()
		}
	})

	t.Run("Rate Limit Error", func(t *testing.T) {
		// Create a client without retries for this test
		noRetryClient := NewClient(5*time.Second, log)
		noRetryClient.baseURL = server.URL
		noRetryClient.retrier = nil
		
		resp, err := noRetryClient.Get(server.URL + "/api/v1/users/123/media/")
		if err != nil {
			t.Errorf("Expected no error from Get, got: %v", err)
		}
		if resp != nil {
			defer resp.Body.Close()
			err = noRetryClient.checkResponseStatus(resp)
			if err == nil {
				t.Error("Expected rate limit error")
			}
		}
	})

	t.Run("Server Error", func(t *testing.T) {
		// Create a client without retries for this test
		noRetryClient := NewClient(5*time.Second, log)
		noRetryClient.baseURL = server.URL
		noRetryClient.retrier = nil
		
		resp, err := noRetryClient.Get(server.URL + "/api/v1/error/")
		if err != nil {
			t.Errorf("Expected no error from Get, got: %v", err)
		}
		if resp != nil {
			defer resp.Body.Close()
			err = noRetryClient.checkResponseStatus(resp)
			if err == nil {
				t.Error("Expected server error")
			}
		}
	})

	t.Run("Auth Error", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/api/v1/auth/")
		// Auth errors are not retried, so we expect an error from Get
		if err == nil {
			t.Error("Expected auth error from Get")
		}
		if resp != nil {
			resp.Body.Close()
		}
	})

	t.Run("Not Found Error", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/api/v1/notfound/")
		// Not found errors are not retried, so we expect an error from Get
		if err == nil {
			t.Error("Expected not found error from Get")
		}
		if resp != nil {
			resp.Body.Close()
		}
	})

	t.Run("JSON Parse Error", func(t *testing.T) {
		var target map[string]interface{}
		err := client.GetJSON(server.URL+"/api/v1/invalid/", &target)
		if err == nil {
			t.Error("Expected JSON parse error")
		}
	})

	t.Run("FetchUserProfile", func(t *testing.T) {
		// For this test, we'll simulate a profile fetch using GetJSON directly
		var result InstagramResponse
		err := client.GetJSON(server.URL+"/api/v1/users/web_profile_info/", &result)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Retry Logic", func(t *testing.T) {
		// Create a request that will trigger retry
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/error/", nil)
		resp, err := client.doRequestWithRetry(req)
		if err == nil {
			t.Error("Expected error after retries")
		}
		if resp != nil {
			resp.Body.Close()
		}
	})
}