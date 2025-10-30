#!/bin/bash

set -e

echo "🧪 Distributed File Store - Advanced Integration Tests"
echo "======================================================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
SERVER_ADDR="localhost:50051"
TEST_DIR="test_files"
RESULTS_FILE="test_results.txt"

# Create test directory
mkdir -p $TEST_DIR
rm -f $RESULTS_FILE

# Helper functions
log_test() {
    echo -e "${BLUE}▶ $1${NC}"
}

log_success() {
    echo -e "${GREEN}✓ $1${NC}"
    echo "PASS: $1" >> $RESULTS_FILE
}

log_error() {
    echo -e "${RED}✗ $1${NC}"
    echo "FAIL: $1" >> $RESULTS_FILE
}

log_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Test 1: Upload Multiple File Types
test_multiple_file_types() {
    log_test "Test 1: Upload Multiple File Types"
    
    # Text file
    echo "This is a text file" > $TEST_DIR/test.txt
    ./bin/client upload $TEST_DIR/test.txt > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        log_success "Text file upload"
    else
        log_error "Text file upload"
        return 1
    fi
    
    # Binary file (image simulation)
    dd if=/dev/urandom of=$TEST_DIR/test.bin bs=1024 count=100 > /dev/null 2>&1
    ./bin/client upload $TEST_DIR/test.bin > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        log_success "Binary file upload"
    else
        log_error "Binary file upload"
        return 1
    fi
    
    # JSON file
    echo '{"key": "value", "array": [1, 2, 3]}' > $TEST_DIR/test.json
    ./bin/client upload $TEST_DIR/test.json > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        log_success "JSON file upload"
    else
        log_error "JSON file upload"
        return 1
    fi
    
    echo ""
}

# Test 2: Large File Upload
test_large_file() {
    log_test "Test 2: Large File Upload (10MB)"
    
    dd if=/dev/urandom of=$TEST_DIR/large_file.bin bs=1M count=10 > /dev/null 2>&1
    log_info "Created 10MB test file"
    
    START_TIME=$(date +%s)
    OUTPUT=$(./bin/client upload $TEST_DIR/large_file.bin)
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    if [ $? -eq 0 ]; then
        FILE_ID=$(echo "$OUTPUT" | grep "File ID:" | awk '{print $3}')
        log_success "Large file uploaded in ${DURATION}s"
        log_info "File ID: $FILE_ID"
        
        # Test download
        ./bin/client download $FILE_ID $TEST_DIR/large_file_download.bin > /dev/null 2>&1
        if [ $? -eq 0 ]; then
            log_success "Large file downloaded"
            
            # Verify integrity
            if diff $TEST_DIR/large_file.bin $TEST_DIR/large_file_download.bin > /dev/null 2>&1; then
                log_success "Large file integrity verified"
            else
                log_error "Large file integrity check failed"
                return 1
            fi
        else
            log_error "Large file download"
            return 1
        fi
    else
        log_error "Large file upload"
        return 1
    fi
    
    echo ""
}

# Test 3: Stress Test - Multiple Concurrent Uploads
test_concurrent_uploads() {
    log_test "Test 3: Concurrent Upload Stress Test (20 files)"
    
    # Create 20 small test files
    for i in {1..20}; do
        echo "Test file $i content" > $TEST_DIR/concurrent_$i.txt
    done
    
    # Upload concurrently
    PIDS=()
    for i in {1..20}; do
        ./bin/client upload $TEST_DIR/concurrent_$i.txt > $TEST_DIR/upload_$i.log 2>&1 &
        PIDS+=($!)
    done
    
    # Wait for all uploads
    FAILED=0
    for pid in "${PIDS[@]}"; do
        wait $pid
        if [ $? -ne 0 ]; then
            FAILED=$((FAILED + 1))
        fi
    done
    
    if [ $FAILED -eq 0 ]; then
        log_success "All 20 concurrent uploads succeeded"
    else
        log_error "$FAILED out of 20 uploads failed"
        return 1
    fi
    
    echo ""
}

