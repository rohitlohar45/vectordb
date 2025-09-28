package db

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// MetadataEngine handles metadata filtering and indexing
type MetadataEngine struct {
	metadataIndex  map[string]map[string]interface{} // vectorID -> metadata
	fieldIndex     map[string]map[interface{}][]string
	namespaceIndex map[string][]string // namespace -> vectorIDs
	mu             sync.RWMutex
}

// NewMetadataEngine creates a new MetadataEngine
func NewMetadataEngine() *MetadataEngine {
	return &MetadataEngine{
		metadataIndex:  make(map[string]map[string]interface{}),
		fieldIndex:     make(map[string]map[interface{}][]string),
		namespaceIndex: make(map[string][]string),
	}
}

// Set sets the metadata for a given vector ID.
// It also updates the field index for efficient querying.
func (me *MetadataEngine) Set(vectorID string, metadata map[string]interface{}) {
	me.mu.Lock()
	defer me.mu.Unlock()

	// Delete old metadata from index
	if oldMetadata, ok := me.metadataIndex[vectorID]; ok {
		for field, value := range oldMetadata {
			me.removeFromFieldIndex(field, value, vectorID)
		}
	}

	me.metadataIndex[vectorID] = metadata

	// Add new metadata to index
	for field, value := range metadata {
		me.addToFieldIndex(field, value, vectorID)
	}
}

func (me *MetadataEngine) addToFieldIndex(field string, value interface{}, vectorID string) {
	if me.fieldIndex[field] == nil {
		me.fieldIndex[field] = make(map[interface{}][]string)
	}

	// Convert complex types to strings for map keys
	key := me.convertToHashableKey(value)
	me.fieldIndex[field][key] = append(me.fieldIndex[field][key], vectorID)
}

func (me *MetadataEngine) removeFromFieldIndex(field string, value interface{}, vectorID string) {
	if me.fieldIndex[field] != nil {
		key := me.convertToHashableKey(value)
		if ids, ok := me.fieldIndex[field][key]; ok {
			for i, id := range ids {
				if id == vectorID {
					me.fieldIndex[field][key] = append(ids[:i], ids[i+1:]...)
					// If the list for this value becomes empty, delete the value entry
					if len(me.fieldIndex[field][key]) == 0 {
						delete(me.fieldIndex[field], key)
					}
					break
				}
			}
		}
		// If the field entry becomes empty, delete the field entry
		if len(me.fieldIndex[field]) == 0 {
			delete(me.fieldIndex, field)
		}
	}
}

// Get retrieves the metadata for a given vector ID.
func (me *MetadataEngine) Get(vectorID string) (map[string]interface{}, bool) {
	me.mu.RLock()
	defer me.mu.RUnlock()

	metadata, ok := me.metadataIndex[vectorID]
	return metadata, ok
}

// Delete removes the metadata for a given vector ID.
func (me *MetadataEngine) Delete(vectorID string) {
	me.mu.Lock()
	defer me.mu.Unlock()

	if oldMetadata, ok := me.metadataIndex[vectorID]; ok {
		for field, value := range oldMetadata {
			me.removeFromFieldIndex(field, value, vectorID)
		}
		delete(me.metadataIndex, vectorID)
	}
}

// Query filters vector IDs based on metadata conditions.
// It supports equality, inequality, and existence checks.
func (me *MetadataEngine) Query(filter map[string]interface{}) []string {
	me.mu.RLock()
	defer me.mu.RUnlock()

	if len(filter) == 0 {
		return me.getAllVectorIDs()
	}

	resultSet := make(map[string]struct{})
	firstField := true

	for field, queryValue := range filter {
		currentMatches := make(map[string]struct{})

		if reflect.TypeOf(queryValue).Kind() == reflect.Map {
			op, val, err := parseQueryCondition(queryValue)
			if err != nil {
				fmt.Printf("Error parsing query condition for field %s: %v\n", field, err)
				return []string{} // Return empty on error
			}
			me.applyCondition(field, op, val, currentMatches)
		} else {
			// Default to equality if no operator is specified
			me.applyCondition(field, "eq", queryValue, currentMatches)
		}

		if firstField {
			resultSet = currentMatches
			firstField = false
		} else {
			resultSet = me.intersect(resultSet, currentMatches)
		}

		if len(resultSet) == 0 {
			return []string{} // Early exit if no matches
		}
	}

	return me.mapToSlice(resultSet)
}

