package manager

import (
	"context"
	"testing"
	"time"

	"github.com/yashlad/distributed-file-store/internal/metadata"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockMetadataStore implements a simple in-memory metadata store for testing
type MockMetadataStore struct {
	files map[string]*metadata.FileMetadata
}

func NewMockMetadataStore() *MockMetadataStore {
	return &MockMetadataStore{
		files: make(map[string]*metadata.FileMetadata),
	}
}

func (m *MockMetadataStore) SaveMetadata(ctx context.Context, meta *metadata.FileMetadata) error {
	meta.UpdatedAt = time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now()
	}
	if meta.ID.IsZero() {
		meta.ID = primitive.NewObjectID()
	}
	m.files[meta.FileID] = meta
	return nil
}

func (m *MockMetadataStore) GetMetadata(ctx context.Context, fileID string) (*metadata.FileMetadata, error) {
	meta, exists := m.files[fileID]
	if !exists {
		return nil, ErrFileNotFound
	}
	return meta, nil
}

func (m *MockMetadataStore) DeleteMetadata(ctx context.Context, fileID string) error {
	delete(m.files, fileID)
	return nil
}

func (m *MockMetadataStore) ListMetadata(ctx context.Context, page, pageSize int32) ([]*metadata.FileMetadata, int64, error) {
	files := make([]*metadata.FileMetadata, 0, len(m.files))
	for _, file := range m.files {
		files = append(files, file)
	}
	return files, int64(len(files)), nil
}

func (m *MockMetadataStore) AddVersion(ctx context.Context, fileID string, version metadata.Version) error {
	meta, exists := m.files[fileID]
	if !exists {
		return ErrFileNotFound
	}
	meta.Versions = append(meta.Versions, version)
	meta.UpdatedAt = time.Now()
	return nil
}

func (m *MockMetadataStore) Close(ctx context.Context) error {
	return nil
}

var ErrFileNotFound = &MockError{"file not found"}

type MockError struct {
	msg string
}

func (e *MockError) Error() string {
	return e.msg
}

func setupTestFileManager(t *testing.T) *FileManager {
	mockStore := NewMockMetadataStore()
	fm := NewFileManager(mockStore, 2)

	// Register test nodes
	tempDir := t.TempDir()
	for i := 1; i <= 3; i++ {
		nodeID := "test-node-" + string(rune('0'+i))
		storagePath := tempDir + "/" + nodeID
		if err := fm.RegisterNode(nodeID, storagePath); err != nil {
			t.Fatalf("Failed to register node: %v", err)
		}
	}

	return fm
}

func TestNewFileManager(t *testing.T) {
	mockStore := NewMockMetadataStore()

	t.Run("create with default replica factor", func(t *testing.T) {
		fm := NewFileManager(mockStore, 2)
		if fm == nil {
			t.Error("FileManager is nil")
		}
		if fm.replicaFactor != 2 {
			t.Errorf("replicaFactor = %d, want 2", fm.replicaFactor)
		}
	})

	t.Run("create with custom replica factor", func(t *testing.T) {
		fm := NewFileManager(mockStore, 3)
		if fm.replicaFactor != 3 {
			t.Errorf("replicaFactor = %d, want 3", fm.replicaFactor)
		}
	})
}

func TestRegisterNode(t *testing.T) {
	fm := setupTestFileManager(t)

	t.Run("node count", func(t *testing.T) {
		count := fm.GetNodeCount()
		if count != 3 {
			t.Errorf("node count = %d, want 3", count)
		}
	})

	t.Run("register new node", func(t *testing.T) {
		tempDir := t.TempDir()
		err := fm.RegisterNode("new-node", tempDir+"/new-node")
		if err != nil {
			t.Errorf("RegisterNode failed: %v", err)
		}
		if fm.GetNodeCount() != 4 {
			t.Errorf("node count = %d, want 4", fm.GetNodeCount())
		}
	})
}

func TestUnregisterNode(t *testing.T) {
	fm := setupTestFileManager(t)

	t.Run("unregister existing node", func(t *testing.T) {
		fm.UnregisterNode("test-node-1")
		if fm.GetNodeCount() != 2 {
			t.Errorf("node count = %d, want 2", fm.GetNodeCount())
		}
	})

	t.Run("unregister non-existing node", func(t *testing.T) {
		initialCount := fm.GetNodeCount()
		fm.UnregisterNode("non-existing-node")
		if fm.GetNodeCount() != initialCount {
			t.Error("node count changed when unregistering non-existing node")
		}
	})
}

