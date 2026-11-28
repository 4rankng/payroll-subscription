package tenantqueue

import (
	"sync"
)

// Manager keeps per-tenant queues
type Manager struct {
	mu      sync.Mutex
	queues  map[string]*TenantQueue
	workers int
	buffer  int
}
