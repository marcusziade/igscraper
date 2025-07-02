# UI Package

This package provides terminal UI components for the Instagram scraper application.

## Components

### terminal.go
Provides color functions and styled output for the terminal:
- Color functions: `Cyan()`, `Yellow()`, `Red()`, `Green()`, `Magenta()`
- Print functions: `PrintLogo()`, `PrintError()`, `PrintSuccess()`, `PrintInfo()`, `PrintWarning()`, `PrintHighlight()`
- ASCII logo constant: `ASCIILogo`

### progress.go
Manages download progress tracking:
- `StatusTracker` struct for tracking download statistics
- Progress bar rendering with customizable appearance
- Batch management for rate limiting
- Methods for tracking total downloads, current batch, and elapsed time

### notifications.go
Cross-platform desktop notification support:
- `Notifier` struct with platform-specific implementations
- Support for Linux (notify-send), macOS (osascript), and Windows (PowerShell)
- Fallback to console output on unsupported platforms
- Methods for different notification types: `SendNotification()`, `SendError()`, `SendSuccess()`

## Usage

```go
import "igscraper/pkg/ui"

// Print colored output
ui.PrintLogo()
ui.PrintInfo("Starting download...")
ui.PrintSuccess("Download completed!")
ui.PrintError("Failed: %v", err)

// Track progress
tracker := ui.NewStatusTracker()
tracker.IncrementDownloaded()
tracker.PrintProgress()

// Send notifications
notifier := ui.NewNotifier()
notifier.SendNotification("Download Complete", "All photos downloaded")
```

## Platform Support

- **Linux**: Uses `notify-send` for desktop notifications
- **macOS**: Uses `osascript` for native notifications
- **Windows**: Uses PowerShell with toast notifications
- **Other**: Falls back to console output only