func TestUploadFile(t *testing.T) {
	fm := setupTestFileManager(t)
	ctx := context.Background()

	t.Run("upload small file", func(t *testing.T) {
		data := []byte("Hello, World!")
		meta, err := fm.UploadFile(ctx, "test.txt", data, "text/plain")
		if err != nil {
			t.Fatalf("UploadFile failed: %v", err)
		}

		if meta.FileID == "" {
			t.Error("FileID is empty")
		}
		if meta.Filename != "test.txt" {
			t.Errorf("Filename = %s, want test.txt", meta.Filename)
		}
		if meta.Size != int64(len(data)) {
			t.Errorf("Size = %d, want %d", meta.Size, len(data))
		}
		if len(meta.Versions) != 1 {
			t.Errorf("Versions count = %d, want 1", len(meta.Versions))
		}
		if len(meta.Replicas) < 1 {
			t.Error("No replicas created")
		}
	})

	t.Run("upload large file", func(t *testing.T) {
		data := make([]byte, 5*1024*1024) // 5MB
		meta, err := fm.UploadFile(ctx, "large.bin", data, "application/octet-stream")
		if err != nil {
			t.Fatalf("UploadFile failed for large file: %v", err)
		}
		if meta.Size != int64(len(data)) {
			t.Errorf("Size = %d, want %d", meta.Size, len(data))
		}
	})

	t.Run("upload with replication", func(t *testing.T) {
		data := []byte("Replicated data")
		meta, err := fm.UploadFile(ctx, "replica.txt", data, "text/plain")
		if err != nil {
			t.Fatalf("UploadFile failed: %v", err)
		}

		// Should have replicas based on replica factor
		if len(meta.Replicas) < 1 {
			t.Error("File not replicated")
		}
		if len(meta.Replicas) > fm.replicaFactor {
			t.Errorf("Too many replicas: %d, want max %d", len(meta.Replicas), fm.replicaFactor)
		}
	})
}

func TestDownloadFile(t *testing.T) {
	fm := setupTestFileManager(t)
	ctx := context.Background()

	t.Run("download existing file", func(t *testing.T) {
		originalData := []byte("Download test data")
		meta, _ := fm.UploadFile(ctx, "download.txt", originalData, "text/plain")

		downloadedData, _, err := fm.DownloadFile(ctx, meta.FileID, "")
		if err != nil {
			t.Fatalf("DownloadFile failed: %v", err)
		}

		if len(downloadedData) != len(originalData) {
			t.Errorf("Downloaded data size = %d, want %d", len(downloadedData), len(originalData))
		}
	})

	t.Run("download non-existing file", func(t *testing.T) {
		_, _, err := fm.DownloadFile(ctx, "non-existing-id", "")
		if err == nil {
			t.Error("expected error for non-existing file")
		}
	})

	t.Run("download specific version", func(t *testing.T) {
		data := []byte("Version test")
		meta, _ := fm.UploadFile(ctx, "version.txt", data, "text/plain")
		versionID := meta.Versions[0].VersionID

		downloadedData, _, err := fm.DownloadFile(ctx, meta.FileID, versionID)
		if err != nil {
			t.Fatalf("DownloadFile with version failed: %v", err)
		}
		if len(downloadedData) != len(data) {
			t.Error("Downloaded data size mismatch")
		}
	})
}

func TestDeleteFile(t *testing.T) {
	fm := setupTestFileManager(t)
	ctx := context.Background()

	t.Run("delete existing file", func(t *testing.T) {
		data := []byte("File to delete")
		meta, _ := fm.UploadFile(ctx, "delete.txt", data, "text/plain")

		err := fm.DeleteFile(ctx, meta.FileID)
		if err != nil {
			t.Errorf("DeleteFile failed: %v", err)
		}

		// Try to download - should fail
		_, _, err = fm.DownloadFile(ctx, meta.FileID, "")
		if err == nil {
			t.Error("file still exists after deletion")
		}
	})

	t.Run("delete non-existing file", func(t *testing.T) {
		err := fm.DeleteFile(ctx, "non-existing-id")
		if err == nil {
			t.Error("expected error when deleting non-existing file")
		}
	})
}

