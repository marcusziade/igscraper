# Instagram Photo Scraper - Netrunner Edition

A powerful Instagram photo downloader with cyberpunk aesthetics, built in Go.

## Features

- 🖼️ Downloads all photos from any public Instagram profile
- 🚀 Concurrent downloads with configurable worker pool
- 🛡️ Smart rate limiting to avoid Instagram's API restrictions
- 📁 Automatic duplicate detection and skipping
- 🔔 Cross-platform desktop notifications
- 🎨 Cyberpunk-themed terminal UI with progress tracking
- 💻 Beautiful interactive TUI mode with real-time statistics
- ⚙️ Flexible configuration via files, environment variables, or CLI flags
- 🔄 Resume capability for interrupted downloads

## Installation

### Homebrew (macOS and Linux)

```bash
brew tap marcusziade/tap
brew install igscraper
```

### Download Binary

Download the latest release for your platform from the [releases page](https://github.com/marcusziade/igscraper/releases).

#### macOS
```bash
# Intel Mac
curl -L https://github.com/marcusziade/igscraper/releases/latest/download/igscraper_Darwin_x86_64.tar.gz | tar xz
sudo mv igscraper /usr/local/bin/

# Apple Silicon Mac
curl -L https://github.com/marcusziade/igscraper/releases/latest/download/igscraper_Darwin_arm64.tar.gz | tar xz
sudo mv igscraper /usr/local/bin/
```

#### Linux
```bash
# AMD64
curl -L https://github.com/marcusziade/igscraper/releases/latest/download/igscraper_Linux_x86_64.tar.gz | tar xz
sudo mv igscraper /usr/local/bin/

# ARM64
curl -L https://github.com/marcusziade/igscraper/releases/latest/download/igscraper_Linux_arm64.tar.gz | tar xz
sudo mv igscraper /usr/local/bin/
```

#### Windows

Download the appropriate `.zip` file from the [releases page](https://github.com/marcusziade/igscraper/releases) and extract `igscraper.exe` to a directory in your PATH.

### Build from Source

```bash
git clone https://github.com/marcusziade/igscraper.git
cd igscraper
go build -o igscraper ./cmd/igscraper
sudo mv igscraper /usr/local/bin/  # Optional: install system-wide
```

## Configuration

The scraper supports multiple configuration methods (in order of precedence):

1. **Command Line Flags** (highest priority)
2. **Environment Variables** (including .env files)
3. **Configuration File** (.igscraper.yaml)
4. **Default values** (lowest priority)

### Quick Start with .env File

1. Copy the example .env file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` and add your Instagram credentials:
   ```env
   IGSCRAPER_SESSION_ID=your_session_id_here
   IGSCRAPER_CSRF_TOKEN=your_csrf_token_here
   ```

3. Run the scraper:
   ```bash
   ./igscraper username
   ```

### Command Line Usage

```bash
./igscraper [flags] <instagram_username>
```

Available flags:
- `--session-id`: Instagram session ID
- `--csrf-token`: Instagram CSRF token
- `--output`: Output directory for downloads (default: ./downloads)
- `--concurrent`: Number of concurrent downloads (default: 3)
- `--rate-limit`: Requests per minute (default: 60)
- `--notifications`: Enable desktop notifications (default: true)
- `--tui`: Enable beautiful terminal UI with real-time progress (default: false)
- `--config`: Path to configuration file

Example:
```bash
./igscraper --session-id "your_session" --csrf-token "your_token" --output "./photos" zuck

# With interactive TUI
./igscraper --tui username
```

### Configuration File

Create `.igscraper.yaml` in your home directory or project root:

```yaml
instagram:
  session_id: "your_session_id"
  csrf_token: "your_csrf_token"
  
rate_limit:
  requests_per_minute: 60
  
download:
  concurrent_downloads: 3
  download_timeout: 30s
  
output:
  base_directory: "./downloads"
  create_user_folders: true
```

See `.igscraper.yaml.example` for all available options.

## Getting Instagram Credentials

See [docs/extract_instagram_cookies.md](docs/extract_instagram_cookies.md) for detailed instructions on how to extract your Instagram session cookies.

## Rate Limiting

The scraper implements intelligent rate limiting:
- Default: 60 requests per minute (configurable)
- Automatic cooldown when limits are reached
- Visual progress tracking
- Desktop notifications for rate limit events

## Advanced Features

### Interactive TUI Mode

Enable the beautiful terminal UI for real-time progress tracking:

```bash
./igscraper --tui username
```

The TUI provides:
- Real-time download progress bars
- Live speed and ETA calculations
- Rate limit visualization
- Download queue status
- System logs with color coding
- Interactive controls (pause/resume with 'p', quit with 'q')

### Concurrent Downloads
Configure the number of simultaneous downloads:
```bash
./igscraper --concurrent 5 username
```

### Custom Output Directory
```bash
./igscraper --output /path/to/photos username
```

### Disable Notifications
```bash
./igscraper --notifications=false username
```

## Development

### Project Structure
```
igscraper/
├── cmd/igscraper/      # Main application entry point
├── pkg/
│   ├── config/         # Configuration management
│   ├── instagram/      # Instagram API client
│   ├── scraper/        # Core scraping logic
│   ├── storage/        # File storage and deduplication
│   ├── ratelimit/      # Rate limiting algorithms
│   └── ui/             # Terminal UI and notifications
│       └── tui/        # Interactive terminal UI
```

### Running Tests
```bash
go test ./...
```

### Building for Different Platforms
```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o igscraper-linux cmd/igscraper/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o igscraper-macos cmd/igscraper/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o igscraper.exe cmd/igscraper/main.go
```

## License

MIT License - See LICENSE file for details

## Disclaimer

I'm not responsible if you get your Instagram account banned, anon.
