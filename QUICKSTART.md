# 🚀 Distributed File Store - Quick Start Guide

A high-performance distributed file storage system with sharding, replication, and versioning.

## ✨ Features

- **gRPC API** - High-performance streaming upload/download
- **Consistent Hashing** - Automatic load distribution across nodes
- **Replication** - Configurable replica factor for durability
- **Versioning** - Track and retrieve file versions
- **MongoDB Metadata** - Scalable metadata storage
- **Docker Ready** - Full containerization support

## 🏗️ Architecture

```
[ Client ] ──► [ gRPC API Gateway ] ──► [ File Manager ]
                                            │
                          ┌─────────────────┼─────────────────┐
                          ▼                 ▼                 ▼
                    [ Node 1 ]        [ Node 2 ]        [ Node 3 ]
                    (Primary)         (Replica)         (Replica)
                          │
                          ▼
                  [ MongoDB Metadata ]
```

## 📦 Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Protocol Buffers compiler (protoc)

## 🚀 Quick Start

### 1. Start MongoDB
```bash
docker-compose up -d mongodb
```

### 2. Build & Run Server
```bash
make build
./bin/server
```

### 3. Use CLI Client

**Upload a file:**
```bash
./bin/client upload /path/to/file.txt
```

**List all files:**
```bash
./bin/client list
```

**Get file info:**
```bash
./bin/client info <file-id>
```

**Download a file:**
```bash
./bin/client download <file-id> output.txt
```

**Delete a file:**
```bash
./bin/client delete <file-id>
```

## 🐳 Docker Deployment

**Start all services:**
```bash
docker-compose up -d
```

**View logs:**
```bash
docker-compose logs -f filestore-server
```

**Stop services:**
```bash
docker-compose down
```

## 🧪 Run Tests

```bash
./test.sh
```

## 📊 Test Results

```
✓ Upload - File sharded and replicated across nodes
✓ Download - Retrieved from replica with checksum verification
✓ List - Metadata retrieved from MongoDB
✓ Info - Version tracking working
✓ Delete - Cleaned from all replicas
```

## 🛠️ Configuration

Environment variables:
- `PORT` - gRPC server port (default: 50051)
- `MONGO_URI` - MongoDB connection string
- `DATABASE` - Database name (default: filestore)

## 📁 Project Structure

```
Distributed_File_Store/
├── api/proto/              # gRPC protocol definitions
├── cmd/
│   ├── server/            # Server application
│   └── client/            # CLI client
├── internal/
│   ├── hash/              # Consistent hashing
│   ├── manager/           # File management coordinator
│   ├── metadata/          # MongoDB operations
│   ├── server/            # gRPC server implementation
│   └── storage/           # Storage node operations
├── docker-compose.yml
├── Dockerfile
└── Makefile
```

## 🔧 Development Commands

```bash
make proto          # Generate protobuf code
make build          # Build binaries
make run            # Run server locally
make docker-build   # Build Docker image
make test           # Run tests
make clean          # Clean artifacts
```

## 🌟 Key Implementation Details

### Consistent Hashing
- 150 virtual nodes per physical node
- MD5-based hash distribution
- Automatic rebalancing on node changes

### Replication
- Configurable replica factor (default: 2)
- Automatic failover on node failure
- SHA-256 checksums for data integrity

### Storage
- File sharding with version control
- Checksum verification on retrieval
- Atomic cleanup on failure

### MongoDB Schema
```javascript
{
  file_id: "uuid",
  filename: "example.txt",
  size: 1024,
  content_type: "text/plain",
  versions: [{
    version_id: "uuid",
    nodes: ["node-1", "node-2"],
    created_at: ISODate()
  }],
  replicas: ["node-1", "node-2"],
  created_at: ISODate(),
  updated_at: ISODate()
}
```

## 📈 Performance

- Streaming uploads/downloads (1MB chunks)
- Concurrent node operations
- Efficient metadata indexing
- gRPC multiplexing

## 🔒 Production Considerations

1. **Security**: Add TLS/mTLS for gRPC
2. **Authentication**: Implement token-based auth
3. **Monitoring**: Add Prometheus metrics
4. **Logging**: Structured logging with levels
5. **Recovery**: Implement automatic node recovery
6. **Scaling**: Deploy storage nodes as separate services

## 🎯 Next Steps

- [ ] Add authentication layer
- [ ] Implement automatic node health checks
- [ ] Add Prometheus metrics
- [ ] Create web UI
- [ ] Add S3-compatible API
- [ ] Implement erasure coding

---

Built with ❤️ using Go, gRPC, and MongoDB