# Test 4: File Listing and Pagination
test_file_listing() {
    log_test "Test 4: File Listing"
    
    OUTPUT=$(./bin/client list 2>&1)
    if [ $? -eq 0 ]; then
        FILE_COUNT=$(echo "$OUTPUT" | grep -c "ID:")
        log_success "List command executed (Found $FILE_COUNT files)"
        
        if [ $FILE_COUNT -gt 0 ]; then
            log_success "Files are listed correctly"
        else
            log_info "No files in storage"
        fi
    else
        log_error "List command failed"
        return 1
    fi
    
    echo ""
}

# Test 5: File Info Retrieval
test_file_info() {
    log_test "Test 5: File Info Retrieval"
    
    # Upload a test file
    echo "Info test file" > $TEST_DIR/info_test.txt
    OUTPUT=$(./bin/client upload $TEST_DIR/info_test.txt)
    FILE_ID=$(echo "$OUTPUT" | grep "File ID:" | awk '{print $3}')
    
    if [ -z "$FILE_ID" ]; then
        log_error "Could not extract File ID"
        return 1
    fi
    
    # Get file info
    INFO_OUTPUT=$(./bin/client info $FILE_ID 2>&1)
    if [ $? -eq 0 ]; then
        log_success "File info retrieved"
        
        # Verify info contains expected fields
        if echo "$INFO_OUTPUT" | grep -q "File ID:" && \
           echo "$INFO_OUTPUT" | grep -q "Filename:" && \
           echo "$INFO_OUTPUT" | grep -q "Size:" && \
           echo "$INFO_OUTPUT" | grep -q "Replicas:"; then
            log_success "File info contains all expected fields"
        else
            log_error "File info missing fields"
            return 1
        fi
    else
        log_error "File info retrieval failed"
        return 1
    fi
    
    echo ""
}

# Test 6: Upload and Download Cycle
test_upload_download_cycle() {
    log_test "Test 6: Upload/Download Cycle (10 iterations)"
    
    for i in {1..10}; do
        # Create unique content
        CONTENT="Cycle test iteration $i - $(date +%s%N)"
        echo "$CONTENT" > $TEST_DIR/cycle_$i.txt
        
        # Upload
        OUTPUT=$(./bin/client upload $TEST_DIR/cycle_$i.txt 2>&1)
        if [ $? -ne 0 ]; then
            log_error "Upload failed at iteration $i"
            return 1
        fi
        
        FILE_ID=$(echo "$OUTPUT" | grep "File ID:" | awk '{print $3}')
        
        # Download
        ./bin/client download $FILE_ID $TEST_DIR/cycle_download_$i.txt > /dev/null 2>&1
        if [ $? -ne 0 ]; then
            log_error "Download failed at iteration $i"
            return 1
        fi
        
        # Verify
        if ! diff $TEST_DIR/cycle_$i.txt $TEST_DIR/cycle_download_$i.txt > /dev/null 2>&1; then
            log_error "Content mismatch at iteration $i"
            return 1
        fi
    done
    
    log_success "All 10 upload/download cycles passed"
    echo ""
}

# Test 7: Delete Operations
test_delete_operations() {
    log_test "Test 7: Delete Operations"
    
    # Upload files to delete
    echo "Delete test 1" > $TEST_DIR/delete1.txt
    echo "Delete test 2" > $TEST_DIR/delete2.txt
    
    OUTPUT1=$(./bin/client upload $TEST_DIR/delete1.txt)
    OUTPUT2=$(./bin/client upload $TEST_DIR/delete2.txt)
    
    FILE_ID1=$(echo "$OUTPUT1" | grep "File ID:" | awk '{print $3}')
    FILE_ID2=$(echo "$OUTPUT2" | grep "File ID:" | awk '{print $3}')
    
    # Delete first file
    ./bin/client delete $FILE_ID1 > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        log_success "File deletion succeeded"
        
        # Verify file is gone
        ./bin/client download $FILE_ID1 /tmp/should_fail.txt > /dev/null 2>&1
        if [ $? -ne 0 ]; then
            log_success "Deleted file cannot be downloaded (expected)"
        else
            log_error "Deleted file still accessible"
            return 1
        fi
    else
        log_error "File deletion failed"
        return 1
    fi
    
    # Verify second file still exists
    ./bin/client download $FILE_ID2 $TEST_DIR/verify_exists.txt > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        log_success "Other files unaffected by deletion"
    else
        log_error "Other files affected by deletion"
        return 1
    fi
    
    echo ""
}

