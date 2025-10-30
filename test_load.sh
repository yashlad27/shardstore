#!/bin/bash

set -e

echo "🔥 Distributed File Store - Load Testing"
echo "========================================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
NUM_CLIENTS=${NUM_CLIENTS:-10}
NUM_OPERATIONS=${NUM_OPERATIONS:-100}
FILE_SIZE_KB=${FILE_SIZE_KB:-100}
TEST_DIR="load_test_files"
RESULTS_DIR="load_test_results"

# Create directories
mkdir -p $TEST_DIR
mkdir -p $RESULTS_DIR

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Generate test files
generate_test_files() {
    log_info "Generating test files (${FILE_SIZE_KB}KB each)..."
    
    for i in $(seq 1 $NUM_CLIENTS); do
        dd if=/dev/urandom of=$TEST_DIR/test_file_$i.bin bs=1K count=$FILE_SIZE_KB > /dev/null 2>&1
    done
    
    log_success "Generated $NUM_CLIENTS test files"
}

# Single client upload test
client_upload_test() {
    CLIENT_ID=$1
    RESULTS_FILE=$RESULTS_DIR/client_${CLIENT_ID}_results.txt
    
    SUCCESS=0
    FAILED=0
    START_TIME=$(date +%s)
    
    for i in $(seq 1 $NUM_OPERATIONS); do
        OUTPUT=$(./bin/client upload $TEST_DIR/test_file_$CLIENT_ID.bin 2>&1)
        if [ $? -eq 0 ]; then
            SUCCESS=$((SUCCESS + 1))
            FILE_ID=$(echo "$OUTPUT" | grep "File ID:" | awk '{print $3}')
            echo "$FILE_ID" >> $RESULTS_FILE
        else
            FAILED=$((FAILED + 1))
        fi
    done
    
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    echo "Client $CLIENT_ID: $SUCCESS success, $FAILED failed, Duration: ${DURATION}s" >> $RESULTS_DIR/summary.txt
}

# Concurrent upload load test
load_test_upload() {
    log_info "Starting concurrent upload test..."
    log_info "Clients: $NUM_CLIENTS, Operations per client: $NUM_OPERATIONS"
    
    rm -f $RESULTS_DIR/summary.txt
    START_TIME=$(date +%s)
    
    # Launch clients in parallel
    PIDS=()
    for i in $(seq 1 $NUM_CLIENTS); do
        client_upload_test $i &
        PIDS+=($!)
    done
    
    # Wait for all clients
    for pid in "${PIDS[@]}"; do
        wait $pid
    done
    
    END_TIME=$(date +%s)
    TOTAL_DURATION=$((END_TIME - START_TIME))
    
    # Calculate statistics
    TOTAL_SUCCESS=$(grep "success" $RESULTS_DIR/summary.txt | awk '{sum += $3} END {print sum}')
    TOTAL_FAILED=$(grep "failed" $RESULTS_DIR/summary.txt | awk '{sum += $5} END {print sum}')
    TOTAL_OPS=$((NUM_CLIENTS * NUM_OPERATIONS))
    
    echo ""
    log_success "Load test completed in ${TOTAL_DURATION}s"
    echo -e "${GREEN}Total Operations: $TOTAL_OPS${NC}"
    echo -e "${GREEN}Success: $TOTAL_SUCCESS${NC}"
    echo -e "${RED}Failed: $TOTAL_FAILED${NC}"
    
    if [ $TOTAL_SUCCESS -gt 0 ]; then
        OPS_PER_SEC=$(echo "scale=2; $TOTAL_SUCCESS / $TOTAL_DURATION" | bc)
        echo -e "${BLUE}Throughput: $OPS_PER_SEC ops/sec${NC}"
    fi
}

