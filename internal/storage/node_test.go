package storage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewNode(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("create node with new directory", func(t *testing.T) {
		storagePath := filepath.Join(tempDir, "node-test")
		node, err := NewNode("test-node", storagePath)
		if err != nil {
			t.Fatalf("NewNode failed: %v", err)
		}
		if node.ID != "test-node" {
			t.Errorf("node ID = %s, want test-node", node.ID)
		}
		if _, err := os.Stat(storagePath); os.IsNotExist(err) {
			t.Error("storage directory was not created")
		}
	})

	t.Run("create node with existing directory", func(t *testing.T) {
		storagePath := filepath.Join(tempDir, "existing-node")
		os.MkdirAll(storagePath, 0755)
		
		node, err := NewNode("existing-node", storagePath)
		if err != nil {
			t.Fatalf("NewNode failed: %v", err)
		}
		if node == nil {
			t.Error("node is nil")
		}
	})
}

func TestStoreFile(t *testing.T) {
	tempDir := t.TempDir()
	node, err := NewNode("test-node", tempDir)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}

	t.Run("store small file", func(t *testing.T) {
		data := []byte("Hello, World!")
		err := node.StoreFile("file-1", "version-1", data)
		if err != nil {
			t.Errorf("StoreFile failed: %v", err)
		}

		// Verify file exists
		filePath := filepath.Join(tempDir, "file-1", "version-1", "data")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("file was not stored")
		}

		// Verify checksum exists
		checksumPath := filepath.Join(tempDir, "file-1", "version-1", "checksum")
		if _, err := os.Stat(checksumPath); os.IsNotExist(err) {
			t.Error("checksum was not stored")
		}
	})

	t.Run("store large file", func(t *testing.T) {
		data := make([]byte, 1024*1024) // 1MB
		for i := range data {
			data[i] = byte(i % 256)
		}
		
		err := node.StoreFile("file-2", "version-1", data)
		if err != nil {
			t.Errorf("StoreFile failed for large file: %v", err)
		}
	})

	t.Run("store multiple versions", func(t *testing.T) {
		data1 := []byte("Version 1")
		data2 := []byte("Version 2")
		
		err := node.StoreFile("file-3", "v1", data1)
		if err != nil {
			t.Errorf("StoreFile v1 failed: %v", err)
		}
		
		err = node.StoreFile("file-3", "v2", data2)
		if err != nil {
			t.Errorf("StoreFile v2 failed: %v", err)
		}

		// Both versions should exist
		if !node.FileExists("file-3", "v1") {
			t.Error("version 1 does not exist")
		}
		if !node.FileExists("file-3", "v2") {
			t.Error("version 2 does not exist")
		}
	})
}

func TestRetrieveFile(t *testing.T) {
	tempDir := t.TempDir()
	node, _ := NewNode("test-node", tempDir)

	t.Run("retrieve existing file", func(t *testing.T) {
		originalData := []byte("Test data for retrieval")
		node.StoreFile("file-1", "version-1", originalData)

		retrievedData, err := node.RetrieveFile("file-1", "version-1")
		if err != nil {
			t.Errorf("RetrieveFile failed: %v", err)
		}
		if !bytes.Equal(retrievedData, originalData) {
			t.Error("retrieved data does not match original")
		}
	})

	t.Run("retrieve non-existing file", func(t *testing.T) {
		_, err := node.RetrieveFile("non-existing", "version-1")
		if err == nil {
			t.Error("expected error for non-existing file")
		}
	})

	t.Run("checksum verification", func(t *testing.T) {
		data := []byte("Data with checksum")
		node.StoreFile("file-2", "version-1", data)

		// Corrupt the data file
		filePath := filepath.Join(tempDir, "file-2", "version-1", "data")
		corruptData := []byte("Corrupted data")
		os.WriteFile(filePath, corruptData, 0644)

		// Should fail checksum verification
		_, err := node.RetrieveFile("file-2", "version-1")
		if err == nil {
			t.Error("expected checksum mismatch error")
		}
	})
}

func TestDeleteFile(t *testing.T) {
	tempDir := t.TempDir()
	node, _ := NewNode("test-node", tempDir)

	t.Run("delete existing file", func(t *testing.T) {
		data := []byte("File to delete")
		node.StoreFile("file-1", "version-1", data)

		err := node.DeleteFile("file-1", "version-1")
		if err != nil {
			t.Errorf("DeleteFile failed: %v", err)
		}

		if node.FileExists("file-1", "version-1") {
			t.Error("file still exists after deletion")
		}
	})

	t.Run("delete keeps other versions", func(t *testing.T) {
		node.StoreFile("file-2", "v1", []byte("Version 1"))
		node.StoreFile("file-2", "v2", []byte("Version 2"))

		err := node.DeleteFile("file-2", "v1")
		if err != nil {
			t.Errorf("DeleteFile failed: %v", err)
		}

		if node.FileExists("file-2", "v1") {
			t.Error("deleted version still exists")
		}
		if !node.FileExists("file-2", "v2") {
			t.Error("other version was deleted")
		}
	})

	t.Run("delete last version removes directory", func(t *testing.T) {
		node.StoreFile("file-3", "v1", []byte("Only version"))
		node.DeleteFile("file-3", "v1")

		filePath := filepath.Join(tempDir, "file-3")
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Error("file directory still exists after deleting last version")
		}
	})
}

