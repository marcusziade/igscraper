#!/bin/bash

# Preview documentation site locally

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Default port (8888 is less commonly used than 8080)
PORT=${1:-8888}

# Function to kill existing process on port
kill_port() {
    local pids=$(lsof -ti :$PORT 2>/dev/null)
    if [ ! -z "$pids" ]; then
        echo -e "${YELLOW}Killing existing process on port $PORT...${NC}"
        kill -9 $pids 2>/dev/null || true
        sleep 0.5  # 500ms delay to ensure port is freed
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

# Function to get local IP address
get_local_ip() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        ip addr show | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | grep -v '127.0.0.1' | head -n1
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        ifconfig | grep "inet " | grep -v 127.0.0.1 | awk '{print $2}' | head -n1
    else
        echo "localhost"
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

# Kill any existing process on the port
kill_port

# Get local IP address
LOCAL_IP=$(get_local_ip)

# Check if live-server.py exists
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
LIVE_SERVER="$SCRIPT_DIR/live-server.py"

if [ -f "$LIVE_SERVER" ]; then
    # Use the live reload server
    echo -e "${GREEN}Starting server with live reload...${NC}"
    echo -e "${YELLOW}Files will auto-refresh when changed!${NC}"
    $PYTHON_CMD "$LIVE_SERVER" $PORT
else
    # Fallback to standard server
    echo -e "${YELLOW}Starting server accessible on your network${NC}"
    echo -e "${GREEN}Local access:   http://localhost:$PORT${NC}"
    echo -e "${GREEN}Network access: http://$LOCAL_IP:$PORT${NC}"
    echo -e "${YELLOW}Press Ctrl+C to stop the server${NC}"
    echo ""
    
    # Change to docs directory
    cd docs
    
    # Add initial delay before starting server
    sleep 0.5  # 500ms delay before starting
    
    # Try to open browser after a short delay
    (sleep 2 && open_browser "http://localhost:$PORT") &
    
    # Start Python HTTP server bound to all interfaces (0.0.0.0)
    if $PYTHON_CMD -c "import sys; sys.exit(0 if sys.version_info[0] >= 3 else 1)" 2>/dev/null; then
        # Python 3 - bind to all interfaces
        $PYTHON_CMD -m http.server $PORT --bind 0.0.0.0
    else
        # Python 2 - already binds to all interfaces by default
        $PYTHON_CMD -m SimpleHTTPServer $PORT
    fi
fi