.PHONY: proto build run clean docker-build docker-up docker-down test client

# Variables
PROTO_DIR = api/proto
PROTO_FILE = $(PROTO_DIR)/filestore.proto
SERVER_BIN = bin/server
CLIENT_BIN = bin/client

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@mkdir -p $(PROTO_DIR)
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_FILE)
	@echo "✓ Protobuf code generated"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies installed"

# Build server and client
build: proto
	@echo "Building server..."
	@mkdir -p bin
	go build -o $(SERVER_BIN) ./cmd/server
	@echo "Building client..."
	go build -o $(CLIENT_BIN) ./cmd/client
	@echo "✓ Build complete"

# Run server locally
run: build
	@echo "Starting server..."
	./$(SERVER_BIN)

# Run client
client: build
	@echo "Client ready. Usage: make client ARGS='upload test.txt'"
	./$(CLIENT_BIN) $(ARGS)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf /tmp/filestore/
	rm -f $(PROTO_DIR)/*.pb.go
	@echo "✓ Cleaned"

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker-compose build
	@echo "✓ Docker image built"

docker-up:
	@echo "Starting services..."
	docker-compose up -d
	@echo "✓ Services started"

docker-down:
	@echo "Stopping services..."
	docker-compose down
	@echo "✓ Services stopped"

docker-logs:
	docker-compose logs -f filestore-server

# Test commands
test:
	@echo "Running unit tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

test-race:
	@echo "Running tests with race detector..."
	go test -race ./...

test-integration:
	@echo "Running integration tests..."
	./test.sh

test-advanced:
	@echo "Running advanced integration tests..."
	./test_advanced.sh

test-load:
	@echo "Running load tests..."
	./test_load.sh

test-all: test test-integration test-advanced
	@echo "✓ All tests completed!"

test-bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "✓ Development tools installed"

# Quick start
start: proto deps docker-up
	@echo "✓ Distributed File Store is running!"
	@echo ""
	@echo "Server: localhost:50051"
	@echo "MongoDB: localhost:27017"
	@echo ""
	@echo "Upload a file: make client ARGS='upload /path/to/file.txt'"
	@echo "List files: make client ARGS='list'"

help:
	@echo "Distributed File Store - Available Commands:"
	@echo ""
	@echo "  make proto        - Generate protobuf code"
	@echo "  make deps         - Install Go dependencies"
	@echo "  make build        - Build server and client"
	@echo "  make run          - Run server locally"
	@echo "  make client       - Run client (use ARGS='...')"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-up    - Start Docker services"
	@echo "  make docker-down  - Stop Docker services"
	@echo "  make docker-logs  - View server logs"
	@echo ""
	@echo "  make test         - Run tests"
	@echo "  make dev-setup    - Install development tools"
	@echo "  make start        - Quick start (proto + deps + docker)"
