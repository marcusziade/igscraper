package scraper_test

import (
	"fmt"
	"time"

	"igscraper/pkg/config"
	"igscraper/pkg/scraper"
)

func ExampleScraper_DownloadUserPhotos() {
	// Setup configuration
	cfg := config.DefaultConfig()
	
	// Configure Instagram credentials (you need valid session cookies)
	cfg.Instagram.SessionID = "YOUR_SESSION_ID"
	cfg.Instagram.CSRFToken = "YOUR_CSRF_TOKEN"
	
	// Configure download settings
	cfg.Download.ConcurrentDownloads = 5
	cfg.Download.DownloadTimeout = 30 * time.Second
	
	// Set output directory
	cfg.Output.BaseDirectory = "./downloads"
	cfg.Output.CreateUserFolders = true
	
	// Create scraper
	s, err := scraper.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create scraper: %v\n", err)
		return
	}

	// Download photos
	if err := s.DownloadUserPhotos("example_username"); err != nil {
		fmt.Printf("Failed to download photos: %v\n", err)
		return
	}

	fmt.Println("Photos downloaded successfully!")
}