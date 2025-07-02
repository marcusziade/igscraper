package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MockInstagramServer simulates Instagram API endpoints with realistic behavior
type MockInstagramServer struct {
	server         *httptest.Server
	fixturesDir    string
	rateLimitHits  int32
	requestCount   int32
	errorResponses map[string]int // Map of endpoint patterns to error codes
	mu             sync.RWMutex
	delays         map[string]time.Duration // Simulated response delays
	checkpoints    map[string]string        // Track pagination state
}

// NewMockInstagramServer creates a new mock Instagram API server
func NewMockInstagramServer(fixturesDir string) *MockInstagramServer {
	m := &MockInstagramServer{
		fixturesDir:    fixturesDir,
		errorResponses: make(map[string]int),
		delays:         make(map[string]time.Duration),
		checkpoints:    make(map[string]string),
	}

	mux := http.NewServeMux()
	
	// Profile endpoint
	mux.HandleFunc("/api/v1/users/web_profile_info/", m.handleProfile)
	
	// Media endpoint
	mux.HandleFunc("/graphql/query/", m.handleMedia)
	
	// Photo download endpoint (simulated CDN)
	mux.HandleFunc("/photos/", m.handlePhotoDownload)
	
	// Authentication endpoint
	mux.HandleFunc("/accounts/login/", m.handleLogin)
	
	m.server = httptest.NewServer(mux)
	return m
}

// handleProfile handles user profile requests
func (m *MockInstagramServer) handleProfile(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt32(&m.requestCount, 1)
	
	// Simulate delay if configured
	username := r.URL.Query().Get("username")
	if delay := m.getDelay("/api/v1/users/web_profile_info/" + username); delay > 0 {
		time.Sleep(delay)
	}
	
	// Check for configured errors
	if errorCode := m.getErrorResponse("/api/v1/users/web_profile_info/" + username); errorCode > 0 {
		m.sendError(w, errorCode, username)
		return
	}
	
	// Simulate rate limiting
	if m.shouldRateLimit() {
		atomic.AddInt32(&m.rateLimitHits, 1)
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Please wait a few minutes before you try again.",
			"status":  "fail",
		})
		return
	}
	
	// Load fixture based on username
	fixturePath := filepath.Join(m.fixturesDir, fmt.Sprintf("profile_%s.json", username))
	data, err := ioutil.ReadFile(fixturePath)
	if err != nil {
		// Default to a generic profile if specific fixture not found
		fixturePath = filepath.Join(m.fixturesDir, "profile_default.json")
		data, err = ioutil.ReadFile(fixturePath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "User not found",
				"status":  "fail",
			})
			return
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// handleMedia handles media pagination requests
func (m *MockInstagramServer) handleMedia(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt32(&m.requestCount, 1)
	
	// Parse query parameters
	queryHash := r.URL.Query().Get("query_hash")
	variablesStr := r.URL.Query().Get("variables")
	
	if queryHash != "e769aa130647d2354c40ea6a439bfc08" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	
	var variables map[string]interface{}
	if err := json.Unmarshal([]byte(variablesStr), &variables); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	
	userID := variables["id"].(string)
	after := ""
	if val, ok := variables["after"]; ok && val != nil {
		after = val.(string)
	}
	
	// Simulate delay
	if delay := m.getDelay("/graphql/query/" + userID); delay > 0 {
		time.Sleep(delay)
	}
	
	// Check for configured errors
	if errorCode := m.getErrorResponse("/graphql/query/" + userID); errorCode > 0 {
		m.sendError(w, errorCode, userID)
		return
	}
	
	// Simulate rate limiting
	if m.shouldRateLimit() {
		atomic.AddInt32(&m.rateLimitHits, 1)
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	
	// Load appropriate fixture based on pagination
	fixtureName := fmt.Sprintf("media_%s", userID)
	if after != "" {
		fixtureName = fmt.Sprintf("%s_after_%s", fixtureName, after)
	}
	fixturePath := filepath.Join(m.fixturesDir, fixtureName+".json")
	
	data, err := ioutil.ReadFile(fixturePath)
	if err != nil {
		// Use default media fixture
		fixturePath = filepath.Join(m.fixturesDir, "media_default.json")
		data, err = ioutil.ReadFile(fixturePath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	
	// Track checkpoint for testing resume functionality
	m.mu.Lock()
	m.checkpoints[userID] = after
	m.mu.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// handlePhotoDownload simulates photo CDN downloads
func (m *MockInstagramServer) handlePhotoDownload(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt32(&m.requestCount, 1)
	
	photoID := strings.TrimPrefix(r.URL.Path, "/photos/")
	photoID = strings.TrimSuffix(photoID, ".jpg")
	
	// Simulate delay
	if delay := m.getDelay("/photos/" + photoID); delay > 0 {
		time.Sleep(delay)
	}
	
	// Check for configured errors
	if errorCode := m.getErrorResponse("/photos/" + photoID); errorCode > 0 {
		w.WriteHeader(errorCode)
		return
	}
	
	// Simulate rate limiting for downloads
	if m.shouldRateLimit() {
		atomic.AddInt32(&m.rateLimitHits, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	
	// Return a small test image
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", "1024")
	
	// Create a simple 1KB test image
	testImage := make([]byte, 1024)
	for i := range testImage {
		testImage[i] = byte(i % 256)
	}
	w.Write(testImage)
}

// handleLogin simulates authentication endpoint
func (m *MockInstagramServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt32(&m.requestCount, 1)
	
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	// Simulate successful login
	w.Header().Set("Set-Cookie", "sessionid=test_session_id; Path=/; HttpOnly")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"user": map[string]interface{}{
			"username": "testuser",
			"pk":       "123456789",
		},
		"status": "ok",
	})
}

// sendError sends an error response
func (m *MockInstagramServer) sendError(w http.ResponseWriter, code int, context string) {
	w.WriteHeader(code)
	
	var message string
	switch code {
	case http.StatusUnauthorized:
		message = "Login required"
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":           message,
			"status":            "fail",
			"requires_to_login": true,
		})
	case http.StatusNotFound:
		message = fmt.Sprintf("Resource not found: %s", context)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": message,
			"status":  "fail",
		})
	case http.StatusInternalServerError:
		message = "Internal server error"
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": message,
			"status":  "fail",
		})
	default:
		w.Write([]byte(fmt.Sprintf("Error %d", code)))
	}
}

