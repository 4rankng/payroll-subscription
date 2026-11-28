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

// devUsers are the standard local-dev logins, one per deployment role.
// PAYSUB D28 (founder 2026-11-27): ketoan is its own role again — the D19
// ketoan→giamdoc merge is superseded; role enum is
// admin/quanly/ketoan/giamdoc/nhanvien.
var devUsers = []devUser{
	{username: "admin", fullname: "Trần Quốc Bảo", role: domain.RoleAdmin},
	{username: "quanly", fullname: "Nguyễn Văn Hùng", role: domain.RoleQuanly},
	{username: "ketoan", fullname: "Lê Thị Thu Hà", role: domain.RoleKetoan},
	{username: "giamdoc", fullname: "Phạm Đức Anh", role: domain.RoleGiamdoc},
	{username: "nhanvien", fullname: "Vũ Ngọc Mai", role: domain.RoleNhanvien},
}

// devUserPassword is the shared local-dev password. Never applied outside
// non-production environments — the caller guards on APP_ENV.
const devUserPassword = "Abc123"

// EnsureDevUsers creates or updates the standard local-dev login accounts
// (admin, quanly, ketoan, giamdoc, nhanvien — password Abc123). Idempotent:
// existing rows get their role and password re-asserted so a lost password
// never locks a developer out. Non-production only.
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

	// Give the dev nhanvien account an employee profile so /api/v1/me and the
	// employee mobile views resolve instead of 404ing on a bare user. Idempotent:
	// only created when no employee row links to the account.
	nvUser, err := userRepo.GetByUsername(ctx, "nhanvien")
	if err != nil {
		return fmt.Errorf("lookup dev user nhanvien for employee profile: %w", err)
	}
	var empCount int64
	if err := db.Model(&domain.Employee{}).Where("user_id = ?", nvUser.ID).Count(&empCount).Error; err != nil {
		return fmt.Errorf("count employee profile for dev nhanvien: %w", err)
	}
	if empCount == 0 {
		emp := &domain.Employee{
			Fullname:  nvUser.Fullname,
			CCCD:      "000000000001",
			UserID:    &nvUser.ID,
			CreatedBy: nvUser.ID,
		}
		if err := db.Create(emp).Error; err != nil {
			return fmt.Errorf("create employee profile for dev nhanvien: %w", err)
		}
		fmt.Println("dev employee profile created for nhanvien")
	} else if err := db.Model(&domain.Employee{}).
		Where("user_id = ? AND fullname <> ?", nvUser.ID, nvUser.Fullname).
		Update("fullname", nvUser.Fullname).Error; err != nil {
		return fmt.Errorf("sync employee profile name for dev nhanvien: %w", err)
	}
	return nil
}
