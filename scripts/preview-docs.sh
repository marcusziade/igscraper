#!/bin/bash
# Preview documentation with live reload

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Get the script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Check if Python 3 is installed
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}❌ Python 3 is required but not installed${NC}"
    exit 1
fi

# Kill any existing process on port 8888
echo -e "${BLUE}🔄 Checking for existing servers...${NC}"
if lsof -ti:8888 &> /dev/null; then
    kill -9 $(lsof -ti:8888) 2>/dev/null
    echo -e "${GREEN}✓ Killed existing server on port 8888${NC}"
    sleep 1
fi

# Change to project root
cd "$PROJECT_ROOT"

# Run the Python server
echo -e "${GREEN}🚀 Starting documentation server...${NC}"
python3 "$SCRIPT_DIR/preview-docs.py"