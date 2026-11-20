package seed

import (
	"context"
	"fmt"

	"api-server/internal/config"
	"api-server/internal/domain"
	"api-server/internal/infra/persistence"
	"api-server/internal/pkg/password"

	"gorm.io/gorm"
)

// devUser is one local-dev login account ensured by EnsureDevUsers.
type devUser struct {
	username string
	fullname string
	role     domain.UserRole
}

// devUsers are the standard local-dev logins, one per deployment role. The
// D19 role rename mapped accountant (ketoan) → giamdoc, so `ketoan` is a
// second giamdoc-role account kept under its legacy name.
var devUsers = []devUser{
	{username: "admin", fullname: "System Administrator", role: domain.RoleAdmin},
	{username: "quanly", fullname: "Quản lý", role: domain.RoleQuanly},
	{username: "giamdoc", fullname: "Giám đốc", role: domain.RoleGiamdoc},
	{username: "ketoan", fullname: "Kế toán (giamdoc)", role: domain.RoleGiamdoc},
}

// devUserPassword is the shared local-dev password. Never applied outside
// non-production environments — the caller guards on APP_ENV.
const devUserPassword = "Abc123"

// EnsureDevUsers creates or updates the standard local-dev login accounts
// (admin, quanly, giamdoc, ketoan — password Abc123). Idempotent: existing
// rows get their role and password re-asserted so a lost password never
// locks a developer out. Non-production only.
func EnsureDevUsers(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	userRepo := persistence.NewUserRepository(&persistence.Database{DB: db})

	hashed, err := password.HashPassword(devUserPassword, cfg.Security.HashSecret, cfg.Security.HashSalt)
	if err != nil {
		return fmt.Errorf("hash dev users password: %w", err)
	}

	for _, du := range devUsers {
		du := du
		user, err := userRepo.GetByUsername(ctx, du.username)
		if err != nil {
			if de, ok := err.(*domain.DomainError); ok && de.Type == "NOT_FOUND" {
				u := &domain.User{
					Username: du.username,
					Password: hashed,
					Fullname: du.fullname,
					Role:     du.role,
				}
				if cErr := userRepo.Create(ctx, u); cErr != nil {
					return fmt.Errorf("create dev user %s: %w", du.username, cErr)
				}
				fmt.Println("dev user created:", du.username)
				continue
			}
			return fmt.Errorf("lookup dev user %s: %w", du.username, err)
		}

		changed := false
		if user.Role != du.role {
			user.Role = du.role
			changed = true
		}
		if user.Fullname != du.fullname {
			user.Fullname = du.fullname
			changed = true
		}
		if user.Password != hashed {
			user.Password = hashed
			changed = true
		}
		if !changed {
			continue
		}
		if uErr := userRepo.Update(ctx, user); uErr != nil {
			return fmt.Errorf("update dev user %s: %w", du.username, uErr)
		}
		fmt.Println("dev user updated:", du.username)
	}
	return nil
}
