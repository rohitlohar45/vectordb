package main

import (
	"context"
	"fmt"
	"log"

	"rohitlohar45/vectordb/internal/db"
)

func main() {
	fmt.Println("VectorDB Test Script")
	fmt.Println("===================")

	// Create configuration
	config := &db.Config{
		DataDir:       "./test_data",
		WALEnabled:    true,
		IndexType:     "hnsw",
		DimensionSize: 32, // Small dimension for quick testing
	}

	// Initialize VectorDB
	vectorDB, err := db.NewVectorDB(config)
	if err != nil {
		log.Fatalf("Failed to create VectorDB: %v", err)
	}
	defer vectorDB.Close()

	ctx := context.Background()

	// Test 1: Insert vectors
	fmt.Println("\n1. Inserting test vectors...")
	testVectors := []*db.Vector{
		{
			ID:        "test1",
			Data:      generateVector(32, 1.0),
			Metadata:  map[string]interface{}{"type": "document", "category": "tech"},
			Namespace: "test_ns",
		},
		{
			ID:        "test2",
			Data:      generateVector(32, 2.0),
			Metadata:  map[string]interface{}{"type": "image", "category": "art"},
			Namespace: "test_ns",
		},
		{
			ID:        "test3",
			Data:      generateVector(32, 1.5),
			Metadata:  map[string]interface{}{"type": "document", "category": "science"},
			Namespace: "test_ns",
		},
	}

	for _, v := range testVectors {
		if err := vectorDB.Insert(ctx, v); err != nil {
			log.Printf("Failed to insert %s: %v", v.ID, err)
		} else {
			fmt.Printf("✓ Inserted: %s\n", v.ID)
		}
	}

	// Test 2: Search without filter
	fmt.Println("\n2. Searching without metadata filter...")
	query := &db.SearchQuery{
		Vector:    generateVector(32, 1.2),
		TopK:      3,
		Namespace: "test_ns",
	}

	results, err := vectorDB.Search(ctx, query)
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Found %d results:\n", len(results))
		for i, r := range results {
			fmt.Printf("  %d. %s (distance: %.3f) - %v\n",
				i+1, r.Vector.ID, r.Distance, r.Vector.Metadata)
		}
	}

	// Test 3: Search with metadata filter
	fmt.Println("\n3. Searching with metadata filter (type=document)...")
	filteredQuery := &db.SearchQuery{
		Vector:    generateVector(32, 1.2),
		TopK:      5,
		Namespace: "test_ns",
		MetadataFilter: map[string]interface{}{
			"type": "document",
		},
	}

	filteredResults, err := vectorDB.Search(ctx, filteredQuery)
	if err != nil {
		log.Printf("Filtered search failed: %v", err)
	} else {
		fmt.Printf("Found %d filtered results:\n", len(filteredResults))
		for i, r := range filteredResults {
			fmt.Printf("  %d. %s (distance: %.3f) - %v\n",
				i+1, r.Vector.ID, r.Distance, r.Vector.Metadata)
		}
	}

	// Test 4: Update a vector
	fmt.Println("\n4. Updating a vector...")
	updatedVector := &db.Vector{
		ID:        "test1",
		Data:      generateVector(32, 1.1),
		Metadata:  map[string]interface{}{"type": "document", "category": "tech", "updated": true},
		Namespace: "test_ns",
	}

	if err := vectorDB.Update(ctx, updatedVector); err != nil {
		log.Printf("Update failed: %v", err)
	} else {
		fmt.Printf("✓ Updated: %s\n", updatedVector.ID)
	}

	// Test 5: Delete a vector
	fmt.Println("\n5. Deleting a vector...")
	if err := vectorDB.Delete(ctx, "test2", "test_ns"); err != nil {
		log.Printf("Delete failed: %v", err)
	} else {
		fmt.Printf("✓ Deleted: test2\n")
	}

	// Test 6: Final search
	fmt.Println("\n6. Final search after updates...")
	finalResults, err := vectorDB.Search(ctx, query)
	if err != nil {
		log.Printf("Final search failed: %v", err)
	} else {
		fmt.Printf("Found %d results after updates:\n", len(finalResults))
		for i, r := range finalResults {
			fmt.Printf("  %d. %s (distance: %.3f) - %v\n",
				i+1, r.Vector.ID, r.Distance, r.Vector.Metadata)
		}
	}

	// Test 7: Get statistics
	fmt.Println("\n7. Database statistics:")
	stats := vectorDB.GetStats()
	fmt.Printf("  Total vectors: %d\n", stats.VectorCount)
	fmt.Printf("  Namespaces: %v\n", stats.NamespaceCount)
	fmt.Printf("  Dimension size: %d\n", stats.DimensionSize)
	fmt.Printf("  Index type: %s\n", stats.IndexType)

	fmt.Println("\n✓ All tests completed!")
}

// generateVector creates a vector with a specific pattern for testing
func generateVector(dim int, multiplier float32) []float32 {
	vector := make([]float32, dim)
	for i := range vector {
		vector[i] = float32(i) * multiplier / float32(dim)
	}
	return vector
}
