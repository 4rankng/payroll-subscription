package events

import (
	"context"

	"api-server/internal/app/services/employee"
	"api-server/internal/domain"
)

// UserEmployeeProfileHandler keeps the "every nhanvien account has a linked
// employee profile" invariant at the write path (PAYSUB-28): creating a
// nhanvien user or converting an account to nhanvien provisions the employee
// shell right away, so the account appears in employee lists before its first
// login instead of dead-ending the self-service page with 404.
type UserEmployeeProfileHandler struct {
	profiles *employee.EmployeeProfileService
}

func NewUserEmployeeProfileHandler(profiles *employee.EmployeeProfileService) *UserEmployeeProfileHandler {
	return &UserEmployeeProfileHandler{profiles: profiles}
}

func (h *UserEmployeeProfileHandler) CanHandle(eventType string) bool {
	return eventType == "UserCreated" || eventType == "UserUpdated"
}

func (h *UserEmployeeProfileHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	var userID uint
	var role domain.UserRole
	switch e := event.(type) {
	case domain.UserCreatedEvent:
		userID, role = e.AggregateID(), e.Role
	case domain.UserUpdatedEvent:
		userID, role = e.AggregateID(), e.Role
	default:
		return nil
	}
	if role != domain.RoleNhanvien {
		return nil
	}
	// Idempotent: accounts that already have a profile are returned untouched.
	_, err := h.profiles.EnsureProfileForUser(ctx, userID)
	return err
}
