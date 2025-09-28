package vectordb

import (
	"context"
	"time"

	"rohitlohar45/vectordb/internal/db"
)

// VectorDB is the public interface for the vector database
type VectorDB struct {
	db *db.VectorDB
}

// Config holds database configuration
type Config struct {
	DataDir       string `json:"data_dir"`
	WALEnabled    bool   `json:"wal_enabled"`
	IndexType     string `json:"index_type"` // "hnsw" or "brute"
	DimensionSize int    `json:"dimension_size"`
}

// Vector represents a vector with metadata and ID
type Vector struct {
	ID        string                 `json:"id"`
	Data      []float32              `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
	Namespace string                 `json:"namespace"`
	Timestamp time.Time              `json:"timestamp"`
}

// SearchResult represents a search result with distance
type SearchResult struct {
	Vector   *Vector `json:"vector"`
	Distance float32 `json:"distance"`
}

// SearchQuery represents parameters for vector search
type SearchQuery struct {
	Vector         []float32              `json:"vector"`
	TopK           int                    `json:"top_k"`
	Namespace      string                 `json:"namespace"`
	MetadataFilter map[string]interface{} `json:"metadata_filter"`
}

// IndexStats represents statistics about the index
type IndexStats struct {
	VectorCount    int            `json:"vector_count"`
	NamespaceCount map[string]int `json:"namespace_count"`
	DimensionSize  int            `json:"dimension_size"`
	IndexType      string         `json:"index_type"`
}

// NewVectorDB creates a new vector database instance
func NewVectorDB(config *Config) (*VectorDB, error) {
	internalConfig := &db.Config{
		DataDir:       config.DataDir,
		WALEnabled:    config.WALEnabled,
		IndexType:     config.IndexType,
		DimensionSize: config.DimensionSize,
	}

	internalDB, err := db.NewVectorDB(internalConfig)
	if err != nil {
		return nil, err
	}

	return &VectorDB{db: internalDB}, nil
}

// Insert adds a new vector to the database
func (vdb *VectorDB) Insert(ctx context.Context, vector *Vector) error {
	internalVector := &db.Vector{
		ID:        vector.ID,
		Data:      vector.Data,
		Metadata:  vector.Metadata,
		Namespace: vector.Namespace,
		Timestamp: vector.Timestamp,
	}
	return vdb.db.Insert(ctx, internalVector)
}

// Search performs vector similarity search
func (vdb *VectorDB) Search(ctx context.Context, query *SearchQuery) ([]*SearchResult, error) {
	internalQuery := &db.SearchQuery{
		Vector:         query.Vector,
		TopK:           query.TopK,
		Namespace:      query.Namespace,
		MetadataFilter: query.MetadataFilter,
	}

	internalResults, err := vdb.db.Search(ctx, internalQuery)
	if err != nil {
		return nil, err
	}

	results := make([]*SearchResult, 0, len(internalResults))
	for _, result := range internalResults {
		if result.Distance == 1 {
			continue
		}

		results = append(results, &SearchResult{
			Vector: &Vector{
				ID:        result.Vector.ID,
				Data:      result.Vector.Data,
				Metadata:  result.Vector.Metadata,
				Namespace: result.Vector.Namespace,
				Timestamp: result.Vector.Timestamp,
			},
			Distance: result.Distance,
		})
	}

	return results, nil
}

// Update modifies an existing vector
func (vdb *VectorDB) Update(ctx context.Context, vector *Vector) error {
	internalVector := &db.Vector{
		ID:        vector.ID,
		Data:      vector.Data,
		Metadata:  vector.Metadata,
		Namespace: vector.Namespace,
		Timestamp: vector.Timestamp,
	}
	return vdb.db.Update(ctx, internalVector)
}

// Delete removes a vector from the database
func (vdb *VectorDB) Delete(ctx context.Context, id, namespace string) error {
	return vdb.db.Delete(ctx, id, namespace)
}

// GetStats returns database statistics
func (vdb *VectorDB) GetStats() *IndexStats {
	internalStats := vdb.db.GetStats()
	return &IndexStats{
		VectorCount:    internalStats.VectorCount,
		NamespaceCount: internalStats.NamespaceCount,
		DimensionSize:  internalStats.DimensionSize,
		IndexType:      internalStats.IndexType,
	}
}

// Close gracefully shuts down the database
func (vdb *VectorDB) Close() error {
	return vdb.db.Close()
}
