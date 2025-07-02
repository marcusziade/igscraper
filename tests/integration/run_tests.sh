#!/bin/bash

# Integration test runner for Instagram Scraper

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"

echo "Running Instagram Scraper Integration Tests"
echo "=========================================="

# Change to project root
cd "$PROJECT_ROOT"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if fixtures exist
if [ ! -d "$SCRIPT_DIR/fixtures" ]; then
    echo -e "${RED}Error: Fixtures directory not found${NC}"
    exit 1
fi

# Run go mod tidy to ensure dependencies
echo -e "${YELLOW}Ensuring dependencies...${NC}"
go mod tidy

# Run integration tests with coverage
echo -e "${YELLOW}Running integration tests...${NC}"
cd "$SCRIPT_DIR"

# Run tests with verbose output and coverage
go test -v -race -coverprofile=integration_coverage.out -timeout=5m ./...

# Check test results
if [ $? -eq 0 ]; then
    echo -e "${GREEN}All integration tests passed!${NC}"
    
    # Show coverage summary
    echo -e "${YELLOW}Coverage Summary:${NC}"
    go tool cover -func=integration_coverage.out | tail -1
    
    # Generate HTML coverage report
    go tool cover -html=integration_coverage.out -o integration_coverage.html
    echo -e "${GREEN}Coverage report generated: integration_coverage.html${NC}"
else
    echo -e "${RED}Integration tests failed!${NC}"
    exit 1
fi

# Run benchmarks if requested
if [[ "$1" == "--bench" ]]; then
    echo -e "${YELLOW}Running benchmarks...${NC}"
    go test -bench=. -benchmem ./...
fi

echo -e "${GREEN}Integration test run complete!${NC}"