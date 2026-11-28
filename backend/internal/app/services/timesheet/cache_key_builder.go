package timesheet

import (
	"strings"

	"api-server/internal/domain"
)

// CacheKeyBuilder builds cache keys for timesheet queries using strategy pattern
type CacheKeyBuilder interface {
	Build(filters domain.TimesheetFilters) string
	Invalidate(key string) error
}

// FilterHandler handles individual filter components for cache key building
type FilterHandler interface {
	CanHandle(filters domain.TimesheetFilters) bool
	Apply(builder *strings.Builder, filters domain.TimesheetFilters)
}

// TimesheetCacheKeyBuilder implements cache key building with strategy pattern
type TimesheetCacheKeyBuilder struct {
	prefix   string
	handlers []FilterHandler
}

// NewTimesheetCacheKeyBuilder creates a new cache key builder

// Build builds a cache key from filters

// Invalidate is a placeholder for cache invalidation logic

// Cache invalidation would be handled by cache service

// ProjectIDFilterHandler handles project ID filter component
type ProjectIDFilterHandler struct{}

// EmployeeIDFilterHandler handles employee ID filter component
type EmployeeIDFilterHandler struct{}

// StatusFilterHandler handles status filter component
type StatusFilterHandler struct{}

// PaymentStatusFilterHandler handles payment status filter component
type PaymentStatusFilterHandler struct{}

// PayTypeFilterHandler handles pay type filter component
type PayTypeFilterHandler struct{}

// DateFilterHandler handles date range filter component
type DateFilterHandler struct{}
