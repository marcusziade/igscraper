# TUI Package - Terminal User Interface

The TUI package provides a beautiful, cyberpunk-themed terminal user interface for Instagram Scraper using Bubble Tea and Lipgloss.

## Features

- **Real-time Progress Tracking**: Live updates for all active downloads with progress bars
- **Download Queue Visualization**: See pending, active, and completed downloads
- **System Statistics**: Session time, total downloads, speeds, and ETA
- **Rate Limit Monitoring**: Visual indicator of API rate limit usage
- **System Logs**: Color-coded log messages with timestamps
- **Cyberpunk Theme**: Neon colors and styled borders matching the application theme
- **Interactive Controls**: Pause/resume downloads and view help

## Usage

To enable the TUI when running igscraper:

```bash
igscraper scrape username --tui
```

## Keyboard Shortcuts

- `q` or `Q` - Quit the application
- `p` or `P` - Pause/Resume downloads
- `?` - Toggle help display
- `Ctrl+L` - Clear logs

## Architecture

The TUI consists of several components:

### Model (`model.go`)
- Maintains the application state
- Thread-safe operations for concurrent updates
- Tracks downloads, statistics, and logs

### View (`view.go`)
- Renders the terminal interface
- Responsive layout with two columns
- Multiple panels for different information

### Update (`update.go`)
- Handles events and user input
- Processes messages from the scraper
- Updates the model based on events

### Styles (`styles.go`)
- Cyberpunk color palette
- Consistent styling across components
- Dynamic styles based on state (e.g., rate limit warnings)

### TUI Manager (`tui.go`)
- High-level interface for the scraper
- Simplified API for sending updates
- Manages the Bubble Tea program lifecycle

## Panels

### Left Column
1. **System Stats**: Overall download statistics and performance metrics
2. **Active Downloads**: Currently downloading files with progress bars
3. **Download Queue**: Pending and completed downloads

### Right Column
1. **Rate Limit Status**: Visual indicator of API usage
2. **System Logs**: Recent log messages with color coding

## Integration

The TUI integrates seamlessly with the scraper through the `ui.TUI` interface:

```go
type TUI interface {
    StartDownload(id, username, filename string, size int64)
    UpdateDownloadProgress(id string, downloaded int64, speed float64)
    CompleteDownload(id string)
    FailDownload(id string, err error)
    UpdateRateLimit(used, max int, resetAt time.Time)
    LogInfo(format string, args ...interface{})
    LogSuccess(format string, args ...interface{})
    LogWarning(format string, args ...interface{})
    LogError(format string, args ...interface{})
    IsPaused() bool
}
```

## Color Scheme

The cyberpunk theme uses the following colors:
- **Neon Cyan**: `#00FFFF` - Primary accent
- **Neon Magenta**: `#FF00FF` - Borders and titles
- **Neon Green**: `#39FF14` - Success states
- **Neon Yellow**: `#FFFF00` - Values and warnings
- **Neon Orange**: `#FF6700` - Warnings
- **Dark Background**: `#0A0E27` - Main background
- **Dark Background 2**: `#1A1E37` - Panel backgrounds

## Performance

The TUI is designed to handle high-frequency updates efficiently:
- Batched rendering at 100ms intervals
- Thread-safe state management
- Minimal allocations in hot paths
- Efficient string building for rendering