package services

import (
	infraServices "api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
)

type InfrastructureServices struct {
	CacheService       *infraServices.CacheService
	TransactionManager *infraServices.TransactionManager
	EventBus           domain.EventBus
}
