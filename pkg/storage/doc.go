// Package storage provides file management functionality for the Instagram scraper.
//
// The storage package handles:
//   - Creating and managing output directories
//   - Saving photos with atomic write operations
//   - Detecting duplicate downloads
//   - Thread-safe file operations
//
// The Manager type is the primary interface for storage operations. It maintains
// an in-memory cache of downloaded files for fast duplicate detection and
// provides atomic file writing to prevent corruption.
//
// Features:
//   - Atomic file writes using temporary files and rename
//   - Thread-safe operations with read-write mutex
//   - Automatic scanning of existing files on initialization
//   - In-memory cache for fast duplicate detection
//
// Usage:
//
//	manager, err := storage.NewManager("output_directory")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	
//	// Check if file already exists
//	if !manager.IsDownloaded("shortcode123") {
//	    // Save new photo
//	    err = manager.SavePhoto(photoReader, "shortcode123")
//	    if err != nil {
//	        log.Printf("Failed to save photo: %v", err)
//	    }
//	}
package storage