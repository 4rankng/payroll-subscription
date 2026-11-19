package asynq

import (
	"context"
	"encoding/json"
	"fmt"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/workers"
	"api-server/internal/infra/observability"
)

var logger = observability.GetLogger().With("component", "asynq")

// Task type constants
const (
	TaskEmployeeImport = "employee:import"
	TaskBCCImport      = "timesheet:bcc_import"
	TaskBCCImportSweep = "timesheet:bcc_import_sweep"
	// TaskAuditLogWrite is the asynq task type for persisting audit log entries.
	// Replaces the in-process event-bus goroutine to add durable retries and bounded concurrency.
	TaskAuditLogWrite = "audit:log:write"
)

// Queue name constants
const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)

// Handlers holds all task handler dependencies
type Handlers struct {
	employeeImportWorker *workers.EmployeeImportWorker
	bccImportWorker      *workers.BCCImportWorker
	auditLogWriteWorker  *workers.AuditLogWriteWorker
}

// NewHandlers creates a new Handlers instance
func NewHandlers(
	employeeImportWorker *workers.EmployeeImportWorker,
	bccImportWorker *workers.BCCImportWorker,
	auditLogWriteWorker *workers.AuditLogWriteWorker,
) *Handlers {
	return &Handlers{
		employeeImportWorker: employeeImportWorker,
		bccImportWorker:      bccImportWorker,
		auditLogWriteWorker:  auditLogWriteWorker,
	}
}

type bccImportPayload struct {
	AssetID uint `json:"asset_id"`
}

func (h *Handlers) HandleBCCImport(ctx context.Context, t *asynqlib.Task) error {
	var payload bccImportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil || payload.AssetID == 0 {
		return asynqlib.SkipRetry
	}
	if h.bccImportWorker == nil {
		return nil
	}
	return h.bccImportWorker.ProcessJob(ctx, payload.AssetID)
}

func (h *Handlers) HandleBCCImportSweep(ctx context.Context, _ *asynqlib.Task) error {
	if h.bccImportWorker == nil {
		return nil
	}
	return h.bccImportWorker.Recover(ctx)
}

// HandleEmployeeImport processes employee import tasks
func (h *Handlers) HandleEmployeeImport(ctx context.Context, t *asynqlib.Task) error {
	var p employeeImportPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal employee import payload: %w", err)
	}

	logger.Info("Processing employee import task", "import_id", p.ImportID)

	if err := h.employeeImportWorker.ProcessJob(ctx, p.ImportID); err != nil {
		return fmt.Errorf("employee import failed: %w", err)
	}

	return nil
}

// HandleAuditLogWrite processes an audit log write task.
// Malformed payloads are dropped immediately (SkipRetry); DB errors are retried by asynq.
func (h *Handlers) HandleAuditLogWrite(ctx context.Context, t *asynqlib.Task) error {
	var p AuditLogWritePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		logger.Error("audit:log:write unmarshal payload", "error", err)
		return asynqlib.SkipRetry
	}

	job := workers.AuditLogWriteJob{
		UserID:       p.UserID,
		Action:       p.Action,
		EntityType:   p.EntityType,
		EntityID:     p.EntityID,
		Message:      p.Message,
		IPAddress:    p.IPAddress,
		UserAgent:    p.UserAgent,
		MetadataJSON: p.MetadataJSON,
		CreatedAt:    p.CreatedAt,
	}
	if err := h.auditLogWriteWorker.ProcessJob(ctx, job); err != nil {
		return fmt.Errorf("audit:log:write: %w", err)
	}
	return nil
}
