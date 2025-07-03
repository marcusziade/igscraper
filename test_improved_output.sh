#!/bin/bash
# Test script to demonstrate improved output

echo "Testing improved output and features..."
echo ""

# First, show what happens when checkpoint exists
echo "1. Testing checkpoint message (when not using --resume):"
./igscraper scrape testuser --log-level error 2>&1 | head -10

echo ""
echo "2. Testing clean output with new progress display:"
./igscraper scrape testuser --log-level error --force-restart 2>&1 | head -20

echo ""
echo "3. Testing with info level (default):"
./igscraper scrape testuser --log-level info --force-restart 2>&1 | head -20

echo ""
echo "4. Checking if metadata files are created:"
if [ -d "downloads/testuser_photos" ]; then
    echo "Found photos:"
    ls downloads/testuser_photos/*.json 2>/dev/null | head -5
    
    if [ -f "downloads/testuser_photos/*.json" ]; then
        echo ""
        echo "Sample metadata:"
        cat downloads/testuser_photos/*.json | head -30
    fi
fi