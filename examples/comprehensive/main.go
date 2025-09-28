package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"rohitlohar45/vectordb/internal/db"
)

func main() {
	// Create a configuration for the VectorDB
	config := &db.Config{
		DataDir:       "./vectordb_data",
		WALEnabled:    true,
		IndexType:     "hnsw", // or "brute"
		DimensionSize: 128,
	}

	// Initialize the VectorDB
	vectorDB, err := db.NewVectorDB(config)
	if err != nil {
		log.Fatalf("Failed to create VectorDB: %v", err)
	}
	defer vectorDB.Close()

	fmt.Println("VectorDB initialized successfully!")
	fmt.Println(strings.Repeat("=", 50))

	// Example 1: Insert some vectors with metadata
	fmt.Println("Example 1: Inserting vectors with metadata")
	insertExampleVectors(vectorDB)

	// Example 2: Search for similar vectors
	fmt.Println("\nExample 2: Searching for similar vectors")
	searchExample(vectorDB)

	// Example 3: Search with metadata filtering
	fmt.Println("\nExample 3: Searching with metadata filtering")
	searchWithMetadataFilter(vectorDB)

	// Example 4: Update a vector
	fmt.Println("\nExample 4: Updating a vector")
	updateExample(vectorDB)

	// Example 5: Delete a vector
	fmt.Println("\nExample 5: Deleting a vector")
	deleteExample(vectorDB)

	// Example 6: Get database statistics
	fmt.Println("\nExample 6: Database statistics")
	stats := vectorDB.GetStats()
	fmt.Printf("Total vectors: %d\n", stats.VectorCount)
	fmt.Printf("Namespaces: %v\n", stats.NamespaceCount)
	fmt.Printf("Dimension size: %d\n", stats.DimensionSize)
	fmt.Printf("Index type: %s\n", stats.IndexType)
}

// insertExampleVectors demonstrates how to insert vectors with metadata
func insertExampleVectors(vectorDB *db.VectorDB) {
	ctx := context.Background()

	// Create some sample vectors with metadata
	vectors := []*db.Vector{
		{
			ID:        "doc1",
			Data:      generateRandomVector(128),
			Metadata:  map[string]interface{}{"category": "technology", "author": "john", "year": 2023},
			Namespace: "documents",
		},
		{
			ID:        "doc2",
			Data:      generateRandomVector(128),
			Metadata:  map[string]interface{}{"category": "science", "author": "jane", "year": 2023},
			Namespace: "documents",
		},
		{
			ID:        "doc3",
			Data:      generateRandomVector(128),
			Metadata:  map[string]interface{}{"category": "technology", "author": "bob", "year": 2022},
			Namespace: "documents",
		},
		{
			ID:        "img1",
			Data:      generateRandomVector(128),
			Metadata:  map[string]interface{}{"type": "photo", "format": "jpg", "size": "large"},
			Namespace: "images",
		},
		{
			ID:        "img2",
			Data:      generateRandomVector(128),
			Metadata:  map[string]interface{}{"type": "illustration", "format": "png", "size": "medium"},
			Namespace: "images",
		},
	}

	// Insert each vector
	for _, vector := range vectors {
		err := vectorDB.Insert(ctx, vector)
		if err != nil {
			log.Printf("Failed to insert vector %s: %v", vector.ID, err)
		} else {
			fmt.Printf("✓ Inserted vector: %s (namespace: %s)\n", vector.ID, vector.Namespace)
		}
	}
}

// searchExample demonstrates basic vector similarity search
func searchExample(vectorDB *db.VectorDB) {
	ctx := context.Background()

	// Create a query vector
	queryVector := generateRandomVector(128)

	// Create a search query
	searchQuery := &db.SearchQuery{
		Vector:    queryVector,
		TopK:      3,
		Namespace: "documents",
	}

	// Perform the search
	results, err := vectorDB.Search(ctx, searchQuery)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	fmt.Printf("Found %d similar vectors in 'documents' namespace:\n", len(results))
	for i, result := range results {
		fmt.Printf("  %d. ID: %s, Distance: %.4f\n", i+1, result.Vector.ID, result.Distance)
		fmt.Printf("     Metadata: %v\n", result.Vector.Metadata)
	}
}

