package db

import (
	"context"

	"gorm.io/gorm"
)

// BatchLoader provides utilities for batch loading related entities to prevent N+1 queries.
// This pattern is commonly used when loading relationships for a collection of entities.
type BatchLoader struct {
	ctx context.Context
	db  *gorm.DB
}

// NewBatchLoader creates a new BatchLoader instance

// ExtractIDs extracts unique IDs from a collection based on a key extractor function.
// This is typically used to collect foreign keys before batch loading.
//
// Example:
//
//	employeeIDs := ExtractIDs(timesheets, func(ts *Timesheet) uint { return ts.EmployeeID })

// Use map to ensure uniqueness

// Skip zero IDs

// Convert map keys to slice

// LoadToMap loads entities by IDs and returns them as a map keyed by ID.
// This is useful for quick lookups when assigning relationships.
//
// Example:
//
//	var employees []*Employee
//	employeeMap := LoadToMap(ctx, db, employeeIDs, &employees, "id")

// Load entities

// Convert to map

// Use reflection to get ID field value
// For simplicity, we'll use a callback approach

// getIDFromEntity is a helper to extract ID from an entity
// This is a simplified version - in production you might want to use reflection
// or require entities to implement an interface like type Identifiable interface { GetID() uint }

// This is a placeholder - actual implementation would use reflection or interface
// For now, we'll return 0 and let the caller handle the mapping

// BatchLoadProjects is a concrete batch loader for projects
// This demonstrates the pattern that can be used for other entities

// Replace with actual Project type

// Map projects by ID
// This is a template - actual implementation needs proper type handling

// CreateIDMap creates a map of entities keyed by their ID
// This is a generic helper for creating lookup maps

// CreatePtrIDMap creates a map of entity pointers keyed by their ID
// This is useful when you need to return pointers from the map

// CollectIDs collects IDs from items where the condition is true
// This is useful for conditional ID extraction
