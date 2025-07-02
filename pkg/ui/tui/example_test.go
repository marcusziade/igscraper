package tui_test

import (
	"fmt"
	"time"

	"igscraper/pkg/ui/tui"
)

func ExampleTUI() {
	// Create a new TUI with max 5 concurrent downloads
	terminal := tui.NewTUI(5)

	// Start the TUI in a goroutine
	go func() {
		if err := terminal.Start(); err != nil {
			fmt.Printf("TUI error: %v\n", err)
		}
	}()

	// Simulate downloads
	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("photo_%d", i)
		terminal.StartDownload(id, "testuser", fmt.Sprintf("photo%d.jpg", i), 1024*1024) // 1MB
		
		// Simulate download progress
		go func(photoID string, num int) {
			for progress := 0; progress <= 100; progress += 10 {
				time.Sleep(100 * time.Millisecond)
				downloaded := int64(progress * 1024 * 10) // Convert to bytes
				speed := float64(1024 * 1024) // 1MB/s
				terminal.UpdateDownloadProgress(photoID, downloaded, speed)
			}
			
			// Complete or fail randomly
			if num%3 == 0 {
				terminal.FailDownload(photoID, fmt.Errorf("simulated error"))
			} else {
				terminal.CompleteDownload(photoID)
			}
		}(id, i)
		
		time.Sleep(200 * time.Millisecond) // Stagger starts
	}

	// Update rate limit
	terminal.UpdateRateLimit(30, 100, time.Now().Add(time.Hour))

	// Add some logs
	terminal.LogInfo("Starting download session")
	terminal.LogWarning("Rate limit approaching")
	terminal.LogError("Failed to connect to server")
	terminal.LogSuccess("Download completed successfully")

	// Keep running for demo
	time.Sleep(10 * time.Second)
	terminal.Stop()
}