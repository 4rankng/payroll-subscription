package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProject_ValidateName(t *testing.T) {
	p := &Project{}

	// Test valid name
	p.Name = "Test Project"
	err := p.ValidateName()
	assert.NoError(t, err)

	// Test empty name
	p.Name = ""
	err = p.ValidateName()
	assert.Error(t, err)
}

func TestProject_ValidateStatus(t *testing.T) {
	p := &Project{}

	// Test valid status
	p.ProjectStatus = ProjectStatusDraft
	err := p.ValidateStatus()
	assert.NoError(t, err)

	p.ProjectStatus = ProjectStatusRunning
	err = p.ValidateStatus()
	assert.NoError(t, err)

	p.ProjectStatus = ProjectStatusCompleted
	err = p.ValidateStatus()
	assert.NoError(t, err)

	p.ProjectStatus = ProjectStatusCancelled
	err = p.ValidateStatus()
	assert.NoError(t, err)

	// Test invalid status
	p.ProjectStatus = "invalid"
	err = p.ValidateStatus()
	assert.Error(t, err)
}

func TestProject_IsRunning(t *testing.T) {
	p := &Project{
		ProjectStatus: ProjectStatusRunning,
	}

	assert.True(t, p.IsRunning())

	p.ProjectStatus = ProjectStatusDraft
	assert.False(t, p.IsRunning())
}

func TestProject_IsDraft(t *testing.T) {
	p := &Project{
		ProjectStatus: ProjectStatusDraft,
	}

	assert.True(t, p.IsDraft())

	p.ProjectStatus = ProjectStatusRunning
	assert.False(t, p.IsDraft())
}

func TestProject_CanTransitionTo(t *testing.T) {
	p := &Project{
		ProjectStatus: ProjectStatusDraft,
	}

	// From draft, can transition to running
	assert.True(t, p.CanTransitionTo(ProjectStatusRunning))
	assert.False(t, p.CanTransitionTo(ProjectStatusCompleted))

	// From running, can transition to various states
	p.ProjectStatus = ProjectStatusRunning
	assert.True(t, p.CanTransitionTo(ProjectStatusPaused))
	assert.True(t, p.CanTransitionTo(ProjectStatusCompleted))
	assert.True(t, p.CanTransitionTo(ProjectStatusCancelled))
	assert.False(t, p.CanTransitionTo(ProjectStatusDraft))

	// From completed, no transitions allowed
	p.ProjectStatus = ProjectStatusCompleted
	assert.False(t, p.CanTransitionTo(ProjectStatusDraft))
	assert.False(t, p.CanTransitionTo(ProjectStatusRunning))
	assert.False(t, p.CanTransitionTo(ProjectStatusPaused))
	assert.False(t, p.CanTransitionTo(ProjectStatusCancelled))

	// From cancelled, no transitions allowed
	p.ProjectStatus = ProjectStatusCancelled
	assert.False(t, p.CanTransitionTo(ProjectStatusDraft))
	assert.False(t, p.CanTransitionTo(ProjectStatusRunning))
	assert.False(t, p.CanTransitionTo(ProjectStatusCompleted))
	assert.False(t, p.CanTransitionTo(ProjectStatusPaused))

	// From paused, can transition back to running or complete/cancel
	p.ProjectStatus = ProjectStatusPaused
	assert.True(t, p.CanTransitionTo(ProjectStatusRunning))
	assert.True(t, p.CanTransitionTo(ProjectStatusCompleted))
	assert.True(t, p.CanTransitionTo(ProjectStatusCancelled))
	assert.False(t, p.CanTransitionTo(ProjectStatusDraft))
}

func TestProject_IsValid(t *testing.T) {
	// Test valid project
	p := &Project{
		Name:          "Test Project",
		ProjectStatus: ProjectStatusRunning,
	}
	err := p.IsValid()
	assert.NoError(t, err)

	// Test invalid name
	p2 := &Project{
		Name: "",
	}
	err = p2.IsValid()
	assert.Error(t, err)
}

func TestProject_GetAuditEntityType(t *testing.T) {
	p := Project{}
	entityType := p.GetAuditEntityType()
	assert.Equal(t, "project", entityType)
}

func TestProject_GetAuditEntityID(t *testing.T) {
	p := Project{ID: 789}
	entityID := p.GetAuditEntityID()
	assert.Equal(t, uint(789), entityID)
}
