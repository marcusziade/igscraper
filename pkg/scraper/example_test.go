package scraper_test

import (
	"fmt"
	"net/http"
	"time"

	"igscraper/pkg/scraper"
)

func ExampleScraper_DownloadUserPhotos() {
	// Setup HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Setup headers (you need valid session cookies)
	headers := http.Header{
		"User-Agent":       []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
		"Accept":           []string{"*/*"},
		"Accept-Language":  []string{"en-US,en;q=0.5"},
		"X-IG-App-ID":      []string{"936619743392459"},
		"X-Requested-With": []string{"XMLHttpRequest"},
		"Connection":       []string{"keep-alive"},
		"Referer":          []string{"https://www.instagram.com/"},
		"Cookie": []string{
			"sessionid=YOUR_SESSION_ID;",
			"csrftoken=YOUR_CSRF_TOKEN;",
		},
	}

	// Create scraper
	s, err := scraper.New(client, headers, "username_photos")
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