// searchWithMetadataFilter demonstrates search with metadata filtering
func searchWithMetadataFilter(vectorDB *db.VectorDB) {
	ctx := context.Background()

	// Create a query vector
	queryVector := generateRandomVector(128)

	// Create a search query with metadata filter
	searchQuery := &db.SearchQuery{
		Vector:    queryVector,
		TopK:      5,
		Namespace: "documents",
		MetadataFilter: map[string]interface{}{
			"category": "technology", // Only search in technology category
		},
	}

	// Perform the search
	results, err := vectorDB.Search(ctx, searchQuery)
	if err != nil {
		log.Printf("Search with filter failed: %v", err)
		return
	}

	fmt.Printf("Found %d technology documents:\n", len(results))
	for i, result := range results {
		fmt.Printf("  %d. ID: %s, Distance: %.4f\n", i+1, result.Vector.ID, result.Distance)
		fmt.Printf("     Author: %s, Year: %v\n", result.Vector.Metadata["author"], result.Vector.Metadata["year"])
	}
}

// updateExample demonstrates how to update a vector
func updateExample(vectorDB *db.VectorDB) {
	ctx := context.Background()

	// Create an updated vector
	updatedVector := &db.Vector{
		ID:        "doc1",
		Data:      generateRandomVector(128), // New vector data
		Metadata:  map[string]interface{}{"category": "technology", "author": "john", "year": 2024, "updated": true},
		Namespace: "documents",
	}

	// Update the vector
	err := vectorDB.Update(ctx, updatedVector)
	if err != nil {
		log.Printf("Failed to update vector: %v", err)
	} else {
		fmt.Printf("✓ Updated vector: %s\n", updatedVector.ID)
		fmt.Printf("  New metadata: %v\n", updatedVector.Metadata)
	}
}

// deleteExample demonstrates how to delete a vector
func deleteExample(vectorDB *db.VectorDB) {
	ctx := context.Background()

	// Delete a vector
	err := vectorDB.Delete(ctx, "img2", "images")
	if err != nil {
		log.Printf("Failed to delete vector: %v", err)
	} else {
		fmt.Printf("✓ Deleted vector: img2 from images namespace\n")
	}
}

// generateRandomVector creates a random vector of specified dimension
func generateRandomVector(dimension int) []float32 {
	vector := make([]float32, dimension)
	for i := range vector {
		vector[i] = rand.Float32()*2 - 1 // Random value between -1 and 1
	}
	return vector
}

// Advanced usage examples
func advancedExamples(vectorDB *db.VectorDB) {
	ctx := context.Background()

	fmt.Println("\nAdvanced Examples:")
	fmt.Println(strings.Repeat("=", 30))

	// Example: Complex metadata filtering
	fmt.Println("Complex metadata filtering:")
	searchQuery := &db.SearchQuery{
		Vector:    generateRandomVector(128),
		TopK:      10,
		Namespace: "documents",
		MetadataFilter: map[string]interface{}{
			"year": map[string]interface{}{
				"$gte": 2023, // Greater than or equal to 2023
			},
			"category": "technology",
		},
	}

	results, err := vectorDB.Search(ctx, searchQuery)
	if err != nil {
		log.Printf("Complex search failed: %v", err)
	} else {
		fmt.Printf("Found %d results with complex filter\n", len(results))
	}

	// Example: Batch operations
	fmt.Println("\nBatch operations:")
	vectors := make([]*db.Vector, 10)
	for i := 0; i < 10; i++ {
		vectors[i] = &db.Vector{
			ID:        fmt.Sprintf("batch_%d", i),
			Data:      generateRandomVector(128),
			Metadata:  map[string]interface{}{"batch": true, "index": i},
			Namespace: "batch_test",
		}
	}

	// Insert batch
	for _, vector := range vectors {
		if err := vectorDB.Insert(ctx, vector); err != nil {
			log.Printf("Batch insert failed for %s: %v", vector.ID, err)
		}
	}
	fmt.Printf("✓ Inserted %d vectors in batch\n", len(vectors))
}
