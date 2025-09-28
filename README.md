# VectorDB

A simple vector database implementation in Go with support for similarity search, metadata filtering, and persistence.

## Features

- **Vector Similarity Search**: Find similar vectors using HNSW or brute force indexing
- **Metadata Filtering**: Filter search results based on metadata conditions
- **Namespace Support**: Organize vectors into different namespaces
- **Persistence**: Store vectors on disk with WAL (Write-Ahead Log) support
- **Thread-Safe**: Concurrent access support with proper locking

## Quick Start

### 1. Run the comprehensive example:

```bash
go run examples/comprehensive/main.go
```

### 2. Run the test script:

```bash
go run examples/test/test_vectordb.go
```

### 3. Run the simple example:

```bash
go run examples/simple/example.go
```

## Usage

### Basic Setup

```go
import "./internal/db"

// Create configuration
config := &db.Config{
    DataDir:       "./data",
    WALEnabled:    true,
    IndexType:     "hnsw", // or "brute"
    DimensionSize: 128,
}

// Initialize VectorDB
vectorDB, err := db.NewVectorDB(config)
if err != nil {
    log.Fatal(err)
}
defer vectorDB.Close()
```

### Inserting Vectors

```go
vector := &db.Vector{
    ID:        "unique_id",
    Data:      []float32{0.1, 0.2, 0.3, ...}, // Your vector data
    Metadata:  map[string]interface{}{
        "category": "technology",
        "author":   "john",
        "year":     2023,
    },
    Namespace: "documents",
}

err := vectorDB.Insert(ctx, vector)
```

### Searching Vectors

```go
// Basic similarity search
query := &db.SearchQuery{
    Vector:    []float32{0.1, 0.2, 0.3, ...}, // Query vector
    TopK:      10,                             // Number of results
    Namespace: "documents",
}

results, err := vectorDB.Search(ctx, query)
```

### Metadata Filtering

```go
// Search with metadata filter
query := &db.SearchQuery{
    Vector:    []float32{0.1, 0.2, 0.3, ...},
    TopK:      10,
    Namespace: "documents",
    MetadataFilter: map[string]interface{}{
        "category": "technology",
        "year":     2023,
    },
}

results, err := vectorDB.Search(ctx, query)
```

### Advanced Metadata Queries

```go
// Complex metadata filtering with operators
query := &db.SearchQuery{
    Vector:    []float32{0.1, 0.2, 0.3, ...},
    TopK:      10,
    Namespace: "documents",
    MetadataFilter: map[string]interface{}{
        "year": map[string]interface{}{
            "$gte": 2023, // Greater than or equal
        },
        "category": "technology",
    },
}
```

### Updating Vectors

```go
updatedVector := &db.Vector{
    ID:        "existing_id",
    Data:      []float32{0.4, 0.5, 0.6, ...}, // New vector data
    Metadata:  map[string]interface{}{
        "category": "updated_category",
        "modified": true,
    },
    Namespace: "documents",
}

err := vectorDB.Update(ctx, updatedVector)
```

### Deleting Vectors

```go
err := vectorDB.Delete(ctx, "vector_id", "namespace")
```

### Getting Statistics

```go
stats := vectorDB.GetStats()
fmt.Printf("Total vectors: %d\n", stats.VectorCount)
fmt.Printf("Namespaces: %v\n", stats.NamespaceCount)
```

## Configuration Options

- **DataDir**: Directory to store data files
- **WALEnabled**: Enable Write-Ahead Logging for durability
- **IndexType**: "hnsw" for approximate search or "brute" for exact search
- **DimensionSize**: Size of vector dimensions

## Metadata Query Operators

- `$eq`: Equality (default)
- `$ne`: Not equal
- `$exists`: Field exists (value should be boolean)
- `$nexists`: Field does not exist

## Examples

See `examples/comprehensive/main.go` for comprehensive examples including:
- Basic vector operations
- Metadata filtering
- Batch operations
- Complex queries
- Error handling

See `examples/test/test_vectordb.go` for a focused test script that demonstrates:
- Insert, search, update, delete operations
- Metadata filtering
- Database statistics
- Step-by-step testing workflow

See `examples/simple/example.go` for a quick start example.

## File Structure

```
vectordb/
├── examples/           # Example applications
│   ├── comprehensive/  # Comprehensive examples
│   │   └── main.go
│   ├── test/          # Test script
│   │   └── test_vectordb.go
│   └── simple/        # Quick start example
│       └── example.go
├── README.md          # This file
├── go.mod             # Go module file
├── go.sum             # Go dependencies
└── internal/db/       # Core database implementation
    ├── db.go          # Main database orchestrator
    ├── index.go       # Vector indexing (HNSW, Brute Force)
    ├── metadata.go    # Metadata filtering engine
    ├── storage.go     # Persistent storage
    ├── types.go       # Data structures
    └── wal.go         # Write-Ahead Log
```

## Building and Running

```bash
# Initialize Go module (if not already done)
go mod init vectordb

# Run examples
go run examples/comprehensive/main.go    # Comprehensive examples
go run examples/test/test_vectordb.go    # Test script
go run examples/simple/example.go        # Simple example

# Build executable
go build -o vectordb examples/comprehensive/main.go
./vectordb
```
