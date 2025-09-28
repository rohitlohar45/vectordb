package db

import (
	"time"
)

type Vector struct {
	ID        string                 `json:"id"`
	Data      []float32              `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
	Namespace string                 `json:"namespace"`
	Timestamp time.Time              `json:"timestamp"`
}

type SearchResult struct {
	Vector   *Vector `json:"vector"`
	Distance float32 `json:"distance"`
}

type SearchQuery struct {
	Vector         []float32              `json:"vector"`
	TopK           int                    `json:"top_k"`
	Namespace      string                 `json:"namespace"`
	MetadataFilter map[string]interface{} `json:"metadata_filter"`
}

type WALEntry struct {
	ID        uint64    `json:"id"`
	Operation string    `json:"operation"`
	Vector    *Vector   `json:"vector"`
	Timestamp time.Time `json:"timestamp"`
}

type IndexStats struct {
	VectorCount    int            `json:"vector_count"`
	NamespaceCount map[string]int `json:"namespace_count"`
	DimensionSize  int            `json:"dimension_size"`
	IndexType      string         `json:"index_type"`
}