# Download load test
load_test_download() {
    log_info "Starting download load test..."
    
    # Get list of file IDs
    FILE_IDS=()
    for i in $(seq 1 $NUM_CLIENTS); do
        if [ -f $RESULTS_DIR/client_${i}_results.txt ]; then
            while IFS= read -r line; do
                FILE_IDS+=("$line")
            done < $RESULTS_DIR/client_${i}_results.txt
        fi
    done
    
    if [ ${#FILE_IDS[@]} -eq 0 ]; then
        log_warn "No files to download"
        return
    fi
    
    log_info "Downloading ${#FILE_IDS[@]} files..."
    
    START_TIME=$(date +%s)
    SUCCESS=0
    FAILED=0
    
    # Download files concurrently (in batches)
    BATCH_SIZE=10
    for i in $(seq 0 $BATCH_SIZE ${#FILE_IDS[@]}); do
        PIDS=()
        for j in $(seq 0 $((BATCH_SIZE - 1))); do
            IDX=$((i + j))
            if [ $IDX -lt ${#FILE_IDS[@]} ]; then
                FILE_ID=${FILE_IDS[$IDX]}
                ./bin/client download $FILE_ID $TEST_DIR/download_$IDX.bin > /dev/null 2>&1 &
                PIDS+=($!)
            fi
        done
        
        for pid in "${PIDS[@]}"; do
            wait $pid
            if [ $? -eq 0 ]; then
                SUCCESS=$((SUCCESS + 1))
            else
                FAILED=$((FAILED + 1))
            fi
        done
    done
    
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    echo ""
    log_success "Download test completed in ${DURATION}s"
    echo -e "${GREEN}Success: $SUCCESS${NC}"
    echo -e "${RED}Failed: $FAILED${NC}"
    
    if [ $SUCCESS -gt 0 ]; then
        DOWNLOAD_OPS_PER_SEC=$(echo "scale=2; $SUCCESS / $DURATION" | bc)
        echo -e "${BLUE}Download Throughput: $DOWNLOAD_OPS_PER_SEC ops/sec${NC}"
    fi
}

# Mixed workload test
load_test_mixed() {
    log_info "Starting mixed workload test..."
    log_info "Mix: 50% Upload, 30% Download, 20% List"
    
    START_TIME=$(date +%s)
    PIDS=()
    
    # Upload workers
    for i in $(seq 1 5); do
        (
            for j in $(seq 1 20); do
                ./bin/client upload $TEST_DIR/test_file_$i.bin > /dev/null 2>&1
            done
        ) &
        PIDS+=($!)
    done
    
    # Download workers
    FILE_IDS=($(find $RESULTS_DIR -name "client_*_results.txt" -exec head -1 {} \;))
    for i in $(seq 1 3); do
        (
            for j in $(seq 1 10); do
                if [ ${#FILE_IDS[@]} -gt 0 ]; then
                    RANDOM_ID=${FILE_IDS[$RANDOM % ${#FILE_IDS[@]}]}
                    ./bin/client download $RANDOM_ID /tmp/mixed_$i_$j.bin > /dev/null 2>&1
                fi
            done
        ) &
        PIDS+=($!)
    done
    
    # List workers
    for i in $(seq 1 2); do
        (
            for j in $(seq 1 10); do
                ./bin/client list > /dev/null 2>&1
                sleep 0.1
            done
        ) &
        PIDS+=($!)
    done
    
    # Wait for all workers
    for pid in "${PIDS[@]}"; do
        wait $pid
    done
    
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    log_success "Mixed workload test completed in ${DURATION}s"
}

# Cleanup test
cleanup_test() {
    log_info "Cleaning up test data..."
    
    # Delete all uploaded files
    FILE_COUNT=0
    for i in $(seq 1 $NUM_CLIENTS); do
        if [ -f $RESULTS_DIR/client_${i}_results.txt ]; then
            while IFS= read -r FILE_ID; do
                ./bin/client delete $FILE_ID > /dev/null 2>&1
                FILE_COUNT=$((FILE_COUNT + 1))
            done < $RESULTS_DIR/client_${i}_results.txt
        fi
    done
    
    log_success "Deleted $FILE_COUNT files"
    
    # Clean local files
    rm -rf $TEST_DIR
    rm -rf $RESULTS_DIR
}

# System resource monitoring
monitor_resources() {
    log_info "System Resource Usage:"
    
    # CPU usage
    CPU=$(ps aux | grep "bin/server" | grep -v grep | awk '{print $3}')
    if [ ! -z "$CPU" ]; then
        echo -e "  ${BLUE}Server CPU:${NC} ${CPU}%"
    fi
    
    # Memory usage
    MEM=$(ps aux | grep "bin/server" | grep -v grep | awk '{print $4}')
    if [ ! -z "$MEM" ]; then
        echo -e "  ${BLUE}Server Memory:${NC} ${MEM}%"
    fi
    
    # Storage usage
    if [ -d "/tmp/filestore" ]; then
        STORAGE=$(du -sh /tmp/filestore 2>/dev/null | awk '{print $1}')
        echo -e "  ${BLUE}Storage Used:${NC} ${STORAGE}"
    fi
}

# Main execution
main() {
    echo "Configuration:"
    echo "  Concurrent Clients: $NUM_CLIENTS"
    echo "  Operations per Client: $NUM_OPERATIONS"
    echo "  File Size: ${FILE_SIZE_KB}KB"
    echo ""
    
    # Generate test data
    generate_test_files
    echo ""
    
    # Run load tests
    load_test_upload
    echo ""
    
    load_test_download
    echo ""
    
    load_test_mixed
    echo ""
    
    # Monitor resources
    monitor_resources
    echo ""
    
    # Cleanup
    read -p "Clean up test data? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cleanup_test
    fi
    
    echo ""
    log_success "Load testing completed!"
    echo ""
    echo "Results saved in: $RESULTS_DIR/"
}

# Run main
main