# Test 8: Edge Cases
test_edge_cases() {
    log_test "Test 8: Edge Cases"
    
    # Empty file
    touch $TEST_DIR/empty.txt
    ./bin/client upload $TEST_DIR/empty.txt > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        log_success "Empty file upload"
    else
        log_error "Empty file upload"
    fi
    
    # File with special characters in name
    echo "Special chars" > "$TEST_DIR/file with spaces & special.txt"
    ./bin/client upload "$TEST_DIR/file with spaces & special.txt" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        log_success "Special character filename"
    else
        log_info "Special character filename (may not be supported)"
    fi
    
    # Download non-existent file
    ./bin/client download "non-existent-id-12345" /tmp/should_fail.txt > /dev/null 2>&1
    if [ $? -ne 0 ]; then
        log_success "Non-existent file download fails (expected)"
    else
        log_error "Non-existent file download should fail"
    fi
    
    echo ""
}

# Test 9: Performance Metrics
test_performance() {
    log_test "Test 9: Performance Metrics"
    
    # Create 1MB test file
    dd if=/dev/urandom of=$TEST_DIR/perf_test.bin bs=1M count=1 > /dev/null 2>&1
    
    # Upload performance
    START=$(date +%s%N)
    OUTPUT=$(./bin/client upload $TEST_DIR/perf_test.bin)
    END=$(date +%s%N)
    UPLOAD_TIME=$(( (END - START) / 1000000 ))
    FILE_ID=$(echo "$OUTPUT" | grep "File ID:" | awk '{print $3}')
    
    log_info "Upload time: ${UPLOAD_TIME}ms"
    
    # Download performance
    START=$(date +%s%N)
    ./bin/client download $FILE_ID $TEST_DIR/perf_download.bin > /dev/null 2>&1
    END=$(date +%s%N)
    DOWNLOAD_TIME=$(( (END - START) / 1000000 ))
    
    log_info "Download time: ${DOWNLOAD_TIME}ms"
    
    if [ $UPLOAD_TIME -lt 5000 ] && [ $DOWNLOAD_TIME -lt 5000 ]; then
        log_success "Performance metrics acceptable"
    else
        log_info "Performance metrics recorded (no threshold enforced)"
    fi
    
    echo ""
}

# Run all tests
run_all_tests() {
    TESTS_PASSED=0
    TESTS_FAILED=0
    
    test_multiple_file_types && TESTS_PASSED=$((TESTS_PASSED + 1)) || TESTS_FAILED=$((TESTS_FAILED + 1))
    test_large_file && TESTS_PASSED=$((TESTS_PASSED + 1)) || TESTS_FAILED=$((TESTS_FAILED + 1))
    test_concurrent_uploads && TESTS_PASSED=$((TESTS_PASSED + 1)) || TESTS_FAILED=$((TESTS_FAILED + 1))
    test_file_listing && TESTS_PASSED=$((TESTS_PASSED + 1)) || TESTS_FAILED=$((TESTS_FAILED + 1))
    test_file_info && TESTS_PASSED=$((TESTS_PASSED + 1)) || TESTS_FAILED=$((TESTS_FAILED + 1))
    test_upload_download_cycle && TESTS_PASSED=$((TESTS_PASSED + 1)) || TESTS_FAILED=$((TESTS_FAILED + 1))
    test_delete_operations && TESTS_PASSED=$((TESTS_PASSED + 1)) || TESTS_FAILED=$((TESTS_FAILED + 1))
    test_edge_cases && TESTS_PASSED=$((TESTS_PASSED + 1)) || TESTS_FAILED=$((TESTS_FAILED + 1))
    test_performance && TESTS_PASSED=$((TESTS_PASSED + 1)) || TESTS_FAILED=$((TESTS_FAILED + 1))
    
    echo "======================================================"
    echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
    echo "======================================================"
    
    # Cleanup
    log_info "Cleaning up test files..."
    rm -rf $TEST_DIR
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        exit 1
    fi
}

# Main execution
echo "Starting tests at $(date)"
echo ""
run_all_tests
