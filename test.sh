#!/bin/bash

set -e

echo "🧪 Distributed File Store - Integration Test"
echo "============================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test files
TEST_FILE="test_upload.txt"
DOWNLOAD_FILE="test_download.txt"

# Create test file
echo "Creating test file..."
echo "Hello from Distributed File Store! $(date)" > $TEST_FILE
echo "This is a test file for demonstrating upload/download functionality." >> $TEST_FILE
echo -e "${GREEN}✓${NC} Test file created: $TEST_FILE"
echo ""

# Upload file
echo -e "${BLUE}1. Testing Upload...${NC}"
UPLOAD_OUTPUT=$(./bin/client upload $TEST_FILE)
echo "$UPLOAD_OUTPUT"

# Extract File ID from output
FILE_ID=$(echo "$UPLOAD_OUTPUT" | grep "File ID:" | awk '{print $3}')

if [ -z "$FILE_ID" ]; then
    echo -e "${RED}✗ Failed to get File ID${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Upload successful! File ID: $FILE_ID"
echo ""

# List files
echo -e "${BLUE}2. Testing List Files...${NC}"
./bin/client list
echo ""

# Get file info
echo -e "${BLUE}3. Testing Get File Info...${NC}"
./bin/client info $FILE_ID
echo ""

# Download file
echo -e "${BLUE}4. Testing Download...${NC}"
./bin/client download $FILE_ID $DOWNLOAD_FILE
echo ""

# Verify download
if diff $TEST_FILE $DOWNLOAD_FILE > /dev/null; then
    echo -e "${GREEN}✓${NC} Download verification passed! Files are identical."
else
    echo -e "${RED}✗${NC} Download verification failed! Files differ."
    exit 1
fi
echo ""

# Delete file
echo -e "${BLUE}5. Testing Delete...${NC}"
./bin/client delete $FILE_ID
echo ""

# Cleanup
rm -f $TEST_FILE $DOWNLOAD_FILE

echo ""
echo "============================================"
echo -e "${GREEN}✓ All tests passed!${NC}"
echo "============================================"
