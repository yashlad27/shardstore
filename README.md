# Distributed File Store

A high-performance distributed file storage system built with Go and gRPC that automatically shards and replicates files across multiple nodes using consistent hashing.

## Overview

This project implements a production-ready distributed file storage solution similar to Amazon S3. It provides automatic file sharding, replication across nodes, version control, and strong data integrity guarantees through SHA-256 checksums. The system uses consistent hashing for optimal load distribution and supports automatic failover when nodes become unavailable.

## Key Features

- **Distributed Architecture**: Files are automatically sharded and distributed across multiple storage nodes
- **Consistent Hashing**: Uses 150 virtual nodes per physical node for optimal load balancing
- **Automatic Replication**: Configurable replica factor (default: 2) ensures data durability
- **Version Control**: Track and retrieve previous versions of files
- **Data Integrity**: SHA-256 checksums verify data correctness on every retrieval
- **Streaming I/O**: Efficient handling of large files through chunked streaming (1MB chunks)
- **MongoDB Metadata**: Scalable metadata storage with indexing for fast lookups
- **gRPC API**: High-performance remote procedure calls with Protocol Buffers
- **CLI Client**: User-friendly command-line interface with progress tracking
- **Docker Support**: Complete containerization for easy deployment

## Architecture

```
Client Application
       |
       v
  gRPC API Gateway
       |
       v
  File Manager (Coordinator)
       |
       +-- Consistent Hash Ring
       |
       +-- MongoDB (Metadata Storage)
       |
       v
   Storage Nodes
   [Node 1] [Node 2] [Node 3]
   (Primary) (Replica) (Replica)
```

### Components

**File Manager**: Coordinates all file operations, manages node selection through consistent hashing, and handles replication logic.

**Storage Nodes**: Independent storage units that persist file data with checksums. Each node can store multiple file versions.

**Metadata Store**: MongoDB database storing file metadata including versions, replica locations, and timestamps.

**Consistent Hash Ring**: Distributes files across nodes using MD5-based hashing with virtual nodes for balanced load distribution.

## Technology Stack

- **Language**: Go 1.21+
- **RPC Framework**: gRPC with Protocol Buffers
- **Database**: MongoDB 7.0
- **Containerization**: Docker & Docker Compose
- **Testing**: Go testing framework with 50+ test cases

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Protocol Buffers compiler (protoc)
- MongoDB (or use Docker)

## Installation

### Clone the Repository

```bash
git clone https://github.com/yourusername/distributed-file-store.git
cd distributed-file-store
```

### Install Dependencies

```bash
go mod download
```

### Generate Protocol Buffers

```bash
make proto
```

### Build Binaries

```bash
make build
```

This creates two binaries:
- `bin/server` - The file storage server
- `bin/client` - The CLI client

## Quick Start

### Option 1: Docker Compose (Recommended)

```bash
docker-compose up -d
```

This starts:
- MongoDB on port 27017
- File store server on port 50051

### Option 2: Manual Setup

Start MongoDB:
```bash
docker-compose up -d mongodb
```

Start the server:
```bash
./bin/server
```

The server will:
- Connect to MongoDB at localhost:27017
- Register 3 storage nodes
- Listen for gRPC connections on port 50051

## Usage

### Upload a File

```bash
./bin/client upload /path/to/file.txt
```

Output:
```
Uploading file: file.txt
Progress: 100.00%
Upload successful!
  File ID: 2ef54ab4-bb4f-4e22-9f8e-ebc1efb49c10
  Version ID: 5bcd5327-092f-4cf2-a6bc-b0097a516c94
  Size: 1024 bytes
  Replicas: [node-1 node-2]
```

### List All Files

```bash
./bin/client list
```

### Get File Information

```bash
./bin/client info <file-id>
```

### Download a File

```bash
./bin/client download <file-id> /path/to/output.txt
```

### Delete a File

```bash
./bin/client delete <file-id>
```

## Configuration

Environment variables for the server:

- `PORT` - gRPC server port (default: 50051)
- `MONGO_URI` - MongoDB connection string (default: mongodb://localhost:27017)
- `DATABASE` - Database name (default: filestore)

Example:
```bash
PORT=50052 MONGO_URI=mongodb://mongo-host:27017 ./bin/server
```

## Testing

### Run Unit Tests

```bash
make test
```

### Run with Coverage

```bash
make test-coverage
```

### Run Integration Tests

```bash
make test-integration
```

### Run Advanced Tests

```bash
make test-advanced
```

### Run Load Tests

```bash
make test-load
```

### Test Coverage Summary

- Consistent Hash: 100%
- Storage Operations: 87%
- File Manager: 73%
- Total Test Cases: 50+

See [TESTING.md](TESTING.md) for detailed testing documentation.

## Project Structure

```
distributed-file-store/
├── api/
│   └── proto/              # Protocol buffer definitions
├── cmd/
│   ├── server/            # Server application
│   └── client/            # CLI client
├── internal/
│   ├── hash/              # Consistent hashing implementation
│   ├── manager/           # File operation coordinator
│   ├── metadata/          # MongoDB metadata storage
│   ├── server/            # gRPC server implementation
│   └── storage/           # Storage node operations
├── docker-compose.yml     # Docker orchestration
├── Dockerfile            # Container image definition
├── Makefile              # Build automation
└── README.md             # This file
```

## How It Works

### File Upload Process

1. Client streams file chunks to the server via gRPC
2. File Manager generates unique file ID and version ID
3. Consistent hash determines target nodes based on file ID
4. File is replicated to N nodes (where N = replica factor)
5. Metadata is saved to MongoDB with node locations
6. Server returns file ID and replica locations to client

### File Download Process

1. Client requests file by ID
2. File Manager retrieves metadata from MongoDB
3. System attempts to download from primary replica
4. If primary fails, automatically tries other replicas
5. Downloaded data is verified against stored checksum
6. File is streamed back to client in chunks

### Node Failure Handling

When a storage node fails:
- Consistent hashing automatically routes new files to healthy nodes
- Existing files remain accessible via replica nodes
- System continues operating with reduced capacity
- Failed node can be removed and re-added without downtime

## Performance

Benchmarks on standard hardware (MacBook Pro M1):

- Upload throughput: ~30-40 MB/s
- Download throughput: ~40-50 MB/s
- Consistent hash lookup: ~500 ns/op
- Concurrent operations: 20+ simultaneous clients supported

## Development

### Adding New Features

1. Update Protocol Buffers in `api/proto/`
2. Regenerate code: `make proto`
3. Implement business logic in `internal/`
4. Add tests in `*_test.go` files
5. Update documentation

### Code Style

This project follows standard Go conventions:
- Run `go fmt` before committing
- Run `go vet` to catch common mistakes
- Maintain test coverage above 70%

## Roadmap

- TLS/mTLS support for secure communication
- Authentication and authorization layer
- Prometheus metrics and monitoring
- Web-based management UI
- S3-compatible API
- Erasure coding for storage efficiency
- Cross-datacenter replication

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Acknowledgments

Built as part of a portfolio project demonstrating:
- Distributed systems design
- Go microservices architecture
- gRPC communication patterns
- Database integration
- Container orchestration
- Comprehensive testing practices

## Contact

For questions or feedback, please open an issue on GitHub.

## References

- [gRPC Documentation](https://grpc.io/docs/)
- [Consistent Hashing](https://en.wikipedia.org/wiki/Consistent_hashing)
- [Protocol Buffers](https://developers.google.com/protocol-buffers)
- [MongoDB Go Driver](https://www.mongodb.com/docs/drivers/go/current/)
