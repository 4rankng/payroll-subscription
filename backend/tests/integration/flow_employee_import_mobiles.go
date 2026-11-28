package main

import (
	"fmt"
)

const flowImportMobiles = "EmployeeImportMobilesRBAC"

// runEmployeeImportMobilesTests proves the import-mobiles RBAC expansion:
// quanly passes the handler role gate (the bulk-phone-import CTA on the
// partner dashboard banner works), while unprivileged roles are still
// rejected before any parsing.
func runEmployeeImportMobilesTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Employee Import Mobiles RBAC")

	admin := client.WithToken(data.AdminToken)
	reporter.RunTest(flowImportMobiles, "Import mobiles passes RBAC for admin", func() error {
		_, status, err := admin.PostExpectError("/api/v1/employees/import-mobiles", map[string]any{})
		if err != nil {
			return fmt.Errorf("request: %w", err)
		}
		if status == 403 {
			return fmt.Errorf("admin must pass the role gate, got 403")
		}
		// Without a workbook the upload guard answers 400 — that is past RBAC.
		fmt.Printf("    admin status %d (400 = upload guard, expected without a workbook)\n", status)
		return nil
	})

	if len(data.Partners) > 0 {
		quanly := client.WithToken(data.Partners[0].Token)
		reporter.RunTest(flowImportMobiles, "Import mobiles passes RBAC for quanly", func() error {
			_, status, err := quanly.PostExpectError("/api/v1/employees/import-mobiles", map[string]any{})
			if err != nil {
				return fmt.Errorf("request: %w", err)
			}
			if status == 403 {
				return fmt.Errorf("quanly must pass the role gate, got 403")
			}
			fmt.Printf("    quanly status %d (400 = upload guard, expected without a workbook)\n", status)
			return nil
		})
	}

	if data.EmployeeTokenForAdv != "" {
		nhanvien := client.WithToken(data.EmployeeTokenForAdv)
		reporter.RunTest(flowImportMobiles, "Import mobiles rejected for nhanvien", func() error {
			_, status, err := nhanvien.PostExpectError("/api/v1/employees/import-mobiles", map[string]any{})
			if err != nil {
				return fmt.Errorf("request: %w", err)
			}
			if status != 403 {
				return fmt.Errorf("expected 403 for nhanvien, got %d", status)
			}
			return nil
		})
	}
}
