package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create manager
	manager, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test initial state
	if manager.GetDownloadedCount() != 0 {
		t.Error("Expected initial download count to be 0")
	}

	// Test IsDownloaded for non-existent file
	if manager.IsDownloaded("test123") {
		t.Error("Expected IsDownloaded to return false for non-existent file")
	}

	// Test SavePhoto
	testData := []byte("test photo data")
	reader := bytes.NewReader(testData)
	
	err = manager.SavePhoto(reader, "test123")
	if err != nil {
		t.Fatalf("Failed to save photo: %v", err)
	}

	// Verify file was created
	expectedPath := filepath.Join(tempDir, "test123.jpg")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Expected file to be created")
	}

	// Verify file content
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}
	if !bytes.Equal(content, testData) {
		t.Error("File content does not match expected data")
	}

	// Test IsDownloaded for existing file
	if !manager.IsDownloaded("test123") {
		t.Error("Expected IsDownloaded to return true for existing file")
	}

	// Test download count
	if manager.GetDownloadedCount() != 1 {
		t.Errorf("Expected download count to be 1, got %d", manager.GetDownloadedCount())
	}

	// Test scanning existing files
	// Create another file manually
	manualFile := filepath.Join(tempDir, "manual456.jpg")
	if err := os.WriteFile(manualFile, []byte("manual"), 0644); err != nil {
		t.Fatalf("Failed to create manual file: %v", err)
	}

	// Create new manager to test scanning
	manager2, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create second manager: %v", err)
	}

	// Should detect both files
	if manager2.GetDownloadedCount() != 2 {
		t.Errorf("Expected download count to be 2 after scanning, got %d", manager2.GetDownloadedCount())
	}

	if !manager2.IsDownloaded("manual456") {
		t.Error("Expected manually created file to be detected")
	}
}