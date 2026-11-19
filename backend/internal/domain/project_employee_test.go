package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProjectEmployee_ValidateProjectID(t *testing.T) {
	pe := &ProjectEmployee{}

	// Test valid project ID
	pe.ProjectID = 1
	err := pe.ValidateProjectID()
	assert.NoError(t, err)

	// Test invalid project ID
	pe.ProjectID = 0
	err = pe.ValidateProjectID()
	assert.Error(t, err)
}

func TestProjectEmployee_ValidateEmployeeID(t *testing.T) {
	pe := &ProjectEmployee{}

	// Test valid employee ID
	pe.EmployeeID = 1
	err := pe.ValidateEmployeeID()
	assert.NoError(t, err)

	// Test invalid employee ID
	pe.EmployeeID = 0
	err = pe.ValidateEmployeeID()
	assert.Error(t, err)
}

func TestProjectEmployee_ValidateDates(t *testing.T) {
	// Valid: start date present (rows are open-ended, no end date exists)
	pe := &ProjectEmployee{
		StartDate: time.Now().AddDate(0, 0, 1),
	}
	err := pe.ValidateDates()
	assert.NoError(t, err)

	// Invalid: zero start date
	pe.StartDate = time.Time{}
	err = pe.ValidateDates()
	assert.Error(t, err)
}

func TestProjectEmployee_ValidatePosition(t *testing.T) {
	pe := &ProjectEmployee{}

	// Test valid position
	pe.Position = "Developer"
	err := pe.ValidatePosition()
	assert.NoError(t, err)

	// Test empty position
	pe.Position = ""
	err = pe.ValidatePosition()
	assert.Error(t, err)
}

func TestProjectEmployee_IsValid(t *testing.T) {
	pe := &ProjectEmployee{
		ProjectID:  1,
		EmployeeID: 1,
		Position:   "Developer",
		StartDate:  time.Now().AddDate(0, 0, 1),
	}

	// Test valid project employee
	err := pe.IsValid()
	assert.NoError(t, err)

	// Test invalid project employee
	pe.ProjectID = 0
	err = pe.IsValid()
	assert.Error(t, err)
}

func TestProjectEmployee_CanCreateTimesheet(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -5) // 5 days ago

	pe := &ProjectEmployee{
		StartDate: startDate,
	}

	// Should be able to create timesheet for today
	assert.True(t, pe.CanCreateTimesheet(now))

	// Should not be able to create timesheet before start date
	beforeStartDate := now.AddDate(0, 0, -10)
	assert.False(t, pe.CanCreateTimesheet(beforeStartDate))
}

func TestProjectEmployee_IsAssignedOnDate(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -5) // 5 days ago

	pe := &ProjectEmployee{
		StartDate: startDate,
	}

	// Should be assigned on today
	assert.True(t, pe.IsAssignedOnDate(now))

	// Should not be assigned before start date
	beforeStartDate := now.AddDate(0, 0, -10)
	assert.False(t, pe.IsAssignedOnDate(beforeStartDate))
}
