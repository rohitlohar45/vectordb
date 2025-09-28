package db

import (
	"fmt"
	"math"
	"sort"
	"sync"
)

type Index interface {
	Add(vector *Vector) error
	Search(query []float32, topK int, namespace string, candidateIDs []string) ([]*SearchResult, error)
	Update(vector *Vector) error
	Delete(id string) error
	Stats() *IndexStats
}

type BruteForceIndex struct {
	vectors   map[string]*Vector
	dimension int
	mu        sync.RWMutex
}

func NewBruteForceIndex(dimension int) *BruteForceIndex {
	return &BruteForceIndex{
		vectors:   make(map[string]*Vector),
		dimension: dimension,
	}
}

func (idx *BruteForceIndex) Add(vector *Vector) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if len(vector.Data) != idx.dimension {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d", idx.dimension, len(vector.Data))
	}

	idx.vectors[vector.ID] = vector
	return nil
}

func (idx *BruteForceIndex) Search(query []float32, topK int, namespace string, candidateIDs []string) ([]*SearchResult, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(query) != idx.dimension {
		return nil, fmt.Errorf("query dimension mismatch: expected %d, got %d", idx.dimension, len(query))
	}

	var candidates []*Vector
	candidateSet := make(map[string]bool)

	if len(candidateIDs) > 0 {
		for _, id := range candidateIDs {
			candidateSet[id] = true
		}
	}

	for _, vector := range idx.vectors {
		if namespace != "" && vector.Namespace != namespace {
			continue
		}
		if len(candidateIDs) > 0 && !candidateSet[vector.ID] {
			continue
		}
		candidates = append(candidates, vector)
	}

	var results []*SearchResult
	for _, vector := range candidates {
		distance := cosineSimilarity(query, vector.Data)
		results = append(results, &SearchResult{
			Vector:   vector,
			Distance: distance,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK], nil
}

func (idx *BruteForceIndex) Update(vector *Vector) error {
	return idx.Add(vector)
}

func (idx *BruteForceIndex) Delete(id string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.vectors, id)
	return nil
}

func (idx *BruteForceIndex) Stats() *IndexStats {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	namespaceCounts := make(map[string]int)
	for _, vector := range idx.vectors {
		namespaceCounts[vector.Namespace]++
	}

	return &IndexStats{
		VectorCount:    len(idx.vectors),
		NamespaceCount: namespaceCounts,
		DimensionSize:  idx.dimension,
		IndexType:      "brute",
	}
}

type HNSWIndex struct {
	vectors   map[string]*Vector
	dimension int
	mu        sync.RWMutex
	graph     map[string][]string
}

func NewHNSWIndex(dimension int) *HNSWIndex {
	return &HNSWIndex{
		vectors:   make(map[string]*Vector),
		dimension: dimension,
		graph:     make(map[string][]string),
	}
}

func (idx *HNSWIndex) Add(vector *Vector) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if len(vector.Data) != idx.dimension {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d", idx.dimension, len(vector.Data))
	}

	idx.vectors[vector.ID] = vector

	var neighbors []string
	var distances []float32

	for id, v := range idx.vectors {
		if id == vector.ID {
			continue
		}
		if v.Namespace != vector.Namespace {
			continue
		}

		dist := cosineSimilarity(vector.Data, v.Data)
		neighbors = append(neighbors, id)
		distances = append(distances, dist)
	}

	if len(neighbors) > 0 {
		type neighborPair struct {
			id   string
			dist float32
		}

		pairs := make([]neighborPair, len(neighbors))
		for i := range neighbors {
			pairs[i] = neighborPair{neighbors[i], distances[i]}
		}

		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].dist < pairs[j].dist
		})

		maxConnections := 16
		if len(pairs) < maxConnections {
			maxConnections = len(pairs)
		}

		connections := make([]string, maxConnections)
		for i := 0; i < maxConnections; i++ {
			connections[i] = pairs[i].id
		}

		idx.graph[vector.ID] = connections
	}

	return nil
}

func (idx *HNSWIndex) Search(query []float32, topK int, namespace string, candidateIDs []string) ([]*SearchResult, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(query) != idx.dimension {
		return nil, fmt.Errorf("query dimension mismatch: expected %d, got %d", idx.dimension, len(query))
	}

	var candidates []*Vector
	candidateSet := make(map[string]bool)

	if len(candidateIDs) > 0 {
		for _, id := range candidateIDs {
			candidateSet[id] = true
		}
	}

	for _, vector := range idx.vectors {
		if namespace != "" && vector.Namespace != namespace {
			continue
		}
		if len(candidateIDs) > 0 && !candidateSet[vector.ID] {
			continue
		}
		candidates = append(candidates, vector)
	}

	var results []*SearchResult
	for _, vector := range candidates {
		distance := cosineSimilarity(query, vector.Data)
		results = append(results, &SearchResult{
			Vector:   vector,
			Distance: distance,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK], nil
}

func (idx *HNSWIndex) Update(vector *Vector) error {
	idx.Delete(vector.ID)
	return idx.Add(vector)
}

func (idx *HNSWIndex) Delete(id string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.vectors, id)
	delete(idx.graph, id)

	for nodeID, connections := range idx.graph {
		newConnections := make([]string, 0)
		for _, connID := range connections {
			if connID != id {
				newConnections = append(newConnections, connID)
			}
		}
		idx.graph[nodeID] = newConnections
	}

	return nil
}

func (idx *HNSWIndex) Stats() *IndexStats {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	namespaceCounts := make(map[string]int)
	for _, vector := range idx.vectors {
		namespaceCounts[vector.Namespace]++
	}

	return &IndexStats{
		VectorCount:    len(idx.vectors),
		NamespaceCount: namespaceCounts,
		DimensionSize:  idx.dimension,
		IndexType:      "hnsw",
	}
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return math.MaxFloat32
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return math.MaxFloat32
	}

	similarity := dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
	return 1 - similarity
}
