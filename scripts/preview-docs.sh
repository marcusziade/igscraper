#!/bin/bash
# Preview landing page

# Kill any existing process on port 8888
lsof -ti:8888 | xargs kill -9 2>/dev/null

# Start simple server
cd "$(dirname "$0")/../docs"
python3 -m http.server 8888 --bind 0.0.0.0