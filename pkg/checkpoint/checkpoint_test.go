package checkpoint

import (
	"os"
	"testing"
)

func TestCheckpointManager(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "checkpoint_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set environment variable to use temp directory
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Unsetenv("XDG_DATA_HOME")

	username := "testuser"

	t.Run("CreateAndLoad", func(t *testing.T) {
		mgr, err := NewManager(username)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		// Create checkpoint
		cp, err := mgr.Create(username, "12345")
		if err != nil {
			t.Fatalf("Failed to create checkpoint: %v", err)
		}

		if cp.Username != username {
			t.Errorf("Expected username %s, got %s", username, cp.Username)
		}
		if cp.UserID != "12345" {
			t.Errorf("Expected user ID 12345, got %s", cp.UserID)
		}

		// Load checkpoint
		loaded, err := mgr.Load()
		if err != nil {
			t.Fatalf("Failed to load checkpoint: %v", err)
		}
		if loaded == nil {
			t.Fatal("Expected checkpoint, got nil")
		}
		if loaded.Username != username {
			t.Errorf("Expected loaded username %s, got %s", username, loaded.Username)
		}
	})

	t.Run("UpdateProgress", func(t *testing.T) {
		mgr, err := NewManager(username)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		cp, err := mgr.Create(username, "12345")
		if err != nil {
			t.Fatalf("Failed to create checkpoint: %v", err)
		}

		// Update progress
		err = mgr.UpdateProgress(cp, "cursor123", 5)
		if err != nil {
			t.Fatalf("Failed to update progress: %v", err)
		}

		// Verify update
		loaded, err := mgr.Load()
		if err != nil {
			t.Fatalf("Failed to load checkpoint: %v", err)
		}
		if loaded.EndCursor != "cursor123" {
			t.Errorf("Expected cursor cursor123, got %s", loaded.EndCursor)
		}
		if loaded.LastProcessedPage != 5 {
			t.Errorf("Expected page 5, got %d", loaded.LastProcessedPage)
		}
	})

	t.Run("RecordDownload", func(t *testing.T) {
		mgr, err := NewManager(username)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		cp, err := mgr.Create(username, "12345")
		if err != nil {
			t.Fatalf("Failed to create checkpoint: %v", err)
		}

		// Record downloads
		err = mgr.RecordDownload(cp, "ABC123", "ABC123.jpg")
		if err != nil {
			t.Fatalf("Failed to record download: %v", err)
		}
		err = mgr.RecordDownload(cp, "DEF456", "DEF456.jpg")
		if err != nil {
			t.Fatalf("Failed to record download: %v", err)
		}

		// Verify downloads
		if !cp.IsPhotoDownloaded("ABC123") {
			t.Error("Expected ABC123 to be downloaded")
		}
		if !cp.IsPhotoDownloaded("DEF456") {
			t.Error("Expected DEF456 to be downloaded")
		}
		if cp.IsPhotoDownloaded("XYZ789") {
			t.Error("Expected XYZ789 to not be downloaded")
		}
		if cp.TotalDownloaded != 2 {
			t.Errorf("Expected 2 downloads, got %d", cp.TotalDownloaded)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		mgr, err := NewManager(username)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		_, err = mgr.Create(username, "12345")
		if err != nil {
			t.Fatalf("Failed to create checkpoint: %v", err)
		}

		// Verify exists
		if !mgr.Exists() {
			t.Error("Expected checkpoint to exist")
		}

		// Delete
		err = mgr.Delete()
		if err != nil {
			t.Fatalf("Failed to delete checkpoint: %v", err)
		}

		// Verify deleted
		if mgr.Exists() {
			t.Error("Expected checkpoint to not exist after deletion")
		}
	})

	t.Run("AtomicWrite", func(t *testing.T) {
		mgr, err := NewManager(username)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		cp, err := mgr.Create(username, "12345")
		if err != nil {
			t.Fatalf("Failed to create checkpoint: %v", err)
		}

		// Simulate multiple concurrent saves
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(n int) {
				cp.TotalQueued = n
				mgr.Save(cp)
				done <- true
			}(i)
		}

		// Wait for all saves to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify checkpoint is still valid
		loaded, err := mgr.Load()
		if err != nil {
			t.Fatalf("Failed to load checkpoint after concurrent saves: %v", err)
		}
		if loaded == nil {
			t.Fatal("Checkpoint corrupted after concurrent saves")
		}
	})

	t.Run("BackupCheckpoint", func(t *testing.T) {
		mgr, err := NewManager(username)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		cp, err := mgr.Create(username, "12345")
		if err != nil {
			t.Fatalf("Failed to create checkpoint: %v", err)
		}

		// Add some data
		cp.TotalDownloaded = 42
		mgr.Save(cp)

		// Create backup
		err = mgr.BackupCheckpoint()
		if err != nil {
			t.Fatalf("Failed to backup checkpoint: %v", err)
		}

		// Verify backup exists
		backupPath := mgr.checkpointPath + ".backup"
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Error("Backup file not created")
		}
	})
}

func TestGetDataDirectory(t *testing.T) {
	// Test actual implementation
	dir, err := getDataDirectory()
	if err != nil {
		t.Fatalf("Failed to get data directory: %v", err)
	}

	// Verify it's a valid path
	if dir == "" {
		t.Error("Data directory is empty")
	}

	// Verify it can be created
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		t.Errorf("Cannot create data directory: %v", err)
	}
}