package persistence

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"api-server/internal/domain"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/stretchr/testify/require"
)

// getTestDB connects to the MySQL server the integration fixtures run
// against (normally the shared dev database), taking TEST_MYSQL_DSN or
// falling back to DB_DSN. An unset DSN or unreachable server returns an
// error so the caller can skip instead of failing.
func getTestDB() (*gorm.DB, error) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		dsn = os.Getenv("DB_DSN")
	}
	if dsn == "" {
		return nil, fmt.Errorf("no test database configured: set TEST_MYSQL_DSN or DB_DSN")
	}
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

// Fixture IDs live far beyond any real or integration-suite sequence so the
// cleanup can never delete rows it does not own. The shared dev database also
// contains real rows, so assertions target only these fixture employees.
const (
	fxUser     = uint64(214000000001)
	fxProjectA = uint64(214000000001)
	fxProjectB = uint64(214000000002)
	fxPayrate  = uint64(214000000001)
	fxEmpAn    = uint64(214000000101) // salary, working
	fxEmpBinh  = uint64(214000000102) // has mobile
	fxEmpGiang = uint64(214000000105) // soft-deleted
	fxEmpLan   = uint64(214000000107) // boundary dates
	fxEmpPhuc  = uint64(214000000108) // NULL mobile
	fxPE       = uint64(214000000101)
	fxTsBase   = uint64(214000000201) // +0..8
	fxBandLow  = uint64(214000000001)
	fxBandHigh = uint64(214000000999)
)

