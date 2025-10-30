package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/yashlad/distributed-file-store/api/proto"
)

const (
	chunkSize = 1024 * 1024 // 1MB
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	serverAddr := getEnv("SERVER_ADDR", "localhost:50051")

	// Connect to server
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewFileStoreClient(conn)

	command := os.Args[1]

	switch command {
	case "upload":
		if len(os.Args) < 3 {
			log.Fatal("Usage: client upload <filepath>")
		}
		uploadFile(client, os.Args[2])

	case "download":
		if len(os.Args) < 4 {
			log.Fatal("Usage: client download <file_id> <output_path>")
		}
		downloadFile(client, os.Args[2], os.Args[3])

	case "delete":
		if len(os.Args) < 3 {
			log.Fatal("Usage: client delete <file_id>")
		}
		deleteFile(client, os.Args[2])

	case "info":
		if len(os.Args) < 3 {
			log.Fatal("Usage: client info <file_id>")
		}
		getFileInfo(client, os.Args[2])

	case "list":
		listFiles(client)

	default:
		printUsage()
		os.Exit(1)
	}
}

func uploadFile(client pb.FileStoreClient, filepath string) {
	log.Printf("Uploading file: %s", filepath)

	file, err := os.Open(filepath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Fatalf("Failed to stat file: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := client.Upload(ctx)
	if err != nil {
		log.Fatalf("Failed to create upload stream: %v", err)
	}

	// Send file in chunks
	buffer := make([]byte, chunkSize)
	totalSent := int64(0)
	
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}

		req := &pb.UploadRequest{
			Filename:    stat.Name(),
			Chunk:       buffer[:n],
			TotalSize:   stat.Size(),
			ContentType: "application/octet-stream",
		}

		if err := stream.Send(req); err != nil {
			log.Fatalf("Failed to send chunk: %v", err)
		}

		totalSent += int64(n)
		progress := float64(totalSent) / float64(stat.Size()) * 100
		fmt.Printf("\rProgress: %.2f%%", progress)
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("\nFailed to receive response: %v", err)
	}

	if res.Success {
		fmt.Printf("\n✓ Upload successful!\n")
		fmt.Printf("  File ID: %s\n", res.FileId)
		fmt.Printf("  Version ID: %s\n", res.VersionId)
		fmt.Printf("  Size: %d bytes\n", res.Size)
		fmt.Printf("  Replicas: %v\n", res.NodeLocations)
	} else {
		fmt.Printf("\n✗ Upload failed: %s\n", res.Message)
	}
}

func downloadFile(client pb.FileStoreClient, fileID, outputPath string) {
	log.Printf("Downloading file: %s", fileID)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := client.Download(ctx, &pb.DownloadRequest{
		FileId: fileID,
	})
	if err != nil {
		log.Fatalf("Failed to download: %v", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	var totalReceived int64
	var totalSize int64

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Failed to receive chunk: %v", err)
		}

		if totalSize == 0 {
			totalSize = res.TotalSize
		}

		if _, err := file.Write(res.Chunk); err != nil {
			log.Fatalf("Failed to write chunk: %v", err)
		}

		totalReceived += int64(len(res.Chunk))
		progress := float64(totalReceived) / float64(totalSize) * 100
		fmt.Printf("\rProgress: %.2f%%", progress)
	}

	fmt.Printf("\n✓ Download successful! Saved to: %s\n", outputPath)
}

func deleteFile(client pb.FileStoreClient, fileID string) {
	log.Printf("Deleting file: %s", fileID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := client.Delete(ctx, &pb.DeleteRequest{
		FileId: fileID,
	})
	if err != nil {
		log.Fatalf("Failed to delete: %v", err)
	}

	if res.Success {
		fmt.Printf("✓ File deleted successfully\n")
	} else {
		fmt.Printf("✗ Delete failed: %s\n", res.Message)
	}
}

func getFileInfo(client pb.FileStoreClient, fileID string) {
	log.Printf("Getting info for file: %s", fileID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := client.GetFileInfo(ctx, &pb.FileInfoRequest{
		FileId: fileID,
	})
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}

	fmt.Printf("\n📄 File Information:\n")
	fmt.Printf("  File ID: %s\n", res.FileId)
	fmt.Printf("  Filename: %s\n", res.Filename)
	fmt.Printf("  Size: %d bytes\n", res.Size)
	fmt.Printf("  Content Type: %s\n", res.ContentType)
	fmt.Printf("  Created: %s\n", res.CreatedAt)
	fmt.Printf("  Updated: %s\n", res.UpdatedAt)
	fmt.Printf("  Versions: %v\n", res.Versions)
	fmt.Printf("  Replicas: %v\n", res.Replicas)
}

func listFiles(client pb.FileStoreClient) {
	log.Printf("Listing files...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := client.ListFiles(ctx, &pb.ListFilesRequest{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		log.Fatalf("Failed to list files: %v", err)
	}

	fmt.Printf("\n📂 Files (Total: %d):\n\n", res.TotalCount)
	
	if len(res.Files) == 0 {
		fmt.Println("  No files found")
		return
	}

	for i, file := range res.Files {
		fmt.Printf("%d. %s\n", i+1, file.Filename)
		fmt.Printf("   ID: %s\n", file.FileId)
		fmt.Printf("   Size: %d bytes\n", file.Size)
		fmt.Printf("   Created: %s\n", file.CreatedAt)
		fmt.Printf("   Replicas: %v\n\n", file.Replicas)
	}
}

func printUsage() {
	fmt.Println("Distributed File Store CLI Client")
	fmt.Println("\nUsage:")
	fmt.Println("  client upload <filepath>")
	fmt.Println("  client download <file_id> <output_path>")
	fmt.Println("  client delete <file_id>")
	fmt.Println("  client info <file_id>")
	fmt.Println("  client list")
	fmt.Println("\nEnvironment Variables:")
	fmt.Println("  SERVER_ADDR - Server address (default: localhost:50051)")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