// SetErrorResponse configures an endpoint to return a specific error code
func (m *MockInstagramServer) SetErrorResponse(endpoint string, code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorResponses[endpoint] = code
}

// ClearErrorResponse removes error configuration for an endpoint
func (m *MockInstagramServer) ClearErrorResponse(endpoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.errorResponses, endpoint)
}

// SetDelay configures response delay for an endpoint
func (m *MockInstagramServer) SetDelay(endpoint string, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delays[endpoint] = delay
}

// getErrorResponse checks if an error is configured for the endpoint
func (m *MockInstagramServer) getErrorResponse(endpoint string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorResponses[endpoint]
}

// getDelay gets configured delay for an endpoint
func (m *MockInstagramServer) getDelay(endpoint string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.delays[endpoint]
}

// shouldRateLimit determines if a request should be rate limited
func (m *MockInstagramServer) shouldRateLimit() bool {
	// Rate limit every 10th request for testing
	return atomic.LoadInt32(&m.requestCount)%10 == 0
}

// GetURL returns the base URL of the mock server
func (m *MockInstagramServer) GetURL() string {
	return m.server.URL
}

// ReplaceBaseURL modifies Instagram API URLs to use the mock server
func (m *MockInstagramServer) ReplaceBaseURL(originalURL string) string {
	return strings.Replace(originalURL, "https://www.instagram.com", m.server.URL, 1)
}

// GetRequestCount returns the total number of requests
func (m *MockInstagramServer) GetRequestCount() int {
	return int(atomic.LoadInt32(&m.requestCount))
}

// GetRateLimitHits returns the number of rate limit responses
func (m *MockInstagramServer) GetRateLimitHits() int {
	return int(atomic.LoadInt32(&m.rateLimitHits))
}

// GetCheckpoint returns the last pagination state for a user
func (m *MockInstagramServer) GetCheckpoint(userID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.checkpoints[userID]
}

// ResetCounters resets all request counters
func (m *MockInstagramServer) ResetCounters() {
	atomic.StoreInt32(&m.requestCount, 0)
	atomic.StoreInt32(&m.rateLimitHits, 0)
	m.mu.Lock()
	m.checkpoints = make(map[string]string)
	m.mu.Unlock()
}

// Close shuts down the mock server
func (m *MockInstagramServer) Close() {
	m.server.Close()
}

// SimulateNetworkError temporarily makes the server unresponsive
func (m *MockInstagramServer) SimulateNetworkError(duration time.Duration) {
	// Close and restart the server after duration
	m.server.Close()
	time.Sleep(duration)
	
	// Recreate server with same handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users/web_profile_info/", m.handleProfile)
	mux.HandleFunc("/graphql/query/", m.handleMedia)
	mux.HandleFunc("/photos/", m.handlePhotoDownload)
	mux.HandleFunc("/accounts/login/", m.handleLogin)
	
	m.server = httptest.NewServer(mux)
}

// EnableRateLimitForRequests enables rate limiting for a specific number of requests
func (m *MockInstagramServer) EnableRateLimitForRequests(count int) {
	// This is a simplified implementation - in real tests you might want more control
	// The current implementation rate limits every 10th request
}