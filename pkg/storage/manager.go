package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"igscraper/pkg/instagram"
	"igscraper/pkg/logger"
	"igscraper/pkg/metadata"
)

// Manager handles file storage operations and duplicate detection
type Manager struct {
	outputDir        string
	downloadedPhotos map[string]bool
	mu               sync.RWMutex
	logger           logger.Logger
	userMetadata     *metadata.UserMetadata
}

// NewManager creates a new storage manager with default logger
func NewManager(outputDir string) (*Manager, error) {
	return NewManagerWithLogger(outputDir, logger.GetLogger())
}

// NewManagerWithLogger creates a new storage manager with a custom logger
func NewManagerWithLogger(outputDir string, log logger.Logger) (*Manager, error) {
	if log == nil {
		log = logger.GetLogger()
	}
	
	// Create output directory if it doesn't exist
	log.Info("Creating output directory")
	log.WithField("directory", outputDir).Debug("Directory path")
	
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.WithError(err).WithField("directory", outputDir).Error("Failed to create output directory")
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	
	log.WithField("directory", outputDir).Info("Output directory ready")

	manager := &Manager{
		outputDir:        outputDir,
		downloadedPhotos: make(map[string]bool),
		logger:           log,
		userMetadata:     nil, // Will be initialized when starting download
	}

	// Scan existing files for duplicate detection
	log.Info("Scanning existing files for duplicate detection")
	if err := manager.scanExistingFiles(); err != nil {
		log.WithError(err).Error("Failed to scan existing files")
		return nil, fmt.Errorf("failed to scan existing files: %w", err)
	}

	return manager, nil
}

// scanExistingFiles scans the output directory for already downloaded files
func (m *Manager) scanExistingFiles() error {
	m.logger.WithField("directory", m.outputDir).Debug("Reading directory contents")
	
	entries, err := os.ReadDir(m.outputDir)
	if err != nil {
		m.logger.WithError(err).WithField("directory", m.outputDir).Error("Failed to read directory")
		return fmt.Errorf("failed to read directory: %w", err)
	}

	fileCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jpg" {
			// Extract shortcode from filename (format: shortcode.jpg)
			shortcode := filepath.Base(entry.Name())
			shortcode = shortcode[:len(shortcode)-4] // Remove .jpg extension
			m.downloadedPhotos[shortcode] = true
			fileCount++
		}
	}
	
	m.logger.WithFields(map[string]interface{}{
		"directory": m.outputDir,
		"file_count": fileCount,
	}).Info("Completed scanning existing files")

	return nil
}

// IsDownloaded checks if a photo with the given shortcode has already been downloaded
func (m *Manager) IsDownloaded(shortcode string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check in-memory map first
	if m.downloadedPhotos[shortcode] {
		m.logger.WithField("shortcode", shortcode).Debug("Photo already downloaded (found in cache)")
		return true
	}
	
	// Double-check file existence
	filename := filepath.Join(m.outputDir, fmt.Sprintf("%s.jpg", shortcode))
	if _, err := os.Stat(filename); err == nil {
		// Update cache if file exists
		m.mu.RUnlock()
		m.mu.Lock()
		m.downloadedPhotos[shortcode] = true
		m.mu.Unlock()
		m.mu.RLock()
		m.logger.WithField("shortcode", shortcode).Debug("Photo already downloaded (found on disk)")
		return true
	}
	
	return false
}

// SavePhoto saves a photo from the given reader
func (m *Manager) SavePhoto(r io.Reader, shortcode string) error {
	filename := filepath.Join(m.outputDir, fmt.Sprintf("%s.jpg", shortcode))
	
	m.logger.WithFields(map[string]interface{}{
		"shortcode": shortcode,
		"filename": filename,
	}).Debug("Saving photo")
	
	// Create temporary file first
	tempFile := filename + ".tmp"
	out, err := os.Create(tempFile)
	if err != nil {
		m.logger.WithError(err).WithFields(map[string]interface{}{
			"shortcode": shortcode,
			"temp_file": tempFile,
		}).Error("Failed to create temporary file")
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	
	// Copy data
	_, err = io.Copy(out, r)
	closeErr := out.Close()
	
	if err != nil {
		os.Remove(tempFile) // Clean up temp file
		m.logger.WithError(err).WithFields(map[string]interface{}{
			"shortcode": shortcode,
			"temp_file": tempFile,
		}).Error("Failed to save photo data")
		return fmt.Errorf("failed to save photo data: %w", err)
	}
	
	if closeErr != nil {
		os.Remove(tempFile) // Clean up temp file
		m.logger.WithError(closeErr).WithFields(map[string]interface{}{
			"shortcode": shortcode,
			"temp_file": tempFile,
		}).Error("Failed to close file")
		return fmt.Errorf("failed to close file: %w", closeErr)
	}
	
	// Atomic rename
	if err := os.Rename(tempFile, filename); err != nil {
		os.Remove(tempFile) // Clean up temp file
		m.logger.WithError(err).WithFields(map[string]interface{}{
			"shortcode": shortcode,
			"temp_file": tempFile,
			"filename": filename,
		}).Error("Failed to rename temporary file")
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}
	
	// Update downloaded map
	m.mu.Lock()
	m.downloadedPhotos[shortcode] = true
	m.mu.Unlock()
	
	m.logger.WithFields(map[string]interface{}{
		"shortcode": shortcode,
		"filename": filename,
	}).Info("Photo saved successfully")
	
	return nil
}

// SavePhotoWithMetadata saves a photo and its metadata
func (m *Manager) SavePhotoWithMetadata(r io.Reader, shortcode string, node *instagram.Node) error {
	filename := filepath.Join(m.outputDir, fmt.Sprintf("%s.jpg", shortcode))
	
	// Create temporary file first
	tempFile := filename + ".tmp"
	out, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	
	// Copy data and get file size
	size, err := io.Copy(out, r)
	closeErr := out.Close()
	
	if err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to save photo data: %w", err)
	}
	
	if closeErr != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to close file: %w", closeErr)
	}
	
	// Atomic rename
	if err := os.Rename(tempFile, filename); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}
	
	// Add metadata to collection if node data is provided
	if node != nil && m.userMetadata != nil {
		meta := metadata.FromInstagramNode(node, size)
		m.mu.Lock()
		m.userMetadata.AddPhoto(*meta)
		m.mu.Unlock()
	}
	
	// Update downloaded map
	m.mu.Lock()
	m.downloadedPhotos[shortcode] = true
	m.mu.Unlock()
	
	return nil
}

// GetOutputDir returns the output directory path
func (m *Manager) GetOutputDir() string {
	return m.outputDir
}

// GetDownloadedCount returns the number of downloaded photos
func (m *Manager) GetDownloadedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.downloadedPhotos)
}

// InitializeUserMetadata initializes the metadata collection for a user
func (m *Manager) InitializeUserMetadata(username, userID string, totalPhotos int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.userMetadata = &metadata.UserMetadata{
		Username:         username,
		UserID:           userID,
		TotalPhotos:      totalPhotos,
		DownloadStarted:  time.Now(),
		Photos:           make([]metadata.PhotoMetadata, 0),
	}
}

// SaveUserMetadata saves all collected metadata to a single JSON file
func (m *Manager) SaveUserMetadata() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.userMetadata == nil {
		return nil // Nothing to save
	}
	
	return m.userMetadata.Save(m.outputDir)
}

// GetUserMetadata returns the collected user metadata
func (m *Manager) GetUserMetadata() *metadata.UserMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.userMetadata
}
