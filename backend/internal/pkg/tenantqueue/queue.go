package tenantqueue

import (
	"context"
	"sync"
)

// Task represents a job to be executed with context
type Task func(ctx context.Context)

// TenantQueue holds a worker pool for a tenant
type TenantQueue struct {
	tasks chan Task
	wg    sync.WaitGroup
}

// NewTenantQueue creates a new queue with n workers

// Run task with background context; caller can include timeouts inside task

// Enqueue adds a task to the queue (non-blocking if buffer available)
// tenant is the tenant identifier used to label metrics

// record a queue drop metric for this tenant

// Shutdown waits for workers to finish after closing the queue
