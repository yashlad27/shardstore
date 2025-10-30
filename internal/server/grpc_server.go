package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	pb "github.com/yashlad/distributed-file-store/api/proto"
	"github.com/yashlad/distributed-file-store/internal/manager"
)

const (
	maxChunkSize = 1024 * 1024 // 1MB chunks
)

// FileStoreServer implements the gRPC FileStore service
type FileStoreServer struct {
	pb.UnimplementedFileStoreServer
	fileManager *manager.FileManager
}

// NewFileStoreServer creates a new gRPC server
func NewFileStoreServer(fileManager *manager.FileManager) *FileStoreServer {
	return &FileStoreServer{
		fileManager: fileManager,
	}
}

// Upload handles file upload with streaming
func (s *FileStoreServer) Upload(stream pb.FileStore_UploadServer) error {
	var filename string
	var contentType string
	var buffer bytes.Buffer

	// Receive chunks
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error receiving chunk: %w", err)
		}

		if filename == "" {
			filename = req.Filename
			contentType = req.ContentType
		}

		buffer.Write(req.Chunk)
	}

	// Upload file
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metadata, err := s.fileManager.UploadFile(ctx, filename, buffer.Bytes(), contentType)
	if err != nil {
		return stream.SendAndClose(&pb.UploadResponse{
			Success: false,
			Message: fmt.Sprintf("Upload failed: %v", err),
		})
	}

	// Send response
	return stream.SendAndClose(&pb.UploadResponse{
		FileId:        metadata.FileID,
		VersionId:     metadata.Versions[len(metadata.Versions)-1].VersionID,
		Size:          metadata.Size,
		NodeLocations: metadata.Replicas,
		Success:       true,
		Message:       "File uploaded successfully",
	})
}

// Download handles file download with streaming
func (s *FileStoreServer) Download(req *pb.DownloadRequest, stream pb.FileStore_DownloadServer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Download file
	data, metadata, err := s.fileManager.DownloadFile(ctx, req.FileId, req.VersionId)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Stream chunks
	totalSize := int64(len(data))
	for offset := 0; offset < len(data); offset += maxChunkSize {
		end := offset + maxChunkSize
		if end > len(data) {
			end = len(data)
		}

		chunk := data[offset:end]
		if err := stream.Send(&pb.DownloadResponse{
			Chunk:       chunk,
			TotalSize:   totalSize,
			ContentType: metadata.ContentType,
		}); err != nil {
			return fmt.Errorf("error sending chunk: %w", err)
		}
	}

	return nil
}

// Delete handles file deletion
func (s *FileStoreServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := s.fileManager.DeleteFile(ctx, req.FileId)
	if err != nil {
		return &pb.DeleteResponse{
			Success: false,
			Message: fmt.Sprintf("Delete failed: %v", err),
		}, nil
	}

	return &pb.DeleteResponse{
		Success: true,
		Message: "File deleted successfully",
	}, nil
}

// GetFileInfo retrieves file metadata
func (s *FileStoreServer) GetFileInfo(ctx context.Context, req *pb.FileInfoRequest) (*pb.FileInfoResponse, error) {
	metadata, err := s.fileManager.GetFileInfo(ctx, req.FileId)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Extract version IDs
	versions := make([]string, len(metadata.Versions))
	for i, v := range metadata.Versions {
		versions[i] = v.VersionID
	}

	return &pb.FileInfoResponse{
		FileId:      metadata.FileID,
		Filename:    metadata.Filename,
		Size:        metadata.Size,
		ContentType: metadata.ContentType,
		CreatedAt:   metadata.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   metadata.UpdatedAt.Format(time.RFC3339),
		Versions:    versions,
		Replicas:    metadata.Replicas,
	}, nil
}

// ListFiles lists all files with pagination
func (s *FileStoreServer) ListFiles(ctx context.Context, req *pb.ListFilesRequest) (*pb.ListFilesResponse, error) {
	page := req.Page
	pageSize := req.PageSize
	
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	files, total, err := s.fileManager.ListFiles(ctx, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Convert to response format
	fileInfos := make([]*pb.FileInfoResponse, len(files))
	for i, file := range files {
		versions := make([]string, len(file.Versions))
		for j, v := range file.Versions {
			versions[j] = v.VersionID
		}

		fileInfos[i] = &pb.FileInfoResponse{
			FileId:      file.FileID,
			Filename:    file.Filename,
			Size:        file.Size,
			ContentType: file.ContentType,
			CreatedAt:   file.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   file.UpdatedAt.Format(time.RFC3339),
			Versions:    versions,
			Replicas:    file.Replicas,
		}
	}

	return &pb.ListFilesResponse{
		Files:      fileInfos,
		TotalCount: int32(total),
	}, nil
}

// GetVersion retrieves a specific version of a file
func (s *FileStoreServer) GetVersion(req *pb.VersionRequest, stream pb.FileStore_GetVersionServer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Download specific version
	data, err := s.fileManager.GetVersion(ctx, req.FileId, req.VersionId)
	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	// Stream chunks
	totalSize := int64(len(data))
	for offset := 0; offset < len(data); offset += maxChunkSize {
		end := offset + maxChunkSize
		if end > len(data) {
			end = len(data)
		}

		chunk := data[offset:end]
		if err := stream.Send(&pb.DownloadResponse{
			Chunk:     chunk,
			TotalSize: totalSize,
		}); err != nil {
			return fmt.Errorf("error sending chunk: %w", err)
		}
	}

	return nil
}
