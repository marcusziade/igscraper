# Scripts Directory

This directory contains utility scripts for the IGScraper project.

## Available Scripts

### preview-docs.sh

Preview the documentation site locally with **live reload** functionality.

```bash
# Run with default port (8888)
./scripts/preview-docs.sh

# Run with custom port
./scripts/preview-docs.sh 3000
```

The script will:
- Start a local HTTP server with **automatic page refresh** when files change
- Watch for changes in HTML, CSS, JS, JSON, and MD files
- Make the site accessible from any device on your network
- Display both local and network URLs for access
- Automatically open your default browser
- Show colored output for file changes

Press `Ctrl+C` to stop the server.

**Live Reload**: The page automatically refreshes within 500ms when you save changes to any watched file. No manual refresh needed!

**Network Access**: The server is accessible from other devices on your network using the displayed network URL (e.g., `http://192.168.1.100:8888`).

### live-server.py

The Python script that powers the live reload functionality. Features:
- Injects live reload JavaScript into HTML pages
- Watches for file changes in the docs directory
- Provides colored terminal output
- Handles errors gracefully
- Supports both Python 2 and 3