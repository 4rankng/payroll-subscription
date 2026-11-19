package persistence

import (
	"context"
	"testing"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/query_builders"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Duplicate mobile numbers are permitted by the schema but the number is a
// login credential (phone login, self check-in), so a single-row lookup must
// refuse to choose between employees rather than returning an arbitrary row.
// Deletes are hard DELETEs (D16) — no soft-deleted fixtures exist.
func TestEmployeeRepository_GetByMobile_FailsClosedOnDuplicates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/mobile.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
	CREATE TABLE employees (id INTEGER PRIMARY KEY, fullname TEXT, mobile TEXT, cccd TEXT,
		created_by INTEGER);
	CREATE TABLE users (id INTEGER PRIMARY KEY, fullname TEXT);
	INSERT INTO employees (id, fullname, mobile, cccd, created_by) VALUES
		(1, 'Người thứ nhất', '0900000001', '031000000001', 9),
		(2, 'Người thứ hai',  '0900000001', '031000000002', 9),
		(3, 'Người duy nhất', '0900000002', '031000000003', 9);
	INSERT INTO users (id, fullname) VALUES (9, 'Người tạo');
	`).Error)

	repo := &EmployeeRepository{
		BaseRepository: &BaseRepository{DB: db},
		queryBuilder:   query_builders.NewEmployeeQueryBuilder(db),
	}
	ctx := context.Background()

	t.Run("duplicate mobile is a conflict", func(t *testing.T) {
		employee, err := repo.GetByMobile(ctx, "0900000001")
		require.Error(t, err)
		require.Nil(t, employee)
		require.True(t, domain.IsConflictError(err), "want conflict error, got %v", err)
	})

	t.Run("list reports duplicates only", func(t *testing.T) {
		matches, err := repo.ListByMobile(ctx, "0900000001")
		require.NoError(t, err)
		require.Len(t, matches, 2)
		require.Equal(t, uint(1), matches[0].ID, "ordered by id")
		require.Equal(t, uint(2), matches[1].ID)
	})

	t.Run("unique mobile resolves", func(t *testing.T) {
		employee, err := repo.GetByMobile(ctx, "0900000002")
		require.NoError(t, err)
		require.Equal(t, uint(3), employee.ID)
	})

	t.Run("unknown mobile is not found", func(t *testing.T) {
		employee, err := repo.GetByMobile(ctx, "0900000009")
		require.Error(t, err)
		require.Nil(t, employee)
		require.True(t, domain.IsNotFoundError(err), "want not-found error, got %v", err)
	})
}
