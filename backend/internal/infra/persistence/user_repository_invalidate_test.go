package persistence

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserInvalidateTestDB(t *testing.T) (domain.UserRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email TEXT,
			password TEXT NOT NULL,
			fullname TEXT NOT NULL,
			cccd TEXT,
			mobile TEXT,
			role TEXT NOT NULL,
			deleted_at DATETIME,
			last_login DATETIME,
			tokens_invalid_before DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create users table: %v", err)
	}
	return NewUserRepository(&Database{DB: db}), db
}

// TestUpdateUserAndInvalidateSessions_CommitsTogether: a role change lands
// BOTH the row and the tokens_invalid_before stamp in one transaction.
func TestUpdateUserAndInvalidateSessions_CommitsTogether(t *testing.T) {
	repo, db := newUserInvalidateTestDB(t)
	ctx := context.Background()

	user := &domain.User{
		Username: "cuongnv",
		Password: "hashed-password",
		Fullname: "Nguyen Van Cuong",
		Role:     domain.RoleQuanly,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	invalidBefore := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	user.Role = domain.RoleNhanvien
	if err := repo.UpdateUserAndInvalidateSessions(ctx, user, invalidBefore); err != nil {
		t.Fatalf("update user and invalidate sessions: %v", err)
	}

	var stored domain.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if stored.Role != domain.RoleNhanvien {
		t.Fatalf("role = %s, want %s", stored.Role, domain.RoleNhanvien)
	}
	if stored.TokensInvalidBefore == nil {
		t.Fatal("tokens_invalid_before is nil, want the change instant")
	}
	if !stored.TokensInvalidBefore.Equal(invalidBefore) {
		t.Fatalf("tokens_invalid_before = %v, want %v", stored.TokensInvalidBefore, invalidBefore)
	}
}

// TestUpdateUserAndInvalidateSessions_RollsBackTogether: when the row write
// fails, nothing commits — the role must not change while the old sessions
// stay alive (that partial state is the bug this method exists to prevent).
func TestUpdateUserAndInvalidateSessions_RollsBackTogether(t *testing.T) {
	repo, db := newUserInvalidateTestDB(t)
	ctx := context.Background()

	seed := func(username string) *domain.User {
		u := &domain.User{
			Username: username,
			Password: "hashed-password",
			Fullname: "Seed User",
			Role:     domain.RoleQuanly,
		}
		if err := repo.Create(ctx, u); err != nil {
			t.Fatalf("seed user %s: %v", username, err)
		}
		return u
	}
	seed("taken-name")
	victim := seed("other-name")

	// Colliding with the other row's unique username makes the save fail.
	victim.Username = "taken-name"
	victim.Role = domain.RoleNhanvien
	invalidBefore := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	if err := repo.UpdateUserAndInvalidateSessions(ctx, victim, invalidBefore); err == nil {
		t.Fatal("expected the colliding save to fail")
	}

	var stored domain.User
	if err := db.First(&stored, victim.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if stored.Role != domain.RoleQuanly {
		t.Fatalf("role = %s after failed write, want %s (rollback)", stored.Role, domain.RoleQuanly)
	}
	if stored.TokensInvalidBefore != nil {
		t.Fatalf("tokens_invalid_before = %v after failed write, want nil (rollback)", stored.TokensInvalidBefore)
	}
}
