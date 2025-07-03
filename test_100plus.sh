#!/bin/bash
# Test script to verify the app handles 100+ downloads without crashing

echo "Testing Instagram scraper with 100+ downloads..."
echo "This will run for approximately 2-3 minutes to download ~110 photos"
echo "Press Ctrl+C to stop early"
echo ""

# Run the scraper and monitor for the crash
./igscraper scrape rachelc00k --force-restart --log-level info 2>&1 | while IFS= read -r line; do
    echo "$line"
    
    # Check if we've downloaded more than 100
    if [[ "$line" =~ "Total: 110" ]]; then
        echo ""
        echo "✅ SUCCESS: Downloaded 110 photos without crashing!"
        echo "The fix for the progress bar overflow is working correctly."
        pkill -f "igscraper scrape"
        exit 0
    fi
    
    # Check for the panic we were seeing before
    if [[ "$line" =~ "panic: strings: negative Repeat count" ]]; then
        echo ""
        echo "❌ FAILURE: The app crashed with the same error!"
        echo "The progress bar fix didn't work."
        exit 1
    fi
done

echo ""
echo "Test completed. Check downloads/rachelc00k_photos for downloaded files."