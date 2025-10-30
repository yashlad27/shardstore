package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yashlad/distributed-file-store/internal/hash"
	"github.com/yashlad/distributed-file-store/internal/metadata"
	"github.com/yashlad/distributed-file-store/internal/storage"
)

// FileManager coordinates file operations across storage nodes
type FileManager struct {
	nodes         map[string]*storage.Node
	hashRing      *hash.ConsistentHash
	metadataStore metadata.Store
	replicaFactor int
}

// NewFileManager creates a new file manager
func NewFileManager(metadataStore metadata.Store, replicaFactor int) *FileManager {
	return &FileManager{
		nodes:         make(map[string]*storage.Node),
		hashRing:      hash.NewConsistentHash(150, replicaFactor),
		metadataStore: metadataStore,
		replicaFactor: replicaFactor,
	}
}

// RegisterNode registers a storage node
func (fm *FileManager) RegisterNode(nodeID, storagePath string) error {
	node, err := storage.NewNode(nodeID, storagePath)
	if err != nil {
		return err
	}

	fm.nodes[nodeID] = node
	fm.hashRing.AddNode(nodeID)
	
	return nil
}

// UnregisterNode removes a storage node
func (fm *FileManager) UnregisterNode(nodeID string) {
	delete(fm.nodes, nodeID)
	fm.hashRing.RemoveNode(nodeID)
}

// UploadFile handles file upload with sharding and replication
func (fm *FileManager) UploadFile(ctx context.Context, filename string, data []byte, contentType string) (*metadata.FileMetadata, error) {
	fileID := uuid.New().String()
	versionID := uuid.New().String()

	// Get nodes for this file using consistent hashing
	nodeIDs := fm.hashRing.GetNodes(fileID)
	if len(nodeIDs) == 0 {
		return nil, fmt.Errorf("no storage nodes available")
	}

	// Store file on all replica nodes
	var storedNodes []string
	for _, nodeID := range nodeIDs {
		node, exists := fm.nodes[nodeID]
		if !exists {
			continue
		}

		if err := node.StoreFile(fileID, versionID, data); err != nil {
			// Log error but continue with other replicas
			fmt.Printf("Failed to store on node %s: %v\n", nodeID, err)
			continue
		}

		storedNodes = append(storedNodes, nodeID)
	}

	if len(storedNodes) == 0 {
		return nil, fmt.Errorf("failed to store file on any node")
	}

	// Create metadata
	fileMetadata := &metadata.FileMetadata{
		FileID:      fileID,
		Filename:    filename,
		Size:        int64(len(data)),
		ContentType: contentType,
		Replicas:    storedNodes,
		Versions: []metadata.Version{
			{
				VersionID: versionID,
				Size:      int64(len(data)),
				Nodes:     storedNodes,
				CreatedAt: time.Now(),
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save metadata to MongoDB
	if err := fm.metadataStore.SaveMetadata(ctx, fileMetadata); err != nil {
		// Cleanup: delete from storage nodes
		fm.cleanupFailedUpload(fileID, versionID, storedNodes)
		return nil, err
	}

	return fileMetadata, nil
}

// DownloadFile retrieves a file from storage
func (fm *FileManager) DownloadFile(ctx context.Context, fileID, versionID string) ([]byte, *metadata.FileMetadata, error) {
	// Get metadata
	fileMeta, err := fm.metadataStore.GetMetadata(ctx, fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("file not found: %w", err)
	}

	// Determine which version to download
	var targetVersion *metadata.Version
	if versionID == "" {
		// Get latest version
		if len(fileMeta.Versions) == 0 {
			return nil, nil, fmt.Errorf("no versions available")
		}
		targetVersion = &fileMeta.Versions[len(fileMeta.Versions)-1]
	} else {
		// Find specific version
		for i := range fileMeta.Versions {
			if fileMeta.Versions[i].VersionID == versionID {
				targetVersion = &fileMeta.Versions[i]
				break
			}
		}
		if targetVersion == nil {
			return nil, nil, fmt.Errorf("version not found")
		}
	}

	// Try to retrieve from any replica node
	var data []byte
	var lastErr error
	
	for _, nodeID := range targetVersion.Nodes {
		node, exists := fm.nodes[nodeID]
		if !exists {
			continue
		}

		data, lastErr = node.RetrieveFile(fileID, targetVersion.VersionID)
		if lastErr == nil {
			return data, fileMeta, nil
		}
	}

	return nil, nil, fmt.Errorf("failed to retrieve file from any replica: %w", lastErr)
}

// DeleteFile deletes a file and its metadata
func (fm *FileManager) DeleteFile(ctx context.Context, fileID string) error {
	// Get metadata to find all nodes
	fileMeta, err := fm.metadataStore.GetMetadata(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Delete from all replica nodes
	for _, nodeID := range fileMeta.Replicas {
		node, exists := fm.nodes[nodeID]
		if !exists {
			continue
		}

		if err := node.DeleteAllVersions(fileID); err != nil {
			fmt.Printf("Failed to delete from node %s: %v\n", nodeID, err)
		}
	}

	// Delete metadata
	return fm.metadataStore.DeleteMetadata(ctx, fileID)
}

// GetFileInfo retrieves file metadata
func (fm *FileManager) GetFileInfo(ctx context.Context, fileID string) (*metadata.FileMetadata, error) {
	return fm.metadataStore.GetMetadata(ctx, fileID)
}

// ListFiles lists all files with pagination
func (fm *FileManager) ListFiles(ctx context.Context, page, pageSize int32) ([]*metadata.FileMetadata, int64, error) {
	return fm.metadataStore.ListMetadata(ctx, page, pageSize)
}

// GetVersion retrieves a specific version of a file
func (fm *FileManager) GetVersion(ctx context.Context, fileID, versionID string) ([]byte, error) {
	data, _, err := fm.DownloadFile(ctx, fileID, versionID)
	return data, err
}

// cleanupFailedUpload removes file data from nodes on upload failure
func (fm *FileManager) cleanupFailedUpload(fileID, versionID string, nodeIDs []string) {
	for _, nodeID := range nodeIDs {
		node, exists := fm.nodes[nodeID]
		if !exists {
			continue
		}
		node.DeleteFile(fileID, versionID)
	}
}

// HealthCheck checks the health of all storage nodes
func (fm *FileManager) HealthCheck() map[string]bool {
	health := make(map[string]bool)
	
	for nodeID, node := range fm.nodes {
		_, err := node.GetStorageSize()
		health[nodeID] = (err == nil)
	}
	
	return health
}

// GetNodeCount returns the number of active storage nodes
func (fm *FileManager) GetNodeCount() int {
	return len(fm.nodes)
}