func (me *MetadataEngine) getAllVectorIDs() []string {
	ids := make([]string, 0, len(me.metadataIndex))
	for id := range me.metadataIndex {
		ids = append(ids, id)
	}
	return ids
}

func (me *MetadataEngine) intersect(s1, s2 map[string]struct{}) map[string]struct{} {
	intersection := make(map[string]struct{})
	for id := range s1 {
		if _, ok := s2[id]; ok {
			intersection[id] = struct{}{}
		}
	}
	return intersection
}

func (me *MetadataEngine) mapToSlice(m map[string]struct{}) []string {
	slice := make([]string, 0, len(m))
	for k := range m {
		slice = append(slice, k)
	}
	return slice
}

// parseQueryCondition parses a query condition map into an operator and value.
// Expected format: {"$op": value}
func parseQueryCondition(queryValue interface{}) (string, interface{}, error) {
	condMap, ok := queryValue.(map[string]interface{})
	if !ok || len(condMap) != 1 {
		return "", nil, fmt.Errorf("invalid query condition format: %v", queryValue)
	}

	for op, val := range condMap {
		if !strings.HasPrefix(op, "$") {
			return "", nil, fmt.Errorf("invalid operator format, must start with $: %s", op)
		}
		return strings.TrimPrefix(op, "$"), val, nil
	}
	return "", nil, fmt.Errorf("no operator found in query condition")
}

func (me *MetadataEngine) applyCondition(field, op string, value interface{}, matches map[string]struct{}) {
	switch op {
	case "eq": // Equality
		if fieldValues, ok := me.fieldIndex[field]; ok {
			key := me.convertToHashableKey(value)
			if vectorIDs, ok := fieldValues[key]; ok {
				for _, id := range vectorIDs {
					matches[id] = struct{}{}
				}
			}
		}
	case "ne": // Inequality
		// Get all vector IDs that have this field
		allFieldVectorIDs := make(map[string]struct{})
		if fieldValues, ok := me.fieldIndex[field]; ok {
			for _, ids := range fieldValues {
				for _, id := range ids {
					allFieldVectorIDs[id] = struct{}{}
				}
			}
		}

		// Remove vector IDs that have the equal value
		if fieldValues, ok := me.fieldIndex[field]; ok {
			key := me.convertToHashableKey(value)
			if vectorIDsToExclude, ok := fieldValues[key]; ok {
				for _, id := range vectorIDsToExclude {
					delete(allFieldVectorIDs, id)
				}
			}
		}
		for id := range allFieldVectorIDs {
			matches[id] = struct{}{}
		}

	case "exists": // Field exists
		if bVal, ok := value.(bool); ok && bVal {
			if fieldValues, ok := me.fieldIndex[field]; ok {
				for _, vectorIDs := range fieldValues {
					for _, id := range vectorIDs {
						matches[id] = struct{}{}
					}
				}
			}
		}
	case "nexists": // Field does not exist
		// This is more complex as it requires iterating through all metadata and checking absence.
		// For simplicity, this implementation only considers vectors *not* in the fieldIndex for this field.
		// A more robust solution might require iterating `metadataIndex`.

		allVectorIDs := me.getAllVectorIDs()
		existingFieldVectorIDs := make(map[string]struct{})
		if fieldValues, ok := me.fieldIndex[field]; ok {
			for _, vectorIDs := range fieldValues {
				for _, id := range vectorIDs {
					existingFieldVectorIDs[id] = struct{}{}
				}
			}
		}

		for _, id := range allVectorIDs {
			if _, exists := existingFieldVectorIDs[id]; !exists {
				matches[id] = struct{}{}
			}
		}
	default:
		fmt.Printf("Unsupported query operator: %s\n", op)
	}
}

