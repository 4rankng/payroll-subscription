package specs

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockWorkingDaysCounter is a mock implementation of WorkingDaysCounter
type MockWorkingDaysCounter struct {
	mock.Mock
}

func (m *MockWorkingDaysCounter) CountDistinctWorkingDays(ctx context.Context, employeeID, projectID uint, startDate, endDate time.Time) (int, error) {
	args := m.Called(ctx, employeeID, projectID, startDate, endDate)
	return args.Int(0), args.Error(1)
}

// Mock returns exactly the threshold

// Mock returns more than threshold

// Mock returns less than threshold

// Mock returns zero days