func TestDeleteAllVersions(t *testing.T) {
	tempDir := t.TempDir()
	node, _ := NewNode("test-node", tempDir)

	t.Run("delete all versions", func(t *testing.T) {
		node.StoreFile("file-1", "v1", []byte("Version 1"))
		node.StoreFile("file-1", "v2", []byte("Version 2"))
		node.StoreFile("file-1", "v3", []byte("Version 3"))

		err := node.DeleteAllVersions("file-1")
		if err != nil {
			t.Errorf("DeleteAllVersions failed: %v", err)
		}

		if node.FileExists("file-1", "v1") || node.FileExists("file-1", "v2") || node.FileExists("file-1", "v3") {
			t.Error("versions still exist after DeleteAllVersions")
		}
	})
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()
	node, _ := NewNode("test-node", tempDir)

	t.Run("existing file returns true", func(t *testing.T) {
		node.StoreFile("file-1", "version-1", []byte("test"))
		if !node.FileExists("file-1", "version-1") {
			t.Error("FileExists returned false for existing file")
		}
	})

	t.Run("non-existing file returns false", func(t *testing.T) {
		if node.FileExists("non-existing", "version-1") {
			t.Error("FileExists returned true for non-existing file")
		}
	})
}

func TestGetStorageSize(t *testing.T) {
	tempDir := t.TempDir()
	node, _ := NewNode("test-node", tempDir)

	t.Run("empty storage", func(t *testing.T) {
		size, err := node.GetStorageSize()
		if err != nil {
			t.Errorf("GetStorageSize failed: %v", err)
		}
		if size != 0 {
			t.Errorf("empty storage size = %d, want 0", size)
		}
	})

	t.Run("with files", func(t *testing.T) {
		data1 := []byte("Test data 1")
		data2 := []byte("Test data 2 - longer")
		
		node.StoreFile("file-1", "v1", data1)
		node.StoreFile("file-2", "v1", data2)

		size, err := node.GetStorageSize()
		if err != nil {
			t.Errorf("GetStorageSize failed: %v", err)
		}
		if size == 0 {
			t.Error("storage size is 0 with files stored")
		}
	})
}

func TestReplicateFile(t *testing.T) {
	tempDir := t.TempDir()
	node, _ := NewNode("test-node", tempDir)

	t.Run("replicate from reader", func(t *testing.T) {
		data := []byte("Data to replicate")
		reader := bytes.NewReader(data)

		err := node.ReplicateFile("file-1", "version-1", reader)
		if err != nil {
			t.Errorf("ReplicateFile failed: %v", err)
		}

		// Verify file was stored correctly
		retrievedData, err := node.RetrieveFile("file-1", "version-1")
		if err != nil {
			t.Errorf("RetrieveFile after replication failed: %v", err)
		}
		if !bytes.Equal(retrievedData, data) {
			t.Error("replicated data does not match original")
		}
	})

	t.Run("replicate large file", func(t *testing.T) {
		largeData := make([]byte, 5*1024*1024) // 5MB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		reader := bytes.NewReader(largeData)

		err := node.ReplicateFile("file-2", "version-1", reader)
		if err != nil {
			t.Errorf("ReplicateFile failed for large file: %v", err)
		}
	})
}

func TestListFiles(t *testing.T) {
	tempDir := t.TempDir()
	node, _ := NewNode("test-node", tempDir)

	t.Run("empty storage", func(t *testing.T) {
		files, err := node.ListFiles()
		if err != nil {
			t.Errorf("ListFiles failed: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("got %d files, want 0", len(files))
		}
	})

	t.Run("with files", func(t *testing.T) {
		node.StoreFile("file-1", "v1", []byte("data1"))
		node.StoreFile("file-2", "v1", []byte("data2"))
		node.StoreFile("file-3", "v1", []byte("data3"))

		files, err := node.ListFiles()
		if err != nil {
			t.Errorf("ListFiles failed: %v", err)
		}
		if len(files) != 3 {
			t.Errorf("got %d files, want 3", len(files))
		}
	})
}

func TestConcurrentOperations(t *testing.T) {
	tempDir := t.TempDir()
	node, _ := NewNode("test-node", tempDir)

	t.Run("concurrent writes", func(t *testing.T) {
		done := make(chan bool)
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				data := []byte("concurrent data")
				versionID := fmt.Sprintf("version-%d", id)
				err := node.StoreFile("file-concurrent", versionID, data)
				if err != nil {
					t.Errorf("concurrent StoreFile failed: %v", err)
				}
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	t.Run("concurrent reads", func(t *testing.T) {
		data := []byte("read data")
		node.StoreFile("file-read", "v1", data)

		done := make(chan bool)
		numReads := 20

		for i := 0; i < numReads; i++ {
			go func() {
				_, err := node.RetrieveFile("file-read", "v1")
				if err != nil {
					t.Errorf("concurrent RetrieveFile failed: %v", err)
				}
				done <- true
			}()
		}

		for i := 0; i < numReads; i++ {
			<-done
		}
	})
}

func BenchmarkStoreFile(b *testing.B) {
	tempDir := b.TempDir()
	node, _ := NewNode("bench-node", tempDir)
	data := make([]byte, 1024*1024) // 1MB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node.StoreFile("file-bench", "v1", data)
	}
}

func BenchmarkRetrieveFile(b *testing.B) {
	tempDir := b.TempDir()
	node, _ := NewNode("bench-node", tempDir)
	data := make([]byte, 1024*1024) // 1MB
	node.StoreFile("file-bench", "v1", data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node.RetrieveFile("file-bench", "v1")
	}
}
