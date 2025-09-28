package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"rohitlohar45/vectordb/internal/db"
)

func main() {
	config := &db.Config{
		DataDir:       "./data",
		WALEnabled:    true,
		IndexType:     "hnsw",
		DimensionSize: 64,
	}

	vectorDB, err := db.NewVectorDB(config)
	if err != nil {
		log.Fatal(err)
	}
	defer vectorDB.Close()

	ctx := context.Background()

	vectors := []*db.Vector{
		{
			ID:        "vec1",
			Data:      randomVector(64),
			Metadata:  map[string]interface{}{"type": "text", "lang": "en"},
			Namespace: "test",
		},
		{
			ID:        "vec2",
			Data:      randomVector(64),
			Metadata:  map[string]interface{}{"type": "image", "format": "jpg"},
			Namespace: "test",
		},
		{
			ID:        "vec3",
			Data:      randomVector(64),
			Metadata:  map[string]interface{}{"type": "text", "lang": "es"},
			Namespace: "test",
		},
	}

	for _, v := range vectors {
		if err := vectorDB.Insert(ctx, v); err != nil {
			log.Printf("Insert error: %v", err)
		} else {
			fmt.Printf("Inserted: %s\n", v.ID)
		}
	}

	query := &db.SearchQuery{
		Vector:    randomVector(64),
		TopK:      2,
		Namespace: "test",
	}

	results, err := vectorDB.Search(ctx, query)
	if err != nil {
		log.Printf("Search error: %v", err)
	} else {
		fmt.Printf("\nSearch results:\n")
		for i, r := range results {
			fmt.Printf("%d. %s (distance: %.3f)\n", i+1, r.Vector.ID, r.Distance)
		}
	}

	filteredQuery := &db.SearchQuery{
		Vector:    randomVector(64),
		TopK:      5,
		Namespace: "test",
		MetadataFilter: map[string]interface{}{
			"type": "text",
		},
	}

	filteredResults, err := vectorDB.Search(ctx, filteredQuery)
	if err != nil {
		log.Printf("Filtered search error: %v", err)
	} else {
		fmt.Printf("\nFiltered search results (type=text):\n")
		for i, r := range filteredResults {
			fmt.Printf("%d. %s (distance: %.3f)\n", i+1, r.Vector.ID, r.Distance)
		}
	}

	stats := vectorDB.GetStats()
	fmt.Printf("\nDatabase stats:\n")
	fmt.Printf("Vectors: %d\n", stats.VectorCount)
	fmt.Printf("Namespaces: %v\n", stats.NamespaceCount)
}

func randomVector(dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = rand.Float32()*2 - 1
	}
	return v
}
