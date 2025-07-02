package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"igscraper/pkg/ui/tui"
)

func main() {
	// Create a new TUI with max 5 concurrent downloads
	terminal := tui.NewTUI(5)

	// Channel to signal when demo is done
	done := make(chan bool)

	// Start the TUI in a goroutine
	go func() {
		if err := terminal.Start(); err != nil {
			log.Printf("TUI error: %v\n", err)
		}
		done <- true
	}()

	// Give TUI time to initialize
	time.Sleep(500 * time.Millisecond)

	// Log initial messages
	terminal.LogInfo("Instagram Scraper TUI Demo Starting")
	terminal.LogInfo("Fetching user profile: demo_user")

	// Simulate rate limit update
	terminal.UpdateRateLimit(0, 100, time.Now().Add(time.Hour))

	// Simulate downloading photos
	photos := []struct {
		id   string
		name string
		size int64
	}{
		{"abc123", "sunset_beach.jpg", 2 * 1024 * 1024},      // 2MB
		{"def456", "city_lights.jpg", 1.5 * 1024 * 1024},     // 1.5MB
		{"ghi789", "mountain_view.jpg", 3 * 1024 * 1024},     // 3MB
		{"jkl012", "coffee_morning.jpg", 800 * 1024},         // 800KB
		{"mno345", "street_art.jpg", 1.2 * 1024 * 1024},      // 1.2MB
		{"pqr678", "food_plate.jpg", 950 * 1024},             // 950KB
		{"stu901", "pet_portrait.jpg", 1.8 * 1024 * 1024},    // 1.8MB
		{"vwx234", "nature_trail.jpg", 2.5 * 1024 * 1024},    // 2.5MB
		{"yz567", "architecture.jpg", 1.1 * 1024 * 1024},     // 1.1MB
		{"abc890", "selfie_group.jpg", 600 * 1024},           // 600KB
	}

	// Start downloads with delays
	for i, photo := range photos {
		terminal.StartDownload(photo.id, "demo_user", photo.name, photo.size)
		terminal.LogInfo("Queued download: %s", photo.name)

		// Update rate limit periodically
		if i%3 == 0 {
			used := (i + 1) * 10
			terminal.UpdateRateLimit(used, 100, time.Now().Add(time.Hour))
		}

		// Simulate download progress in a goroutine
		go simulateDownload(terminal, photo.id, photo.size, i)

		// Stagger download starts
		time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)
	}

	// Add some warning/error messages during download
	go func() {
		time.Sleep(2 * time.Second)
		terminal.LogWarning("Rate limit approaching threshold")
		
		time.Sleep(3 * time.Second)
		terminal.LogError("Connection timeout on photo xyz123 (retrying...)")
		
		time.Sleep(2 * time.Second)
		terminal.LogSuccess("All retries completed successfully")
	}()

	// Let downloads complete
	time.Sleep(15 * time.Second)

	terminal.LogSuccess("Demo completed! Press 'q' to quit.")

	// Wait for TUI to finish
	<-done
}

func simulateDownload(terminal *tui.TUI, photoID string, totalSize int64, index int) {
	// Random download duration between 3-8 seconds
	duration := time.Duration(3+rand.Intn(5)) * time.Second
	steps := 20
	stepDuration := duration / time.Duration(steps)

	// Simulate progressive download
	for i := 1; i <= steps; i++ {
		downloaded := (totalSize * int64(i)) / int64(steps)
		
		// Calculate speed (bytes per second)
		elapsed := float64(i) * stepDuration.Seconds()
		speed := float64(downloaded) / elapsed

		terminal.UpdateDownloadProgress(photoID, downloaded, speed)
		time.Sleep(stepDuration)

		// Random pause to simulate network fluctuation
		if rand.Float32() < 0.1 {
			time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
		}
	}

	// Complete or fail based on index
	if index == 3 || index == 7 {
		// Simulate some failures
		terminal.FailDownload(photoID, fmt.Errorf("404 Not Found"))
		terminal.LogError("Failed to download photo %s", photoID)
	} else {
		terminal.CompleteDownload(photoID)
	}
}