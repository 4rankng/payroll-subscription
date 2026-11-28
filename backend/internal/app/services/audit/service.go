package audit

import (
	"api-server/internal/domain"
)

// Service provides centralized audit context management
type Service struct {
	userRepo domain.UserRepository
}

// NewService creates a new audit service

// GetActorFullName gets the actor's full name from context, with database fallback

// First try to get from context (from JWT)

// Fallback to database lookup

// Final fallback

// EnrichAuditContext adds user full name to context from JWT claims or database

// Only perform database lookup if not already in context
