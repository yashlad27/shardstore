# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git protobuf-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o filestore-server ./cmd/server

# Build the client binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o filestore-client ./cmd/client

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates grpcurl

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/filestore-server .
COPY --from=builder /build/filestore-client .

# Create storage directories
RUN mkdir -p /tmp/filestore/node-1 /tmp/filestore/node-2 /tmp/filestore/node-3

# Expose gRPC port
EXPOSE 50051

# Run the server
CMD ["./filestore-server"]
