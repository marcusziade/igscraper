#!/bin/bash

# Script to run tests and generate coverage report

echo "Running tests and generating coverage report..."

# Create coverage directory
mkdir -p coverage

# Run tests for each package and generate coverage
packages=("pkg/instagram" "pkg/config" "pkg/scraper" "pkg/ratelimit" "pkg/retry" "pkg/storage" "pkg/logger")

for pkg in "${packages[@]}"; do
    echo "Testing $pkg..."
    go test -v -coverprofile="coverage/$(basename $pkg).cover" ./$pkg 2>&1 | grep -E "(ok|FAIL|coverage:|PASS:|FAIL:)" || true
done

# Merge coverage files
echo "Merging coverage files..."
echo "mode: set" > coverage/coverage.out
for file in coverage/*.cover; do
    if [ -f "$file" ]; then
        tail -n +2 "$file" >> coverage/coverage.out
    fi
done

# Generate HTML report
echo "Generating HTML coverage report..."
go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Show coverage summary
echo -e "\nCoverage Summary:"
go tool cover -func=coverage/coverage.out | grep -E "(total:|pkg/)" | sort

echo -e "\nHTML report generated at: coverage/coverage.html"