// GetPaidWithoutMobile feeds the admin data-quality export: employees who
// received salary (paid timesheets) inside the window but have no mobile on
// file. These tests pin the salary-source semantics, the mobile/soft-delete
// filters, per-source aggregates, the working-status flag, and window
// boundaries. Runs against MySQL because the aggregate timestamps scan as
// engine datetimes.
func TestEmployeeRepository_GetPaidWithoutMobile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB test in short mode")
	}
	db, err := getTestDB()
	if err != nil {
		t.Skipf("cannot connect to test DB: %v", err)
	}

	cleanup := func() {
		for _, stmt := range []string{
			`DELETE FROM timesheets WHERE id BETWEEN ? AND ?`,
			`DELETE FROM payrates WHERE id BETWEEN ? AND ?`,
			`DELETE FROM project_employees WHERE id BETWEEN ? AND ?`,
			// project_users is the one FK child missing here historically: a
			// leftover row blocks DELETE FROM projects (fk_project_users_project)
			// and DELETE FROM users (fk_projects_creator) silently — the deletes
			// error and are ignored — so the next run collides with
			// "Duplicate entry" on the fixture id band. Clean it by both
			// referencing columns before the parents.
			`DELETE FROM project_users WHERE project_id BETWEEN ? AND ?`,
			`DELETE FROM project_users WHERE user_id BETWEEN ? AND ?`,
			`DELETE FROM project_users WHERE granted_by BETWEEN ? AND ?`,
			`DELETE FROM employees WHERE id BETWEEN ? AND ?`,
			`DELETE FROM projects WHERE id BETWEEN ? AND ?`,
			`DELETE FROM users WHERE id BETWEEN ? AND ?`,
		} {
			_ = db.Exec(stmt, fxBandLow, fxBandHigh).Error
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	loc := time.Local
	mustExec := func(q string, args ...any) {
		require.NoError(t, db.Exec(q, args...).Error)
	}

	// Window [2026-07-01, 2026-09-30 18:00]: both boundaries inclusive.
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, loc)
	to := time.Date(2026, 9, 30, 18, 0, 0, 0, loc)

	// Creator satisfies the created_by foreign keys.
	mustExec(`INSERT INTO users (id, username, password, fullname, role) VALUES (?, 'paidnomobile-fixture', 'x', 'Người tạo test', 'quanly')`, fxUser)
	mustExec(`INSERT INTO projects (id, name, created_by) VALUES (?, 'Dự án Alpha', ?), (?, 'Dự án Beta', ?)`, fxProjectA, fxUser, fxProjectB, fxUser)
	mustExec(`INSERT INTO payrates (id, project_id, payrate_json, from_date, created_by) VALUES (?, ?, '{"normal": 10000}', '2026-01-01', ?)`, fxPayrate, fxProjectA, fxUser)

	// An: salary, working; Bình: has mobile; Giang: soft-deleted; Lan:
	// boundary dates; Phúc: NULL mobile.
	mustExec(`INSERT INTO employees (id, fullname, cccd, mobile, deleted_at, created_by) VALUES
		(?, 'Nguyễn Văn An', '214000000101', '', NULL, ?),
		(?, 'Trần Thị Bình', '214000000102', '0900000002', NULL, ?),
		(?, 'Đỗ Thị Giang', '214000000105', '', '2026-08-01 00:00:00', ?),
		(?, 'Đặng Văn Lan', '214000000107', '', NULL, ?),
		(?, 'Bùi Thị Phúc', '214000000108', NULL, NULL, ?)`,
		fxEmpAn, fxUser, fxEmpBinh, fxUser, fxEmpGiang, fxUser,
		fxEmpLan, fxUser, fxEmpPhuc, fxUser)

	// Only An has an active assignment => working.
	mustExec(`INSERT INTO project_employees (id, project_id, employee_id, employee_name, employee_cccd, start_date, last_date, created_by) VALUES
		(?, ?, ?, 'Nguyễn Văn An', '214000000101', '2026-01-01', NULL, ?)`,
		fxPE, fxProjectA, fxEmpAn, fxUser)

	mustExec(`INSERT INTO timesheets (id, project_id, employee_id, payrate_id, date, hours_worked, paytype, payrate, amount, timesheet_status, payment_status, paid_amount, paid_at, created_by) VALUES
		(?, ?, ?, ?, '2026-08-10', 8, 'test', 0, 0, 'approved', 'paid', 2000000, ?, ?),
		(?, ?, ?, ?, '2026-08-20', 8, 'test', 0, 0, 'approved', 'paid', 1500000, ?, ?),
		(?, ?, ?, ?, '2026-08-21', 8, 'test', 0, 0, 'approved', 'pending', 900000, ?, ?),
		(?, ?, ?, ?, '2026-08-11', 8, 'test', 0, 0, 'approved', 'paid', 500000, ?, ?),
		(?, ?, ?, ?, '2026-08-12', 8, 'test', 0, 0, 'approved', 'paid', 700000, ?, ?),
		(?, ?, ?, ?, '2026-09-30', 8, 'test', 0, 0, 'approved', 'paid', 800000, ?, ?),
		(?, ?, ?, ?, '2026-08-01', 8, 'test', 0, 0, 'approved', 'paid', 400000, ?, ?),
		(?, ?, ?, ?, '2026-08-02', 8, 'test', 0, 0, 'approved', 'paid', 600000, ?, ?),
		(?, ?, ?, ?, '2026-08-03', 8, 'test', 0, 0, 'approved', 'paid', 200000, ?, ?)`,
		fxTsBase, fxProjectA, fxEmpAn, fxPayrate, d(2026, 8, 10, loc), fxUser,
		fxTsBase+1, fxProjectA, fxEmpAn, fxPayrate, d(2026, 8, 20, loc), fxUser,
		fxTsBase+2, fxProjectA, fxEmpAn, fxPayrate, d(2026, 8, 21, loc), fxUser,
		fxTsBase+3, fxProjectA, fxEmpBinh, fxPayrate, d(2026, 8, 11, loc), fxUser,
		fxTsBase+4, fxProjectA, fxEmpGiang, fxPayrate, d(2026, 8, 12, loc), fxUser,
		fxTsBase+5, fxProjectA, fxEmpLan, fxPayrate, to, fxUser,
		fxTsBase+6, fxProjectA, fxEmpPhuc, fxPayrate, d(2026, 8, 1, loc), fxUser,
		fxTsBase+7, fxProjectA, fxEmpPhuc, fxPayrate, d(2026, 8, 2, loc), fxUser,
		fxTsBase+8, fxProjectB, fxEmpPhuc, fxPayrate, d(2026, 8, 3, loc), fxUser)

	repo := &EmployeeRepository{
		BaseRepository: &BaseRepository{DB: db},
	}
	ctx := context.Background()

	rows, err := repo.GetPaidWithoutMobile(ctx, from, to)
	require.NoError(t, err)

	// Included: An (salary), Lan (both boundary dates), Phúc (NULL mobile).
	// Excluded: Bình (has mobile), Giang (soft-deleted), plus An's un-paid
	// row.
	byID := make(map[uint]*domain.EmployeePaidActivity, len(rows))
	for _, row := range rows {
		byID[row.EmployeeID] = row
	}

	an := byID[uint(fxEmpAn)]
	require.NotNil(t, an)
	require.Equal(t, int64(3500000), an.SalaryTotal)
	require.Equal(t, int64(2), an.SalaryCount)
	require.True(t, an.LastSalaryPaidAt.Equal(time.Date(2026, 8, 20, 0, 0, 0, 0, loc)))
	require.True(t, an.LastActivityAt.Equal(time.Date(2026, 8, 20, 0, 0, 0, 0, loc)))
	require.True(t, an.IsWorking, "active assignment => working")
	require.Equal(t, "Dự án Alpha", an.Projects)

	lan := byID[uint(fxEmpLan)]
	require.NotNil(t, lan, "paid_at == from and paid_at == to are both inclusive")
	require.Equal(t, int64(800000), lan.SalaryTotal)
	require.True(t, lan.LastActivityAt.Equal(to))

	phuc := byID[uint(fxEmpPhuc)]
	require.NotNil(t, phuc, "NULL mobile counts as missing")
	require.Equal(t, int64(1200000), phuc.SalaryTotal)
	require.Equal(t, "Dự án Alpha,Dự án Beta", phuc.Projects)

	// Excluded fixture employees must never appear.
	for _, excluded := range []uint{uint(fxEmpBinh), uint(fxEmpGiang)} {
		require.NotContains(t, byID, excluded)
	}
}

// d builds a fixture datetime in the given location.
func d(year int, month time.Month, day int, loc *time.Location) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}
