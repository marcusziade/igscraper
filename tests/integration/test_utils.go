package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"igscraper/pkg/checkpoint"
	"igscraper/pkg/config"
	"igscraper/pkg/logger"
)

// TestHelper provides common test utilities
type TestHelper struct {
	t           *testing.T
	mockServer  *MockInstagramServer
	tempDir     string
	cleanupFuncs []func()
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	tempDir, err := ioutil.TempDir("", "igscraper_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	return &TestHelper{
		t:            t,
		tempDir:      tempDir,
		cleanupFuncs: []func(){},
	}
}

// SetupMockServer initializes the mock Instagram server
func (h *TestHelper) SetupMockServer() *MockInstagramServer {
	fixturesDir := filepath.Join(".", "fixtures")
	h.mockServer = NewMockInstagramServer(fixturesDir)
	h.AddCleanup(h.mockServer.Close)
	return h.mockServer
}

// GetTempDir returns the temporary directory for test files
func (h *TestHelper) GetTempDir() string {
	return h.tempDir
}

// CreateTempSubDir creates a subdirectory in the temp directory
func (h *TestHelper) CreateTempSubDir(name string) string {
	dir := filepath.Join(h.tempDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		h.t.Fatalf("Failed to create temp subdir: %v", err)
	}
	return dir
}

// AddCleanup adds a cleanup function to be called when test ends
func (h *TestHelper) AddCleanup(fn func()) {
	h.cleanupFuncs = append(h.cleanupFuncs, fn)
}

// Cleanup runs all cleanup functions
func (h *TestHelper) Cleanup() {
	for i := len(h.cleanupFuncs) - 1; i >= 0; i-- {
		h.cleanupFuncs[i]()
	}
	os.RemoveAll(h.tempDir)
}

// CreateTestLogger creates a test logger
func (h *TestHelper) CreateTestLogger() logger.Logger {
	return logger.NewTestLogger()
}

// CreateTestConfig creates a test configuration
func (h *TestHelper) CreateTestConfig() *config.Config {
	cfg := config.DefaultConfig()
	
	// Override for testing
	cfg.Output.BaseDirectory = h.CreateTempSubDir("downloads")
	cfg.Output.CreateUserFolders = true
	cfg.Output.FileNamePattern = "{shortcode}.jpg"
	
	cfg.Download.ConcurrentDownloads = 3
	cfg.Download.DownloadTimeout = 5 * time.Second
	cfg.Download.RetryAttempts = 3
	cfg.Download.SkipVideos = false
	
	cfg.RateLimit.RequestsPerMinute = 600 // Higher for testing
	cfg.RateLimit.RetryDelay = 100 * time.Millisecond
	
	cfg.Retry.Enabled = true
	cfg.Retry.MaxAttempts = 3
	cfg.Retry.BaseDelay = 100 * time.Millisecond
	cfg.Retry.MaxDelay = 2 * time.Second
	cfg.Retry.NetworkRetries = 5
	cfg.Retry.NetworkBaseDelay = 500 * time.Millisecond
	
	cfg.Instagram.UserAgent = "TestBot/1.0"
	cfg.Instagram.SessionID = "test_session_id"
	cfg.Instagram.CSRFToken = "test_csrf_token"
	
	return cfg
}

// AssertFileExists checks if a file exists
func (h *TestHelper) AssertFileExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		h.t.Errorf("Expected file to exist: %s", path)
	}
}

// AssertFileNotExists checks if a file does not exist
func (h *TestHelper) AssertFileNotExists(path string) {
	if _, err := os.Stat(path); err == nil {
		h.t.Errorf("Expected file to not exist: %s", path)
	}
}

// AssertFileContains checks if a file contains expected content
func (h *TestHelper) AssertFileContains(path string, expected string) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		h.t.Errorf("Failed to read file %s: %v", path, err)
		return
	}
	
	if string(content) != expected {
		h.t.Errorf("File content mismatch. Expected: %s, Got: %s", expected, string(content))
	}
}

