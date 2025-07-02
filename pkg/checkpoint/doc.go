// Package checkpoint provides functionality for saving and resuming download progress.
//
// The checkpoint system allows the scraper to resume downloads after interruptions
// such as network failures, rate limits, or manual stops. It tracks:
//   - Last processed page/cursor position
//   - Downloaded photos (to avoid duplicates)
//   - Overall progress statistics
//
// Checkpoints are stored in platform-specific data directories:
//   - Linux: ~/.local/share/igscraper/checkpoints/
//   - macOS: ~/Library/Application Support/igscraper/checkpoints/
//   - Windows: %APPDATA%/igscraper/checkpoints/
//
// The checkpoint files are saved atomically to prevent corruption and include
// versioning for future compatibility.
package checkpoint