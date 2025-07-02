package checkpoint

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"igscraper/pkg/logger"
)

// Checkpoint represents the state of a download session
type Checkpoint struct {
	Username         string            `json:"username"`
	UserID           string            `json:"user_id"`
	LastProcessedPage int              `json:"last_processed_page"`
	EndCursor        string            `json:"end_cursor"`
	DownloadedPhotos map[string]string `json:"downloaded_photos"` // shortcode -> filename
	TotalQueued      int               `json:"total_queued"`
	TotalDownloaded  int               `json:"total_downloaded"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	Version          int               `json:"version"`
}

// Manager handles checkpoint operations
type Manager struct {
	checkpointPath string
	logger         logger.Logger
}

// NewManager creates a new checkpoint manager
func NewManager(username string) (*Manager, error) {
	dataDir, err := getDataDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	// Create checkpoints directory if it doesn't exist
	checkpointsDir := filepath.Join(dataDir, "checkpoints")
	if err := os.MkdirAll(checkpointsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoints directory: %w", err)
	}

	// Create checkpoint file path
	checkpointPath := filepath.Join(checkpointsDir, fmt.Sprintf("%s.checkpoint.json", username))

	return &Manager{
		checkpointPath: checkpointPath,
		logger:         logger.GetLogger(),
	}, nil
}

// Create creates a new checkpoint
func (m *Manager) Create(username, userID string) (*Checkpoint, error) {
	checkpoint := &Checkpoint{
		Username:         username,
		UserID:           userID,
		LastProcessedPage: 0,
		EndCursor:        "",
		DownloadedPhotos: make(map[string]string),
		TotalQueued:      0,
		TotalDownloaded:  0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Version:          1,
	}

	if err := m.Save(checkpoint); err != nil {
		return nil, fmt.Errorf("failed to save initial checkpoint: %w", err)
	}

	m.logger.InfoWithFields("Checkpoint created", map[string]interface{}{
		"username": username,
		"path":     m.checkpointPath,
	})

	return checkpoint, nil
}

// Load loads an existing checkpoint
func (m *Manager) Load() (*Checkpoint, error) {
	file, err := os.Open(m.checkpointPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No checkpoint exists
		}
		return nil, fmt.Errorf("failed to open checkpoint file: %w", err)
	}
	defer file.Close()

	var checkpoint Checkpoint
	if err := json.NewDecoder(file).Decode(&checkpoint); err != nil {
		return nil, fmt.Errorf("failed to decode checkpoint: %w", err)
	}

	m.logger.InfoWithFields("Checkpoint loaded", map[string]interface{}{
		"username":         checkpoint.Username,
		"total_downloaded": checkpoint.TotalDownloaded,
		"last_cursor":      checkpoint.EndCursor,
		"updated_at":       checkpoint.UpdatedAt,
	})

	return &checkpoint, nil
}

// Save saves the checkpoint to disk atomically
func (m *Manager) Save(checkpoint *Checkpoint) error {
	checkpoint.UpdatedAt = time.Now()

	// Create temporary file
	tempPath := m.checkpointPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary checkpoint file: %w", err)
	}

	// Write checkpoint data
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(checkpoint); err != nil {
		file.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to encode checkpoint: %w", err)
	}

	// Ensure data is written to disk
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to sync checkpoint file: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close checkpoint file: %w", err)
	}

	// Atomically replace the old checkpoint file
	if err := os.Rename(tempPath, m.checkpointPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to replace checkpoint file: %w", err)
	}

	m.logger.DebugWithFields("Checkpoint saved", map[string]interface{}{
		"username":         checkpoint.Username,
		"total_downloaded": checkpoint.TotalDownloaded,
		"last_cursor":      checkpoint.EndCursor,
	})

	return nil
}

// Delete removes the checkpoint file
func (m *Manager) Delete() error {
	if err := os.Remove(m.checkpointPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	m.logger.Info("Checkpoint deleted")
	return nil
}

// Exists checks if a checkpoint file exists
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.checkpointPath)
	return err == nil
}

// UpdateProgress updates the checkpoint with current progress
func (m *Manager) UpdateProgress(checkpoint *Checkpoint, endCursor string, pageNum int) error {
	checkpoint.EndCursor = endCursor
	checkpoint.LastProcessedPage = pageNum
	return m.Save(checkpoint)
}

// RecordDownload records a successfully downloaded photo
func (m *Manager) RecordDownload(checkpoint *Checkpoint, shortcode, filename string) error {
	checkpoint.DownloadedPhotos[shortcode] = filename
	checkpoint.TotalDownloaded++
	return m.Save(checkpoint)
}

// IsPhotoDownloaded checks if a photo has already been downloaded
func (checkpoint *Checkpoint) IsPhotoDownloaded(shortcode string) bool {
	_, exists := checkpoint.DownloadedPhotos[shortcode]
	return exists
}

// GetCheckpointInfo returns a summary of the checkpoint
func (m *Manager) GetCheckpointInfo() (map[string]interface{}, error) {
	checkpoint, err := m.Load()
	if err != nil {
		return nil, err
	}
	if checkpoint == nil {
		return nil, nil
	}

	return map[string]interface{}{
		"username":          checkpoint.Username,
		"total_downloaded":  checkpoint.TotalDownloaded,
		"last_cursor":       checkpoint.EndCursor,
		"created_at":        checkpoint.CreatedAt,
		"updated_at":        checkpoint.UpdatedAt,
		"age":               time.Since(checkpoint.UpdatedAt),
	}, nil
}

// BackupCheckpoint creates a backup of the current checkpoint
func (m *Manager) BackupCheckpoint() error {
	if !m.Exists() {
		return nil // Nothing to backup
	}

	backupPath := m.checkpointPath + ".backup"
	
	// Copy checkpoint file to backup
	src, err := os.Open(m.checkpointPath)
	if err != nil {
		return fmt.Errorf("failed to open checkpoint for backup: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy checkpoint to backup: %w", err)
	}

	m.logger.Debug("Checkpoint backed up")
	return nil
}

// getDataDirectory returns the appropriate data directory for the current OS
func getDataDirectory() (string, error) {
	var dataDir string

	switch runtime.GOOS {
	case "linux":
		// Use XDG_DATA_HOME if set, otherwise ~/.local/share
		if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
			dataDir = filepath.Join(xdgDataHome, "igscraper")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			dataDir = filepath.Join(home, ".local", "share", "igscraper")
		}
	case "darwin":
		// macOS: ~/Library/Application Support
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataDir = filepath.Join(home, "Library", "Application Support", "igscraper")
	case "windows":
		// Windows: %APPDATA%
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		dataDir = filepath.Join(appData, "igscraper")
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// Create the data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return dataDir, nil
}