func TestGetFileInfo(t *testing.T) {
	fm := setupTestFileManager(t)
	ctx := context.Background()

	t.Run("get info for existing file", func(t *testing.T) {
		data := []byte("Info test")
		meta, _ := fm.UploadFile(ctx, "info.txt", data, "text/plain")

		info, err := fm.GetFileInfo(ctx, meta.FileID)
		if err != nil {
			t.Fatalf("GetFileInfo failed: %v", err)
		}

		if info.FileID != meta.FileID {
			t.Error("FileID mismatch")
		}
		if info.Filename != "info.txt" {
			t.Error("Filename mismatch")
		}
	})

	t.Run("get info for non-existing file", func(t *testing.T) {
		_, err := fm.GetFileInfo(ctx, "non-existing-id")
		if err == nil {
			t.Error("expected error for non-existing file")
		}
	})
}

func TestListFiles(t *testing.T) {
	fm := setupTestFileManager(t)
	ctx := context.Background()

	t.Run("list empty storage", func(t *testing.T) {
		files, total, err := fm.ListFiles(ctx, 1, 10)
		if err != nil {
			t.Fatalf("ListFiles failed: %v", err)
		}
		if total != 0 {
			t.Errorf("total = %d, want 0", total)
		}
		if len(files) != 0 {
			t.Errorf("files count = %d, want 0", len(files))
		}
	})

	t.Run("list with files", func(t *testing.T) {
		// Upload multiple files
		for i := 0; i < 5; i++ {
			data := []byte("File " + string(rune('0'+i)))
			fm.UploadFile(ctx, "file"+string(rune('0'+i))+".txt", data, "text/plain")
		}

		files, total, err := fm.ListFiles(ctx, 1, 10)
		if err != nil {
			t.Fatalf("ListFiles failed: %v", err)
		}
		if total != 5 {
			t.Errorf("total = %d, want 5", total)
		}
		if len(files) != 5 {
			t.Errorf("files count = %d, want 5", len(files))
		}
	})
}

func TestHealthCheck(t *testing.T) {
	fm := setupTestFileManager(t)

	t.Run("all nodes healthy", func(t *testing.T) {
		health := fm.HealthCheck()
		if len(health) != 3 {
			t.Errorf("health check count = %d, want 3", len(health))
		}

		for nodeID, healthy := range health {
			if !healthy {
				t.Errorf("node %s is unhealthy", nodeID)
			}
		}
	})
}

func TestConcurrentUploads(t *testing.T) {
	fm := setupTestFileManager(t)
	ctx := context.Background()

	t.Run("concurrent uploads", func(t *testing.T) {
		done := make(chan bool)
		numUploads := 10

		for i := 0; i < numUploads; i++ {
			go func(id int) {
				data := []byte("concurrent data " + string(rune('0'+id)))
				_, err := fm.UploadFile(ctx, "concurrent"+string(rune('0'+id))+".txt", data, "text/plain")
				if err != nil {
					t.Errorf("concurrent upload failed: %v", err)
				}
				done <- true
			}(i)
		}

		for i := 0; i < numUploads; i++ {
			<-done
		}

		// Verify most files were uploaded (allowing for some failures in concurrent scenarios)
		files, total, _ := fm.ListFiles(ctx, 1, 20)
		minExpected := int64(numUploads * 8 / 10) // At least 80% should succeed
		if total < minExpected {
			t.Errorf("only %d files uploaded, want at least %d", total, minExpected)
		}
		if int64(len(files)) < minExpected {
			t.Errorf("only %d files in list, want at least %d", len(files), minExpected)
		}
	})
}

func BenchmarkUploadFile(b *testing.B) {
	mockStore := NewMockMetadataStore()
	fm := NewFileManager(mockStore, 2)
	tempDir := b.TempDir()
	fm.RegisterNode("bench-node-1", tempDir+"/node1")
	fm.RegisterNode("bench-node-2", tempDir+"/node2")

	data := make([]byte, 1024*1024) // 1MB
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fm.UploadFile(ctx, "bench.bin", data, "application/octet-stream")
	}
}

func BenchmarkDownloadFile(b *testing.B) {
	mockStore := NewMockMetadataStore()
	fm := NewFileManager(mockStore, 2)
	tempDir := b.TempDir()
	fm.RegisterNode("bench-node-1", tempDir+"/node1")
	fm.RegisterNode("bench-node-2", tempDir+"/node2")

	data := make([]byte, 1024*1024) // 1MB
	ctx := context.Background()
	meta, _ := fm.UploadFile(ctx, "bench.bin", data, "application/octet-stream")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fm.DownloadFile(ctx, meta.FileID, "")
	}
}
