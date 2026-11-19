package asynq

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/config"
)

// Client wraps asynq.Client for enqueueing tasks
type Client struct {
	client *asynqlib.Client
	cfg    config.AsynqConfig
}

// AsynqClient returns the underlying asynq client for direct task enqueueing.
func (c *Client) AsynqClient() *asynqlib.Client { return c.client }

// NewClient creates a new Asynq client
func NewClient(cfg config.AsynqConfig) (*Client, error) {
	client := asynqlib.NewClient(asynqlib.RedisClientOpt{
		Addr: cfg.RedisAddr,
		DB:   cfg.RedisDB,
	})

	return &Client{
		client: client,
		cfg:    cfg,
	}, nil
}

// EnqueueEmployeeImport enqueues an employee import task
func (c *Client) EnqueueEmployeeImport(importID string) error {
	payload, _ := json.Marshal(employeeImportPayload{ImportID: importID})

	task := asynqlib.NewTask(TaskEmployeeImport, payload,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(c.cfg.RetryMax),
		asynqlib.Unique(24*time.Hour),
		asynqlib.TaskID(fmt.Sprintf("employee-import:%s", importID)),
	)

	info, err := c.client.Enqueue(task)
	if err != nil {
		if errors.Is(err, asynqlib.ErrDuplicateTask) {
			return nil
		}
		return fmt.Errorf("failed to enqueue employee import task: %w", err)
	}

	logger.Info("Enqueued employee import task",
		"task_id", info.ID,
		"import_id", importID,
		"queue", info.Queue,
	)
	return nil
}

// EnqueueBCCImport enqueues one durable BCC import by asset/job ID.
func (c *Client) EnqueueBCCImport(assetID uint) error {
	payload, _ := json.Marshal(bccImportPayload{AssetID: assetID})
	task := asynqlib.NewTask(
		TaskBCCImport,
		payload,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(c.cfg.RetryMax),
		asynqlib.Timeout(10*time.Minute),
		asynqlib.Unique(time.Minute),
	)
	_, err := c.client.Enqueue(task)
	if errors.Is(err, asynqlib.ErrDuplicateTask) || errors.Is(err, asynqlib.ErrTaskIDConflict) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to enqueue BCC import task: %w", err)
	}
	return nil
}

// AuditEnqueuer is the minimal interface the audit event handler needs from the asynq client.
// Using an interface here keeps the handler testable without a real Redis connection.
type AuditEnqueuer interface {
	EnqueueAuditLogWrite(p AuditLogWritePayload) error
}

// AuditLogWritePayload is the asynq task payload for audit:log:write.
// The event handler builds it from a domain event; the worker deserialises and persists the row.
// CreatedAt is captured at enqueue time so retries preserve the original action timestamp.
type AuditLogWritePayload struct {
	UserID       uint      `json:"user_id"`
	Action       string    `json:"action"`
	EntityType   string    `json:"entity_type"`
	EntityID     *uint     `json:"entity_id,omitempty"`
	Message      string    `json:"message"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	MetadataJSON string    `json:"metadata_json,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// EnqueueAuditLogWrite enqueues a single audit log write task.
// No deduplication: every audit event is a distinct row.
func (c *Client) EnqueueAuditLogWrite(p AuditLogWritePayload) error {
	raw, _ := json.Marshal(p)

	task := asynqlib.NewTask(TaskAuditLogWrite, raw,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(c.cfg.RetryMax),
	)
	if _, err := c.client.Enqueue(task); err != nil {
		return fmt.Errorf("failed to enqueue audit log write task: %w", err)
	}
	return nil
}

// Close closes the asynq client connection
func (c *Client) Close() error {
	return c.client.Close()
}

type employeeImportPayload struct {
	ImportID string `json:"import_id"`
}