// AssertDirContainsFiles checks if directory contains expected number of files
func (h *TestHelper) AssertDirContainsFiles(dir string, expectedCount int) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		h.t.Errorf("Failed to read directory %s: %v", dir, err)
		return
	}
	
	actualCount := 0
	for _, f := range files {
		if !f.IsDir() {
			actualCount++
		}
	}
	
	if actualCount != expectedCount {
		h.t.Errorf("Directory %s contains %d files, expected %d", dir, actualCount, expectedCount)
	}
}

// CreateCheckpoint creates a test checkpoint
func (h *TestHelper) CreateCheckpoint(username string, userID string, downloaded map[string]string) error {
	manager, err := checkpoint.NewManager(username)
	if err != nil {
		return err
	}
	
	cp := &checkpoint.Checkpoint{
		Username:         username,
		UserID:           userID,
		LastProcessedPage: 1,
		EndCursor:        "CURSOR_PAGE_1",
		DownloadedPhotos: downloaded,
		TotalQueued:      25,
		TotalDownloaded:  len(downloaded),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Version:          1,
	}
	
	return manager.Save(cp)
}

// LoadCheckpoint loads a checkpoint for testing
func (h *TestHelper) LoadCheckpoint(username string) (*checkpoint.Checkpoint, error) {
	manager, err := checkpoint.NewManager(username)
	if err != nil {
		return nil, err
	}
	return manager.Load()
}

// WaitForCondition waits for a condition to be true with timeout
func (h *TestHelper) WaitForCondition(condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	h.t.Errorf("Timeout waiting for condition: %s", message)
}

// GenerateTestPhotos generates a specified number of test photo nodes
func GenerateTestPhotos(count int, prefix string) []map[string]interface{} {
	photos := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		photos[i] = map[string]interface{}{
			"node": map[string]interface{}{
				"id":          fmt.Sprintf("%s%d", prefix, i),
				"shortcode":   fmt.Sprintf("SC%s%d", prefix, i),
				"display_url": fmt.Sprintf("/photos/%s_%d.jpg", prefix, i),
				"is_video":    false,
			},
		}
	}
	return photos
}

// GenerateProfileResponse generates a test profile response
func GenerateProfileResponse(userID string, photoCount int, hasNextPage bool, nextCursor string) map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"user": map[string]interface{}{
				"id": userID,
				"edge_owner_to_timeline_media": map[string]interface{}{
					"count": photoCount,
					"page_info": map[string]interface{}{
						"has_next_page": hasNextPage,
						"end_cursor":    nextCursor,
					},
					"edges": GenerateTestPhotos(12, userID), // First page with 12 items
				},
			},
		},
		"status":            "ok",
		"requires_to_login": false,
	}
}

// GenerateMediaResponse generates a test media pagination response
func GenerateMediaResponse(userID string, page int, hasNextPage bool) map[string]interface{} {
	nextCursor := ""
	if hasNextPage {
		nextCursor = fmt.Sprintf("PAGE_%d_CURSOR", page+1)
	}
	
	return map[string]interface{}{
		"data": map[string]interface{}{
			"user": map[string]interface{}{
				"edge_owner_to_timeline_media": map[string]interface{}{
					"page_info": map[string]interface{}{
						"has_next_page": hasNextPage,
						"end_cursor":    nextCursor,
					},
					"edges": GenerateTestPhotos(12, fmt.Sprintf("%s_p%d", userID, page)),
				},
			},
		},
		"status": "ok",
	}
}

// AssertNoError fails the test if err is not nil
func (h *TestHelper) AssertNoError(err error) {
	if err != nil {
		h.t.Fatalf("Unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil
func (h *TestHelper) AssertError(err error) {
	if err == nil {
		h.t.Fatal("Expected error but got nil")
	}
}

// AssertErrorContains checks if error contains expected substring
func (h *TestHelper) AssertErrorContains(err error, substr string) {
	if err == nil {
		h.t.Fatal("Expected error but got nil")
	}
	if !containsString(err.Error(), substr) {
		h.t.Errorf("Error message '%s' does not contain '%s'", err.Error(), substr)
	}
}

// AssertEqual checks if two values are equal
func (h *TestHelper) AssertEqual(expected, actual interface{}) {
	if expected != actual {
		h.t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// containsString checks if a string contains a substring
func containsString(str, substr string) bool {
	return strings.Contains(str, substr)
}