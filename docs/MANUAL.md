# IGScraper CLI Manual

A comprehensive guide to using IGScraper - the powerful Instagram photo downloader.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Authentication](#authentication)
- [Commands](#commands)
- [Configuration](#configuration)
- [Advanced Usage](#advanced-usage)
- [Troubleshooting](#troubleshooting)
- [Legal & Disclaimer](#legal--disclaimer)

## Overview

IGScraper is a command-line tool for downloading photos from Instagram profiles. It features concurrent downloads, smart rate limiting, checkpoint/resume functionality, and multiple output modes.

### Key Features

- 🚀 **Concurrent Downloads** - Download multiple photos simultaneously
- 🛡️ **Smart Rate Limiting** - Avoid Instagram API restrictions
- 📁 **Duplicate Detection** - Skip already downloaded photos
- 🔄 **Resume Support** - Continue interrupted downloads
- 🎨 **Multiple UI Modes** - TUI, progress bar, or quiet mode
- 🔐 **Secure Authentication** - Multiple credential storage options
- 📊 **Metadata Extraction** - Save photo captions and details

## Installation

### Homebrew (Recommended)
```bash
brew tap marcusziade/tap
brew install igscraper
```

### Direct Download
Download the latest binary from [releases](https://github.com/marcusziade/igscraper/releases).

### Build from Source
```bash
git clone https://github.com/marcusziade/igscraper
cd igscraper
go build -o igscraper ./cmd/igscraper
```

## Authentication

IGScraper requires Instagram session credentials to access profiles.

### Quick Start

1. **Get your Instagram credentials:**
   - Open Instagram in your browser
   - Open Developer Tools (F12)
   - Go to Application/Storage → Cookies
   - Find `sessionid` and `csrftoken` values

2. **Configure authentication:**
   ```bash
   # Interactive login
   igscraper auth login
   
   # Or use environment variables
   export IGSCRAPER_SESSION_ID="your_session_id"
   export IGSCRAPER_CSRF_TOKEN="your_csrf_token"
   ```

### Authentication Commands

```bash
# Add new credentials
igscraper auth login

# List saved accounts
igscraper auth list

# Switch between accounts
igscraper auth switch

# Remove credentials
igscraper auth logout
```

### Storage Options

1. **System Keychain** (Default)
   - Secure storage using OS keychain
   - macOS: Keychain Access
   - Linux: Secret Service
   - Windows: Credential Manager

2. **Encrypted File**
   - AES-256 encryption
   - Passphrase protected

3. **Environment Variables**
   - `IGSCRAPER_SESSION_ID`
   - `IGSCRAPER_CSRF_TOKEN`

## Commands

### Basic Usage

```bash
# Download all photos from a profile
igscraper username

# Download with beautiful TUI
igscraper --tui username

# Quiet mode (only errors)
igscraper -q username

# Verbose logging
igscraper -v username
```

### Global Flags

```
-c, --config string         Config file (default: $HOME/.igscraper.yaml)
    --log-level string      Log level (debug, info, warn, error) (default: info)
    --no-color             Disable colored output
    --notifications        Enable desktop notifications (default: true)
-p, --progress             Show progress bar (default mode)
-q, --quiet                Suppress all output except errors
    --tui                  Use beautiful terminal UI
-v, --verbose              Show detailed output
-h, --help                 Show help
    --version              Show version information
```

### Scrape Command Options

```bash
igscraper scrape [flags] username
```

**Flags:**
```
-o, --output string         Output directory (default: "./username_photos")
-l, --limit int            Maximum photos to download (0 = all)
-w, --workers int          Concurrent download workers (default: 3)
    --skip-videos          Skip video downloads
    --high-quality         Download highest quality available
    --metadata             Save metadata for each photo
    --resume               Resume from last checkpoint
    --force                Skip duplicate checking
    --dry-run              Preview what would be downloaded
```

**Examples:**
```bash
# Download to specific directory
igscraper -o ./downloads username

# Limit downloads and use more workers
igscraper -l 100 -w 5 username

# Download with metadata
igscraper --metadata username

# Resume interrupted download
igscraper --resume username

# Preview without downloading
igscraper --dry-run username
```

## Configuration

IGScraper uses a cascading configuration system:
1. Command-line flags (highest priority)
2. Environment variables
3. Configuration file
4. Default values

### Configuration File

Create `~/.igscraper.yaml`:

```yaml
# Authentication
auth:
  storage_type: keyring  # keyring, encrypted, or environment
  
# Download settings
download:
  output_dir: "./downloads"
  concurrent_downloads: 5
  timeout: 30s
  retry_attempts: 3
  retry_delay: 5s
  high_quality: true
  skip_videos: false
  save_metadata: true
  
# Rate limiting
rate_limit:
  requests_per_minute: 60
  download_delay: 1s
  
# UI settings
ui:
  show_notifications: true
  progress_bar_style: "gradient"  # simple, gradient, or blocks
  
# Logging
log:
  level: info
  file: ""  # Empty for stdout
  format: text  # text or json
```

### Environment Variables

All configuration options can be set via environment:

```bash
# Authentication
export IGSCRAPER_SESSION_ID="your_session"
export IGSCRAPER_CSRF_TOKEN="your_token"

# Download settings
export IGSCRAPER_OUTPUT_DIR="./downloads"
export IGSCRAPER_CONCURRENT_DOWNLOADS=5
export IGSCRAPER_HIGH_QUALITY=true

# Rate limiting
export IGSCRAPER_REQUESTS_PER_MINUTE=60
```

## Advanced Usage

### Checkpoint System

IGScraper automatically saves progress for resumable downloads:

```bash
# Resume from checkpoint
igscraper --resume username

# Checkpoint files are stored in:
# ~/.config/igscraper/checkpoints/username.checkpoint.json
```

### Batch Downloads

Download multiple profiles:

```bash
# Using a file
cat profiles.txt | xargs -I {} igscraper {}

# Using a loop
for user in user1 user2 user3; do
  igscraper $user
done
```

### Filtering Downloads

```bash
# Download only recent photos (with jq)
igscraper --metadata --dry-run username | \
  jq '.photos[] | select(.timestamp > "2024-01-01")'
```

### Integration Examples

**Notification on completion:**
```bash
igscraper username && notify-send "Download Complete"
```

**Archive downloads:**
```bash
igscraper -o temp_photos username && \
  tar -czf username_$(date +%Y%m%d).tar.gz temp_photos/
```

## Troubleshooting

### Common Issues

**Authentication Failed**
- Ensure credentials are correct and not expired
- Instagram may require re-authentication periodically
- Try logging in via browser and getting fresh tokens

**Rate Limit Errors**
- Reduce concurrent workers: `--workers 1`
- Increase delays in configuration
- Wait before retrying

**Connection Timeouts**
- Check internet connectivity
- Increase timeout in configuration
- Use fewer concurrent workers

**Missing Photos**
- Private accounts require following
- Some photos may be restricted by region
- Check `--high-quality` flag for different versions

### Debug Mode

Enable detailed logging:

```bash
# Debug output
igscraper --log-level debug username

# Save logs to file
igscraper --log-level debug username 2> debug.log
```

### Getting Help

```bash
# General help
igscraper --help

# Command-specific help
igscraper auth --help
igscraper scrape --help

# Version information
igscraper --version
```

## Legal & Disclaimer

**IMPORTANT**: This tool is for educational purposes only.

- Respect Instagram's Terms of Service
- Only download content you have permission to access
- Be mindful of copyright and intellectual property rights
- The developers are not responsible for misuse

### License

IGScraper is released under the MIT License. See [LICENSE](https://github.com/marcusziade/igscraper/blob/master/LICENSE) for details.

### Disclaimer

By using this tool, you acknowledge that:
- You are solely responsible for your use of the tool
- You will comply with all applicable laws and regulations
- You will respect the rights of content creators
- The tool is provided "as is" without warranties

For more information, see our [full disclaimer](https://github.com/marcusziade/igscraper?tab=readme-ov-file#license).

---

For issues, feature requests, or contributions, visit [GitHub](https://github.com/marcusziade/igscraper).