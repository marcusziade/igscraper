#!/bin/bash
# Test script to demonstrate improved logging and metadata features

echo "Testing improved logging and metadata extraction..."
echo ""

# Run scraper with debug mode to show detailed logging
echo "1. Running in normal mode (clean output):"
./igscraper scrape johndoe --log-level info 2>&1 | head -20

echo ""
echo "2. Running in debug mode (detailed output):"
./igscraper scrape johndoe --log-level debug 2>&1 | head -30

echo ""
echo "3. Checking metadata files:"
find downloads/johndoe_photos -name "*.json" -type f | head -5 | while read -r file; do
    echo "Metadata for $(basename "$file"):"
    jq '.' "$file" | head -20
    echo "---"
done

echo ""
echo "Test completed. Check downloads/johndoe_photos for photos and their .json metadata files."