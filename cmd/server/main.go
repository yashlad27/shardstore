package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/yashlad/distributed-file-store/api/proto"
	"github.com/yashlad/distributed-file-store/internal/manager"
	"github.com/yashlad/distributed-file-store/internal/metadata"
	"github.com/yashlad/distributed-file-store/internal/server"
)

const (
	defaultPort        = "50051"
	defaultMongoURI    = "mongodb://localhost:27017"
	defaultDatabase    = "filestore"
	defaultReplicaFactor = 2
)

func main() {
	// Get configuration from environment variables
	port := getEnv("PORT", defaultPort)
	mongoURI := getEnv("MONGO_URI", defaultMongoURI)
	database := getEnv("DATABASE", defaultDatabase)

	log.Printf("Starting Distributed File Store Server...")
	log.Printf("Port: %s", port)
	log.Printf("MongoDB URI: %s", mongoURI)

	// Initialize metadata store
	metadataStore, err := metadata.NewMetadataStore(mongoURI, database)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	log.Printf("✓ Connected to MongoDB")

	// Initialize file manager
	fileManager := manager.NewFileManager(metadataStore, defaultReplicaFactor)

	// Register storage nodes
	// In production, these would be separate servers
	nodes := []struct {
		id   string
		path string
	}{
		{"node-1", "/tmp/filestore/node-1"},
		{"node-2", "/tmp/filestore/node-2"},
		{"node-3", "/tmp/filestore/node-3"},
	}

	for _, node := range nodes {
		if err := fileManager.RegisterNode(node.id, node.path); err != nil {
			log.Fatalf("Failed to register node %s: %v", node.id, err)
		}
		log.Printf("✓ Registered storage node: %s", node.id)
	}

	// Create gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB
		grpc.MaxSendMsgSize(10 * 1024 * 1024), // 10MB
	)

	// Register FileStore service
	fileStoreServer := server.NewFileStoreServer(fileManager)
	pb.RegisterFileStoreServer(grpcServer, fileStoreServer)

	// Enable reflection for debugging with grpcurl
	reflection.Register(grpcServer)

	log.Printf("✓ gRPC server listening on port %s", port)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("\nShutting down gracefully...")
		grpcServer.GracefulStop()
		log.Println("✓ Server stopped")
	}()

	// Start server
	log.Println("🚀 Server ready to accept connections")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
