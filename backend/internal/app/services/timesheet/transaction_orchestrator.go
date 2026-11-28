package timesheet

import (
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
)

// TransactionOrchestrator handles transactional operations with event publishing
type TransactionOrchestrator struct {
	tm     *infrastructure.TransactionManager
	events domain.EventBus
}

// NewTransactionOrchestrator creates a new transaction orchestrator

// ExecuteInTransaction executes an operation within a transaction

// ExecuteInTransactionWithEvent executes an operation in transaction with optional event publishing

// Execute operation in transaction

// Publish event after successful transaction

// Log error but don't fail the operation
// Event publishing failure should be retried by outbox pattern

// ExecuteInTransactionWithEvents executes an operation in transaction with multiple events

// Execute operation in transaction

// Publish all events after successful transaction

// Log error but don't fail the operation
