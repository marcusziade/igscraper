// Package scraper provides the core functionality for downloading Instagram photos.
//
// The scraper package orchestrates the entire download process, coordinating
// between the Instagram API client, storage management, and rate limiting.
//
// Architecture:
//
// The Scraper struct is the main component that:
//   - Manages HTTP requests to Instagram's API
//   - Handles pagination through user's photos
//   - Implements rate limiting to avoid being blocked
//   - Manages file storage and duplicate detection
//   - Provides progress tracking and notifications
//
// Usage:
//
//	client := &http.Client{Timeout: 30 * time.Second}
//	headers := http.Header{
//	    "User-Agent": []string{"Mozilla/5.0..."},
//	    "Cookie": []string{"sessionid=...; csrftoken=..."},
//	    // ... other required headers
//	}
//	
//	scraper, err := scraper.New(client, headers, "output_directory")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	
//	err = scraper.DownloadUserPhotos("instagram_username")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Rate Limiting:
//
// The scraper implements a token bucket rate limiter with a default of 50
// downloads per hour. When the rate limit is reached, the scraper will
// automatically pause for the required cool-down period.
//
// Storage:
//
// Downloaded photos are saved to the specified output directory with the
// filename format: {shortcode}.jpg. The scraper automatically detects and
// skips previously downloaded photos.
package scraper