// AddVector adds a vector to the metadata engine
func (me *MetadataEngine) AddVector(vector *Vector) {
	me.mu.Lock()
	defer me.mu.Unlock()

	// Add to metadata index
	me.metadataIndex[vector.ID] = vector.Metadata

	// Add to namespace index
	me.namespaceIndex[vector.Namespace] = append(me.namespaceIndex[vector.Namespace], vector.ID)

	// Add to field index
	for field, value := range vector.Metadata {
		me.addToFieldIndex(field, value, vector.ID)
	}
}

// UpdateVector updates a vector in the metadata engine
func (me *MetadataEngine) UpdateVector(vector *Vector) {
	me.mu.Lock()
	defer me.mu.Unlock()

	// Remove old metadata from field index
	if oldMetadata, ok := me.metadataIndex[vector.ID]; ok {
		for field, value := range oldMetadata {
			me.removeFromFieldIndex(field, value, vector.ID)
		}
	}

	// Update metadata index
	me.metadataIndex[vector.ID] = vector.Metadata

	// Add new metadata to field index
	for field, value := range vector.Metadata {
		me.addToFieldIndex(field, value, vector.ID)
	}
}

// DeleteVector removes a vector from the metadata engine
func (me *MetadataEngine) DeleteVector(vectorID string) {
	me.mu.Lock()
	defer me.mu.Unlock()

	// Remove from metadata index and field index
	if oldMetadata, ok := me.metadataIndex[vectorID]; ok {
		for field, value := range oldMetadata {
			me.removeFromFieldIndex(field, value, vectorID)
		}
		delete(me.metadataIndex, vectorID)
	}

	// Remove from namespace index
	for namespace, vectorIDs := range me.namespaceIndex {
		for i, id := range vectorIDs {
			if id == vectorID {
				me.namespaceIndex[namespace] = append(vectorIDs[:i], vectorIDs[i+1:]...)
				if len(me.namespaceIndex[namespace]) == 0 {
					delete(me.namespaceIndex, namespace)
				}
				break
			}
		}
	}
}

// Filter returns vector IDs that match the metadata filter within a namespace
func (me *MetadataEngine) Filter(namespace string, filter map[string]interface{}) []string {
	me.mu.RLock()
	defer me.mu.RUnlock()

	// Get all vector IDs in the namespace
	namespaceVectorIDs := me.namespaceIndex[namespace]
	if len(namespaceVectorIDs) == 0 {
		return []string{}
	}

	// If no filter, return all vector IDs in namespace
	if len(filter) == 0 {
		return namespaceVectorIDs
	}

	// Apply metadata filter
	filteredIDs := me.Query(filter)

	// Intersect with namespace vector IDs
	namespaceSet := make(map[string]struct{})
	for _, id := range namespaceVectorIDs {
		namespaceSet[id] = struct{}{}
	}

	result := []string{}
	for _, id := range filteredIDs {
		if _, exists := namespaceSet[id]; exists {
			result = append(result, id)
		}
	}

	return result
}

// convertToHashableKey converts complex types to hashable keys for map indexing
func (me *MetadataEngine) convertToHashableKey(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// Check if the value is already hashable
	switch v := value.(type) {
	case string:
		// For long strings (like content), use a hash instead of the full string
		if len(v) > 1000 {
			// Use a simple hash for very long strings to avoid memory issues
			hash := 0
			for _, char := range v {
				hash = (hash*31 + int(char)) % 1000000
			}
			return fmt.Sprintf("hash_%d", hash)
		}
		return v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return v
	case []interface{}:
		// Convert slice to JSON string
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", v)
	case map[string]interface{}:
		// Convert map to JSON string
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", v)
	default:
		// For other types, convert to string
		return fmt.Sprintf("%v", v)
	}
}
