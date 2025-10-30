package metadata

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FileMetadata represents file metadata stored in MongoDB
type FileMetadata struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	FileID      string             `bson:"file_id"`
	Filename    string             `bson:"filename"`
	Size        int64              `bson:"size"`
	ContentType string             `bson:"content_type"`
	Versions    []Version          `bson:"versions"`
	Replicas    []string           `bson:"replicas"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

// Version represents a file version
type Version struct {
	VersionID   string    `bson:"version_id"`
	Size        int64     `bson:"size"`
	Nodes       []string  `bson:"nodes"`
	CreatedAt   time.Time `bson:"created_at"`
}

// MetadataStore handles MongoDB operations for file metadata
type MetadataStore struct {
	client     *mongo.Client
	collection *mongo.Collection
}

// NewMetadataStore creates a new metadata store
func NewMetadataStore(mongoURI, database string) (*MetadataStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	collection := client.Database(database).Collection("files")

	// Create indexes
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "file_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return nil, err
	}

	return &MetadataStore{
		client:     client,
		collection: collection,
	}, nil
}

// SaveMetadata saves or updates file metadata
func (ms *MetadataStore) SaveMetadata(ctx context.Context, metadata *FileMetadata) error {
	metadata.UpdatedAt = time.Now()
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now()
	}

	filter := bson.M{"file_id": metadata.FileID}
	update := bson.M{"$set": metadata}
	opts := options.Update().SetUpsert(true)

	_, err := ms.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetMetadata retrieves file metadata by file ID
func (ms *MetadataStore) GetMetadata(ctx context.Context, fileID string) (*FileMetadata, error) {
	var metadata FileMetadata
	filter := bson.M{"file_id": fileID}
	
	err := ms.collection.FindOne(ctx, filter).Decode(&metadata)
	if err != nil {
		return nil, err
	}
	
	return &metadata, nil
}

// DeleteMetadata deletes file metadata
func (ms *MetadataStore) DeleteMetadata(ctx context.Context, fileID string) error {
	filter := bson.M{"file_id": fileID}
	_, err := ms.collection.DeleteOne(ctx, filter)
	return err
}

// ListMetadata lists all file metadata with pagination
func (ms *MetadataStore) ListMetadata(ctx context.Context, page, pageSize int32) ([]*FileMetadata, int64, error) {
	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	opts := options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.D{{Key: "created_at", Value: -1}})
	
	cursor, err := ms.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var files []*FileMetadata
	if err := cursor.All(ctx, &files); err != nil {
		return nil, 0, err
	}

	total, err := ms.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

// AddVersion adds a new version to file metadata
func (ms *MetadataStore) AddVersion(ctx context.Context, fileID string, version Version) error {
	filter := bson.M{"file_id": fileID}
	update := bson.M{
		"$push": bson.M{"versions": version},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	_, err := ms.collection.UpdateOne(ctx, filter, update)
	return err
}

// Close closes the MongoDB connection
func (ms *MetadataStore) Close(ctx context.Context) error {
	return ms.client.Disconnect(ctx)
}
