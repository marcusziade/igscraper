#!/bin/bash

# Preview documentation site locally

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Default port
PORT=${1:-8080}

# Function to check if port is in use
check_port() {
    if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${RED}Port $PORT is already in use${NC}"
        echo "Please specify a different port: ./scripts/preview-docs.sh 8081"
        exit 1
    fi
}

# Function to open browser
open_browser() {
    local url=$1
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        xdg-open "$url" 2>/dev/null || echo "Please open $url in your browser"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        open "$url"
    elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
        start "$url"
    else
        echo "Please open $url in your browser"
    fi
}

# Main script
echo -e "${GREEN}Starting IGScraper documentation preview...${NC}"

# Check if docs directory exists
if [ ! -d "docs" ]; then
    echo -e "${RED}Error: docs directory not found${NC}"
    echo "Please run this script from the project root"
    exit 1
fi

# Check if Python is installed
if ! command -v python3 &> /dev/null; then
    if ! command -v python &> /dev/null; then
        echo -e "${RED}Python is not installed${NC}"
        echo "Please install Python 3 to preview the documentation"
        exit 1
    else
        PYTHON_CMD="python"
    fi
else
    PYTHON_CMD="python3"
fi

# Check port availability
check_port

# Start the server
echo -e "${YELLOW}Starting local server on http://localhost:$PORT${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop the server${NC}"
echo ""

# Change to docs directory
cd docs

# Try to open browser after a short delay
(sleep 2 && open_browser "http://localhost:$PORT") &

# Start Python HTTP server
if $PYTHON_CMD -c "import sys; sys.exit(0 if sys.version_info[0] >= 3 else 1)" 2>/dev/null; then
    # Python 3
    $PYTHON_CMD -m http.server $PORT
else
    # Python 2
    $PYTHON_CMD -m SimpleHTTPServer $PORT
fi