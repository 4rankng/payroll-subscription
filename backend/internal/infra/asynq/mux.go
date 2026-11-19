package asynq

import (
	"fmt"

	asynqlib "github.com/hibiken/asynq"
)

// RegisterHandlers registers all task handlers on the server's mux
func RegisterHandlers(srv *Server, h *Handlers) {
	srv.Mux().Handle(TaskEmployeeImport, asynqlib.HandlerFunc(h.HandleEmployeeImport))
	srv.Mux().Handle(TaskBCCImport, asynqlib.HandlerFunc(h.HandleBCCImport))
	srv.Mux().Handle(TaskBCCImportSweep, asynqlib.HandlerFunc(h.HandleBCCImportSweep))
	srv.Mux().Handle(TaskAuditLogWrite, asynqlib.HandlerFunc(h.HandleAuditLogWrite))

	registered := []string{
		TaskEmployeeImport, TaskBCCImport, TaskBCCImportSweep,
		TaskAuditLogWrite,
	}
	for _, name := range registered {
		logger.Info("Registered handler", "task", name)
	}
}

// RegisterPeriodicTasks registers periodic tasks on the scheduler
func RegisterPeriodicTasks(srv *Server, _ *Client) error {
	if _, err := srv.Scheduler().Register(
		"@every 1m",
		asynqlib.NewTask(TaskBCCImportSweep, nil),
		asynqlib.Queue(QueueLow),
	); err != nil {
		return fmt.Errorf("failed to register BCC import recovery sweep: %w", err)
	}
	logger.Info(
		"Registered asynq periodic tasks",
		"bcc_import_recovery", "1m",
	)
	return nil
}
