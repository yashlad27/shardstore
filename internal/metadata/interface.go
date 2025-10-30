package metadata

import "context"

// Store defines the interface for metadata storage operations
type Store interface {
	SaveMetadata(ctx context.Context, metadata *FileMetadata) error
	GetMetadata(ctx context.Context, fileID string) (*FileMetadata, error)
	DeleteMetadata(ctx context.Context, fileID string) error
	ListMetadata(ctx context.Context, page, pageSize int32) ([]*FileMetadata, int64, error)
	AddVersion(ctx context.Context, fileID string, version Version) error
	Close(ctx context.Context) error
}
