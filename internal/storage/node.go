package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Node represents a storage node that stores file chunks
type Node struct {
	ID          string
	StoragePath string
	mu          sync.RWMutex
}

// NewNode creates a new storage node
func NewNode(id, storagePath string) (*Node, error) {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, err
	}

	return &Node{
		ID:          id,
		StoragePath: storagePath,
	}, nil
}

// StoreFile stores a file on this node
func (n *Node) StoreFile(fileID, versionID string, data []byte) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Create version directory
	versionPath := filepath.Join(n.StoragePath, fileID, versionID)
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		return err
	}

	// Write file data
	filePath := filepath.Join(versionPath, "data")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return err
	}

	// Store checksum
	checksum := n.calculateChecksum(data)
	checksumPath := filepath.Join(versionPath, "checksum")
	if err := os.WriteFile(checksumPath, []byte(checksum), 0644); err != nil {
		return err
	}

	return nil
}

// RetrieveFile retrieves a file from this node
func (n *Node) RetrieveFile(fileID, versionID string) ([]byte, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	versionPath := filepath.Join(n.StoragePath, fileID, versionID)
	filePath := filepath.Join(versionPath, "data")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Verify checksum
	checksumPath := filepath.Join(versionPath, "checksum")
	storedChecksum, err := os.ReadFile(checksumPath)
	if err != nil {
		return nil, err
	}

	calculatedChecksum := n.calculateChecksum(data)
	if calculatedChecksum != string(storedChecksum) {
		return nil, fmt.Errorf("checksum mismatch: data corrupted")
	}

	return data, nil
}

// DeleteFile deletes a file from this node
func (n *Node) DeleteFile(fileID, versionID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	versionPath := filepath.Join(n.StoragePath, fileID, versionID)
	
	// Remove version directory
	if err := os.RemoveAll(versionPath); err != nil {
		return err
	}

	// Check if file directory is empty and remove it
	filePath := filepath.Join(n.StoragePath, fileID)
	entries, err := os.ReadDir(filePath)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return os.Remove(filePath)
	}

	return nil
}

// DeleteAllVersions deletes all versions of a file
func (n *Node) DeleteAllVersions(fileID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	filePath := filepath.Join(n.StoragePath, fileID)
	return os.RemoveAll(filePath)
}

// FileExists checks if a file exists on this node
func (n *Node) FileExists(fileID, versionID string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	versionPath := filepath.Join(n.StoragePath, fileID, versionID)
	filePath := filepath.Join(versionPath, "data")
	
	_, err := os.Stat(filePath)
	return err == nil
}

// GetStorageSize returns the total storage used by this node
func (n *Node) GetStorageSize() (int64, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var size int64
	err := filepath.Walk(n.StoragePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// ReplicateFile replicates a file from source data
func (n *Node) ReplicateFile(fileID, versionID string, source io.Reader) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	versionPath := filepath.Join(n.StoragePath, fileID, versionID)
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(versionPath, "data")
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy data and calculate checksum simultaneously
	hasher := sha256.New()
	multiWriter := io.MultiWriter(file, hasher)
	
	if _, err := io.Copy(multiWriter, source); err != nil {
		return err
	}

	// Store checksum
	checksum := hex.EncodeToString(hasher.Sum(nil))
	checksumPath := filepath.Join(versionPath, "checksum")
	return os.WriteFile(checksumPath, []byte(checksum), 0644)
}

// calculateChecksum calculates SHA-256 checksum of data
func (n *Node) calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// ListFiles returns all file IDs stored on this node
func (n *Node) ListFiles() ([]string, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	entries, err := os.ReadDir(n.StoragePath)
	if err != nil {
		return nil, err
	}

	var fileIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			fileIDs = append(fileIDs, entry.Name())
		}
	}

	return fileIDs, nil
}
