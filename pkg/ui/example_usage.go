// Package ui provides terminal UI components for the Instagram scraper
// This file demonstrates example usage of the UI components
package ui

/*
Example usage of the UI components:

// Terminal colors and output
ui.PrintLogo()                                    // Print ASCII logo
ui.PrintInfo("Starting download...")             // Cyan info message
ui.PrintSuccess("Download completed!")           // Green success message
ui.PrintError("Failed to download: %v", err)    // Red error message
ui.PrintWarning("Rate limit approaching")        // Yellow warning message
ui.PrintHighlight("[PROCESSING]")                // Magenta highlight message

// Progress tracking
tracker := ui.NewStatusTracker()
tracker.IncrementDownloaded()                    // Increment counters
tracker.PrintProgress()                          // Print progress bar
tracker.PrintBatchStatus()                       // Print batch status
if tracker.IsRateLimitReached() {               // Check rate limit
    tracker.ResetBatch()                         // Reset batch counter
}

// Notifications (cross-platform)
notifier := ui.NewNotifier()
notifier.SendNotification("Download Complete", "All photos downloaded successfully")
notifier.SendError("Error", "Failed to download photo")
notifier.SendSuccess("Success", "Profile scraped successfully")

// Direct color usage
fmt.Printf("%s: %s\n", ui.Cyan("Username"), ui.Yellow("john_doe"))
fmt.Println(ui.Green("✓ Success"))
fmt.Println(ui.Red("✗ Failed"))
*/