package transaction

import (
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test model for transaction testing
type TestModel struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:100"`
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&TestModel{})
	assert.NoError(t, err)

	return db
}

func TestNewGormTransactionManager(t *testing.T) {
	db := setupTestDB(t)
	tm := NewGormTransactionManager(db)

	assert.NotNil(t, tm)
	assert.Implements(t, (*domain.TransactionManager)(nil), tm)
}

// Verify transaction context is set

// Create test data

// Verify data was committed

// Create test data

// Force rollback

// Verify data was rolled back

// Create test data

// Verify data was committed

// Create test data

// Verify data was rolled back
