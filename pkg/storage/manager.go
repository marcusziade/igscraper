package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Manager handles file storage operations and duplicate detection
type Manager struct {
	outputDir        string
	downloadedPhotos map[string]bool
	mu               sync.RWMutex
}

// NewManager creates a new storage manager
func NewManager(outputDir string) (*Manager, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	manager := &Manager{
		outputDir:        outputDir,
		downloadedPhotos: make(map[string]bool),
	}

	// Scan existing files for duplicate detection
	if err := manager.scanExistingFiles(); err != nil {
		return nil, fmt.Errorf("failed to scan existing files: %w", err)
	}

	return manager, nil
}

// scanExistingFiles scans the output directory for already downloaded files
func (m *Manager) scanExistingFiles() error {
	entries, err := os.ReadDir(m.outputDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jpg" {
			// Extract shortcode from filename (format: shortcode.jpg)
			shortcode := filepath.Base(entry.Name())
			shortcode = shortcode[:len(shortcode)-4] // Remove .jpg extension
			m.downloadedPhotos[shortcode] = true
		}
	}

	return nil
}

// IsDownloaded checks if a photo with the given shortcode has already been downloaded
func (m *Manager) IsDownloaded(shortcode string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check in-memory map first
	if m.downloadedPhotos[shortcode] {
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
		return true
	}
	
	return false
}

// SavePhoto saves a photo from the given reader
func (m *Manager) SavePhoto(r io.Reader, shortcode string) error {
	filename := filepath.Join(m.outputDir, fmt.Sprintf("%s.jpg", shortcode))
	
	// Create temporary file first
	tempFile := filename + ".tmp"
	out, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	
	// Copy data
	_, err = io.Copy(out, r)
	closeErr := out.Close()
	
	if err != nil {
		os.Remove(tempFile) // Clean up temp file
		return fmt.Errorf("failed to save photo data: %w", err)
	}
	
	if closeErr != nil {
		os.Remove(tempFile) // Clean up temp file
		return fmt.Errorf("failed to close file: %w", closeErr)
	}
	
	// Atomic rename
	if err := os.Rename(tempFile, filename); err != nil {
		os.Remove(tempFile) // Clean up temp file
		return fmt.Errorf("failed to rename temporary file: %w", err)
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
