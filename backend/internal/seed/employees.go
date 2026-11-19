package seed

import (
	"context"
	"fmt"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence"

	"gorm.io/gorm"
)

func (s *Seeder) seedEmployees(ctx context.Context, db *gorm.DB) error {
	employeeRepo := persistence.NewEmployeeRepository(&persistence.Database{DB: db})

	var adminUser domain.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		return fmt.Errorf("find admin user: %w", err)
	}

	// Track existing emails, usernames and CCCD numbers to avoid duplicates
	existingEmails := make(map[string]bool)
	existingUsernames := make(map[string]bool)
	existingCCCDs := make(map[string]bool)

	var existingEmployees []domain.Employee
	if err := db.Find(&existingEmployees).Error; err != nil {
		return fmt.Errorf("check existing employees: %w", err)
	}

	// Also check existing users for email conflicts
	var existingUsers []domain.User
	if err := db.Find(&existingUsers).Error; err != nil {
		return fmt.Errorf("check existing users: %w", err)
	}

	for _, emp := range existingEmployees {
		if emp.Email != nil && *emp.Email != "" {
			existingEmails[*emp.Email] = true
		}
		existingCCCDs[emp.CCCD] = true
	}

	for _, user := range existingUsers {
		if user.Email != nil {
			existingEmails[*user.Email] = true
		}
		existingUsernames[user.Username] = true
	}

	// Generate diverse employee profiles with different experience levels
	numEmployees := 120
	employeeProfiles := make([]*EmployeeProfile, 0, numEmployees)

	for i := 0; i < numEmployees; i++ {
		profile := GenerateRealisticEmployeeProfile(existingUsernames, existingEmails, existingCCCDs)
		employeeProfiles = append(employeeProfiles, profile)
	}

	// Create employees with realistic profiles
	for _, profile := range employeeProfiles {
		var email *string
		if profile.Email != "" {
			email = &profile.Email
		}

		employee := &domain.Employee{
			Fullname:  profile.FullName,
			Email:     email,
			CCCD:      profile.CCCD,
			Mobile:    profile.Phone,
			CreatedBy: adminUser.ID,
		}

		if err := employeeRepo.Create(ctx, employee); err != nil {
			return fmt.Errorf("create employee %s: %w", employee.Fullname, err)
		}
	}

	fmt.Printf("    Created %d employees with diverse skill levels and realistic profiles\n", len(employeeProfiles))
	return nil
}
