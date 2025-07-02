package main

import (
	"fmt"
	"log"
	"time"

	"igscraper/pkg/config"
	"igscraper/pkg/scraper"
)

func main() {
	// Create configuration with concurrent downloads enabled
	cfg := config.DefaultConfig()
	
	// Set Instagram credentials (these would normally come from environment variables)
	cfg.Instagram.SessionID = "your_session_id"
	cfg.Instagram.CSRFToken = "your_csrf_token"
	
	// Configure concurrent downloads
	cfg.Download.ConcurrentDownloads = 5 // Use 5 workers
	cfg.Download.DownloadTimeout = 30 * time.Second
	
	// Configure rate limiting
	cfg.RateLimit.RequestsPerMinute = 60
	
	// Set output directory
	cfg.Output.BaseDirectory = "./downloads"
	cfg.Output.CreateUserFolders = true
	
	// Create scraper
	s, err := scraper.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create scraper: %v", err)
	}
	
	// Start downloading photos
	username := "example_user"
	fmt.Printf("Starting concurrent download for user: %s\n", username)
	fmt.Printf("Using %d concurrent workers\n", cfg.Download.ConcurrentDownloads)
	
	startTime := time.Now()
	
	err = s.DownloadUserPhotos(username)
	if err != nil {
		log.Fatalf("Failed to download photos: %v", err)
	}
	
	elapsed := time.Since(startTime)
	fmt.Printf("\nDownload completed in %v\n", elapsed)
	fmt.Println("Concurrent downloads significantly improve performance!")
}