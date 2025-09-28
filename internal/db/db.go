package db

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type VectorDB struct {
	storage  *Storage
	index    Index
	wal      *WAL
	metadata *MetadataEngine
	mu       sync.RWMutex
	config   *Config
	closed   bool
}

type Config struct {
	DataDir       string `json:"data_dir"`
	WALEnabled    bool   `json:"wal_enabled"`
	IndexType     string `json:"index_type"`
	DimensionSize int    `json:"dimension_size"`
}

func NewVectorDB(config *Config) (*VectorDB, error) {
	if config.DataDir == "" {
		config.DataDir = "./data"
	}
	if config.IndexType == "" {
		config.IndexType = "hnsw"
	}

	storage, err := NewStorage(config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	var wal *WAL
	if config.WALEnabled {
		wal, err = NewWAL(config.DataDir + "/wal")
		if err != nil {
			storage.Close()
			return nil, fmt.Errorf("failed to initialize WAL: %w", err)
		}
	}

	metadataEngine := NewMetadataEngine()

	var index Index
	switch config.IndexType {
	case "hnsw":
		index = NewHNSWIndex(config.DimensionSize)
	case "brute":
		index = NewBruteForceIndex(config.DimensionSize)
	default:
		return nil, fmt.Errorf("unsupported index type: %s", config.IndexType)
	}

	db := &VectorDB{
		storage:  storage,
		index:    index,
		wal:      wal,
		metadata: metadataEngine,
		config:   config,
	}

	if err := db.bootstrap(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to bootstrap: %w", err)
	}

	log.Printf("VectorDB initialized with %s index", config.IndexType)
	return db, nil
}

func (db *VectorDB) bootstrap() error {
	log.Println("Bootstrapping database from disk...")

	vectors, err := db.storage.LoadAll()
	if err != nil {
		return fmt.Errorf("failed to load vectors from storage: %w", err)
	}

	for _, vector := range vectors {
		if err := db.index.Add(vector); err != nil {
			return fmt.Errorf("failed to add vector to index: %w", err)
		}
		db.metadata.AddVector(vector)
	}

	if db.wal != nil {
		if err := db.replayWAL(); err != nil {
			return fmt.Errorf("failed to replay WAL: %w", err)
		}
	}

	log.Printf("Bootstrapped %d vectors", len(vectors))
	return nil
}

func (db *VectorDB) replayWAL() error {
	entries, err := db.wal.ReadAll()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		switch entry.Operation {
		case "INSERT":
			if err := db.index.Add(entry.Vector); err != nil {
				log.Printf("WAL replay error for INSERT: %v", err)
			}
			db.metadata.AddVector(entry.Vector)
		case "UPDATE":
			if err := db.index.Update(entry.Vector); err != nil {
				log.Printf("WAL replay error for UPDATE: %v", err)
			}
			db.metadata.UpdateVector(entry.Vector)
		case "DELETE":
			if err := db.index.Delete(entry.Vector.ID); err != nil {
				log.Printf("WAL replay error for DELETE: %v", err)
			}
			db.metadata.DeleteVector(entry.Vector.ID)
		}
	}

	log.Printf("Replayed %d WAL entries", len(entries))
	return nil
}

func (db *VectorDB) Insert(ctx context.Context, vector *Vector) error {
	if db.closed {
		return fmt.Errorf("database is closed")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	vector.Timestamp = time.Now()

	if db.wal != nil {
		entry := &WALEntry{
			Operation: "INSERT",
			Vector:    vector,
			Timestamp: time.Now(),
		}
		if err := db.wal.Append(entry); err != nil {
			return fmt.Errorf("failed to write to WAL: %w", err)
		}
	}

	if err := db.index.Add(vector); err != nil {
		return fmt.Errorf("failed to add to index: %w", err)
	}

	db.metadata.AddVector(vector)

	if err := db.storage.Store(vector); err != nil {
		return fmt.Errorf("failed to store vector: %w", err)
	}

	return nil
}

func (db *VectorDB) Search(ctx context.Context, query *SearchQuery) ([]*SearchResult, error) {
	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	db.mu.RLock()
	defer db.mu.RUnlock()

	var candidateIDs []string
	if len(query.MetadataFilter) > 0 {
		candidateIDs = db.metadata.Filter(query.Namespace, query.MetadataFilter)
		if len(candidateIDs) == 0 {
			return []*SearchResult{}, nil
		}
	}

	results, err := db.index.Search(query.Vector, query.TopK, query.Namespace, candidateIDs)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return results, nil
}

func (db *VectorDB) Update(ctx context.Context, vector *Vector) error {
	if db.closed {
		return fmt.Errorf("database is closed")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	vector.Timestamp = time.Now()

	if db.wal != nil {
		entry := &WALEntry{
			Operation: "UPDATE",
			Vector:    vector,
			Timestamp: time.Now(),
		}
		if err := db.wal.Append(entry); err != nil {
			return fmt.Errorf("failed to write to WAL: %w", err)
		}
	}

	if err := db.index.Update(vector); err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}

	db.metadata.UpdateVector(vector)

	if err := db.storage.Store(vector); err != nil {
		return fmt.Errorf("failed to update storage: %w", err)
	}

	return nil
}

func (db *VectorDB) Delete(ctx context.Context, id, namespace string) error {
	if db.closed {
		return fmt.Errorf("database is closed")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	vector := &Vector{ID: id, Namespace: namespace}

	if db.wal != nil {
		entry := &WALEntry{
			Operation: "DELETE",
			Vector:    vector,
			Timestamp: time.Now(),
		}
		if err := db.wal.Append(entry); err != nil {
			return fmt.Errorf("failed to write to WAL: %w", err)
		}
	}

	if err := db.index.Delete(id); err != nil {
		return fmt.Errorf("failed to delete from index: %w", err)
	}

	db.metadata.DeleteVector(id)

	if err := db.storage.Delete(id); err != nil {
		return fmt.Errorf("failed to delete from storage: %w", err)
	}

	return nil
}

func (db *VectorDB) GetStats() *IndexStats {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.index.Stats()
}

func (db *VectorDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil
	}

	db.closed = true

	if db.wal != nil {
		if err := db.wal.Close(); err != nil {
			log.Printf("Error closing WAL: %v", err)
		}
	}

	if err := db.storage.Close(); err != nil {
		log.Printf("Error closing storage: %v", err)
	}

	log.Println("VectorDB closed successfully")
	return nil
}
