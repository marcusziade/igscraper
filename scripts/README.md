# Scripts Directory

This directory contains utility scripts for the IGScraper project.

## Available Scripts

### preview-docs.sh

Preview the documentation site locally using Python's built-in HTTP server.

```bash
# Run with default port (8888)
./scripts/preview-docs.sh

# Run with custom port
./scripts/preview-docs.sh 3000
```

The script will:
- Start a local HTTP server in the docs directory
- Make the site accessible from any device on your network
- Display both local and network URLs for access
- Automatically open your default browser
- Display the documentation site with full theme switching functionality

Press `Ctrl+C` to stop the server.

**Network Access**: The server is accessible from other devices on your network using the displayed network URL (e.g., `http://192.168.1.100:8888`).