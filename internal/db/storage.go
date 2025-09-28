package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/dgraph-io/badger/v3"
)

type Storage struct {
	db   *badger.DB
	path string
	mu   sync.RWMutex
}

func NewStorage(dataDir string) (*Storage, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	opts := badger.DefaultOptions(filepath.Join(dataDir, "badger"))
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %w", err)
	}

	return &Storage{
		db:   db,
		path: dataDir,
	}, nil
}

func (s *Storage) Store(vector *Vector) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(vector)
	if err != nil {
		return fmt.Errorf("failed to marshal vector: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte(vector.Namespace + ":" + vector.ID)
		return txn.Set(key, data)
	})
}

func (s *Storage) Load(id, namespace string) (*Vector, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var vector *Vector
	key := []byte(namespace + ":" + id)

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &vector)
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load vector: %w", err)
	}

	return vector, nil
}

func (s *Storage) LoadAll() ([]*Vector, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var vectors []*Vector

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				var vector Vector
				if err := json.Unmarshal(val, &vector); err != nil {
					return err
				}
				vectors = append(vectors, &vector)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load all vectors: %w", err)
	}

	return vectors, nil
}

func (s *Storage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(txn *badger.Txn) error {
		// We need to find and delete all keys with this ID across namespaces
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		var keysToDelete [][]byte
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()

			// Extract ID from key (format: namespace:id)
			keyStr := string(key)
			if len(keyStr) > 0 {
				parts := []byte(keyStr)
				// Find the last colon to separate namespace:id
				colonIdx := -1
				for i := len(parts) - 1; i >= 0; i-- {
					if parts[i] == ':' {
						colonIdx = i
						break
					}
				}
				if colonIdx > 0 && string(parts[colonIdx+1:]) == id {
					keysToDelete = append(keysToDelete, append([]byte(nil), key...))
				}
			}
		}

		for _, key := range keysToDelete {
			if err := txn.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Storage) Close() error {
	return s.db.Close()
}
