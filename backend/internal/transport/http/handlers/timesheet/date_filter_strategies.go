package timesheet

import (
	"context"

	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
)

// DateFilterStrategy interface for date filtering strategies
type DateFilterStrategy interface {
	Apply(ctx context.Context, c *gin.Context, filters *domain.TimesheetFilters) error
	Name() string
}

// SpecificDateFilter handles specific date filtering
type SpecificDateFilter struct{}

// Not applicable, try next strategy

// MonthFilter handles month-based filtering
type MonthFilter struct{}

// DateRangeFilter handles date range filtering
type DateRangeFilter struct{}

// Check if already handled by other strategies

// Parse fromDate

// Parse toDate

// DateFilterProcessor chains date filter strategies
type DateFilterProcessor struct {
	strategies []DateFilterStrategy
}

// NewDateFilterProcessor creates a new date filter processor

// ApplyFilters applies all filter strategies in order

// First strategy that matches and fails
