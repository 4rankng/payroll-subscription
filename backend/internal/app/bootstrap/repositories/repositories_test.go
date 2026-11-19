package repositories

import (
	"testing"

	"api-server/internal/infra/persistence"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitialize(t *testing.T) {
	gormDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	db := &persistence.Database{DB: gormDB}

	repos := Initialize(db, nil)
	if repos == nil {
		t.Fatal("Initialize returned nil")
		return
	}

	for _, repo := range []struct {
		name string
		repo interface{}
	}{
		{"User", repos.User},
		{"Project", repos.Project},
		{"ProjectUser", repos.ProjectUser},
		{"Employee", repos.Employee},
		{"EmployeeUser", repos.EmployeeUser},
		{"ProjectEmployee", repos.ProjectEmployee},
		{"Payrate", repos.Payrate},
		{"Timesheet", repos.Timesheet},
		{"Notification", repos.Notification},
		{"Settings", repos.Settings},
		{"Asset", repos.Asset},
		{"PushSubscription", repos.PushSubscription},
		{"TransferUpload", repos.TransferUpload},
		{"EmployeePayCycle", repos.EmployeePayCycle},
		{"BillingReport", repos.BillingReport},
		{"TimesheetImportJob", repos.TimesheetImportJob},
		{"AuditLog", repos.AuditLog},
		{"BlacklistedToken", repos.BlacklistedToken},
	} {
		if repo.repo == nil {
			t.Errorf("%s repository is nil", repo.name)
		}
	}
}
