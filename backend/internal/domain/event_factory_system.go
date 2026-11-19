package domain

import (
	"context"
	"fmt"

	auditctx "api-server/internal/pkg/context"
)

// NewDataImportedEvent creates a DataImportedEvent
func NewDataImportedEvent(ctx context.Context, dataType string, recordCount int, fileName string) DataImportedEvent {
	auditMessage := fmt.Sprintf("%s đã nhập %d bản ghi %s từ tệp %s", auditctx.GetFullName(ctx), recordCount, dataType, fileName)
	return DataImportedEvent{
		BaseEvent:   newBaseEventWithAudit(ctx, "DataImported", 0, AuditActionImport, EntityTypeAsset, auditMessage),
		DataType:    dataType,
		RecordCount: recordCount,
		FileName:    fileName,
	}
}

// NewDataExportedEvent creates a DataExportedEvent
func NewDataExportedEvent(ctx context.Context, dataType string, recordCount int, fileName string) DataExportedEvent {
	auditMessage := fmt.Sprintf("%s đã xuất %d bản ghi %s ra tệp %s", auditctx.GetFullName(ctx), recordCount, dataType, fileName)
	return DataExportedEvent{
		BaseEvent:   newBaseEventWithAudit(ctx, "DataExported", 0, AuditActionExport, EntityTypeAsset, auditMessage),
		DataType:    dataType,
		RecordCount: recordCount,
		FileName:    fileName,
	}
}

func NewPayrateCreatedEvent(ctx context.Context, payrate *Payrate) PayrateCreatedEvent {
	projectName := payrateProjectName(payrate)
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypePayrate, actorName, projectName)
	return PayrateCreatedEvent{
		BaseEvent:   newBaseEventWithActor(ctx, "PayrateCreated", payrate.ID, getUserIDFromContext(ctx), AuditActionCreate, EntityTypePayrate, auditMessage),
		ProjectID:   payrate.ProjectID,
		ProjectName: projectName,
		EmployeeID:  0, // Payrate is per project, not per employee
		Rate:        0, // Rate is in PayrateJSON field, not a simple int64
	}
}

// NewPayrateUpdatedEvent creates a PayrateUpdatedEvent
func NewPayrateUpdatedEvent(ctx context.Context, payrate *Payrate) PayrateUpdatedEvent {
	projectName := payrateProjectName(payrate)
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypePayrate, actorName, projectName)
	return PayrateUpdatedEvent{
		BaseEvent:   newBaseEventWithActor(ctx, "PayrateUpdated", payrate.ID, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypePayrate, auditMessage),
		ProjectID:   payrate.ProjectID,
		ProjectName: projectName,
		EmployeeID:  0,
		Rate:        0,
	}
}

// NewPayrateDeletedEvent creates a PayrateDeletedEvent
func NewPayrateDeletedEvent(ctx context.Context, payrate *Payrate) PayrateDeletedEvent {
	projectName := payrateProjectName(payrate)
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypePayrate, actorName, projectName)
	return PayrateDeletedEvent{
		BaseEvent:   newBaseEventWithActor(ctx, "PayrateDeleted", payrate.ID, getUserIDFromContext(ctx), AuditActionDelete, EntityTypePayrate, auditMessage),
		ProjectID:   payrate.ProjectID,
		ProjectName: projectName,
		EmployeeID:  0,
	}
}

// payrateProjectName resolves the display project for a payrate event; the
// joined Project row is not always preloaded.
func payrateProjectName(payrate *Payrate) string {
	if payrate.Project.ID != 0 {
		return payrate.Project.Name
	}
	return ""
}

// NewAssetCreatedEvent creates an AssetCreatedEvent
func NewAssetCreatedEvent(ctx context.Context, asset *Asset) AssetCreatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypeAsset, actorName, asset.Filename)
	return AssetCreatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "AssetCreated", asset.ID, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeAsset, auditMessage),
		FileName:  asset.Filename,
	}
}

// NewAssetDeletedEvent creates an AssetDeletedEvent
func NewAssetDeletedEvent(ctx context.Context, asset *Asset) AssetDeletedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypeAsset, actorName, asset.Filename)
	return AssetDeletedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "AssetDeleted", asset.ID, getUserIDFromContext(ctx), AuditActionDelete, EntityTypeAsset, auditMessage),
		FileName:  asset.Filename,
	}
}

func NewSettingsCreatedEvent(ctx context.Context, key string, entityID uint) SettingsCreatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypeSettings, actorName, key)
	return SettingsCreatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "SettingsCreated", entityID, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeSettings, auditMessage),
		Key:       key,
	}
}

// NewSettingsUpdatedEvent creates a SettingsUpdatedEvent
func NewSettingsUpdatedEvent(ctx context.Context, key string, entityID uint) SettingsUpdatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypeSettings, actorName, key)
	return SettingsUpdatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "SettingsUpdated", entityID, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypeSettings, auditMessage),
		Key:       key,
	}
}

// NewSettingsDeletedEvent creates a SettingsDeletedEvent
func NewSettingsDeletedEvent(ctx context.Context, key string, entityID uint) SettingsDeletedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypeSettings, actorName, key)
	return SettingsDeletedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "SettingsDeleted", entityID, getUserIDFromContext(ctx), AuditActionDelete, EntityTypeSettings, auditMessage),
		Key:       key,
	}
}
