package integration

import (
	"testing"
	"time"
	"net/http"
	"encoding/json"
	
	"igscraper/pkg/instagram"
	"igscraper/pkg/logger"
)

// TestMockServerFunctionality tests that the mock server works correctly
func TestMockServerFunctionality(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	mockServer := helper.SetupMockServer()
	
	// Test profile endpoint
	resp, err := http.Get(mockServer.GetURL() + "/api/v1/users/web_profile_info/?username=testuser")
	if err != nil {
		t.Fatalf("Failed to get profile: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var profile instagram.InstagramResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		t.Fatalf("Failed to decode profile response: %v", err)
	}
	
	if profile.Data.User.ID != "987654321" {
		t.Errorf("Expected user ID 987654321, got %s", profile.Data.User.ID)
	}
}

// TestRateLimitingBehavior tests the mock server's rate limiting
func TestRateLimitingBehavior(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	mockServer := helper.SetupMockServer()
	mockServer.ResetCounters()
	
	// Make 10 requests - the 10th should be rate limited
	var rateLimited bool
	for i := 1; i <= 10; i++ {
		resp, err := http.Get(mockServer.GetURL() + "/api/v1/users/web_profile_info/?username=test")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		
		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimited = true
		}
		resp.Body.Close()
	}
	
	if !rateLimited {
		t.Error("Expected at least one rate limited response")
	}
	
	if mockServer.GetRateLimitHits() == 0 {
		t.Error("Expected rate limit hits to be recorded")
	}
}

// TestErrorSimulation tests error simulation functionality
func TestErrorSimulation(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	mockServer := helper.SetupMockServer()
	
	// Test 500 error
	mockServer.SetErrorResponse("/api/v1/users/web_profile_info/erroruser", http.StatusInternalServerError)
	
	resp, err := http.Get(mockServer.GetURL() + "/api/v1/users/web_profile_info/?username=erroruser")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
	
	// Clear error and test again
	mockServer.ClearErrorResponse("/api/v1/users/web_profile_info/erroruser")
	
	resp2, err := http.Get(mockServer.GetURL() + "/api/v1/users/web_profile_info/?username=erroruser")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp2.Body.Close()
	
	if resp2.StatusCode == http.StatusInternalServerError {
		t.Error("Expected error to be cleared")
	}
}

// TestPhotoDownload tests photo download simulation
func TestPhotoDownload(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	mockServer := helper.SetupMockServer()
	
	// Test downloading a photo
	resp, err := http.Get(mockServer.GetURL() + "/photos/test_photo.jpg")
	if err != nil {
		t.Fatalf("Failed to download photo: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	if resp.Header.Get("Content-Type") != "image/jpeg" {
		t.Errorf("Expected Content-Type image/jpeg, got %s", resp.Header.Get("Content-Type"))
	}
}

// TestInstagramClientBasics tests basic Instagram client functionality
func TestInstagramClientBasics(t *testing.T) {
	log := logger.NewTestLogger()
	client := instagram.NewClient(5*time.Second, log)
	
	// Test that client is created properly
	if client == nil {
		t.Fatal("Failed to create Instagram client")
	}
	
	// Test setting headers
	client.SetHeader("Test-Header", "test-value")
	client.SetHeaders(map[string]string{
		"Another-Header": "another-value",
		"Third-Header": "third-value",
	})
}

// TestCheckpointFunctionality tests checkpoint operations
func TestCheckpointFunctionality(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()
	
	username := "testuser"
	userID := "123456"
	downloaded := map[string]string{
		"photo1": "photo1.jpg",
		"photo2": "photo2.jpg",
	}
	
	// Create checkpoint
	err := helper.CreateCheckpoint(username, userID, downloaded)
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}
	
	// Load checkpoint
	checkpoint, err := helper.LoadCheckpoint(username)
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}
	
	if checkpoint == nil {
		t.Fatal("Checkpoint should not be nil")
	}
	
	if checkpoint.Username != username {
		t.Errorf("Expected username %s, got %s", username, checkpoint.Username)
	}
	
	if checkpoint.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, checkpoint.UserID)
	}
	
	if len(checkpoint.DownloadedPhotos) != 2 {
		t.Errorf("Expected 2 downloaded photos, got %d", len(checkpoint.DownloadedPhotos))
	}
}