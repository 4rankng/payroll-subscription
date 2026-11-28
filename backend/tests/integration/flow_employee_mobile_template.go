package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

const flowMobileTemplate = "EmployeeMobileTemplateExport"

// runEmployeeMobileTemplateTests proves the "NV chưa có SĐT" template export —
// the source half of the import-mobiles loop. Admin and quanly must get a real
// xlsx workbook whose header row carries the CCCD/Mobile columns the backfill
// parser needs; unprivileged roles are rejected before any data leaves.
func runEmployeeMobileTemplateTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Employee Mobile Template Export")

	admin := client.WithToken(data.AdminToken)
	reporter.RunTest(flowMobileTemplate, "Template export returns a valid workbook for admin", func() error {
		body, headers, status, err := admin.DownloadGet("/api/v1/employees/export-missing-mobile-template")
		if err != nil {
			return fmt.Errorf("request: %w", err)
		}
		if status != 200 {
			return fmt.Errorf("expected 200, got %d", status)
		}
		if cd := headers.Get("Content-Disposition"); !strings.Contains(cd, ".xlsx") {
			return fmt.Errorf("expected an xlsx attachment, got Content-Disposition %q", cd)
		}
		if !bytes.HasPrefix(body, []byte("PK\x03\x04")) {
			return fmt.Errorf("body is not an xlsx (zip) file")
		}
		f, err := excelize.OpenReader(bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("open workbook: %w", err)
		}
		defer func() { _ = f.Close() }()
		rows, err := f.GetRows(f.GetSheetList()[0])
		if err != nil {
			return fmt.Errorf("read rows: %w", err)
		}
		if len(rows) == 0 {
			return fmt.Errorf("workbook has no header row")
		}
		header := strings.Join(rows[0], "|")
		if !strings.Contains(header, "CCCD") || !strings.Contains(header, "Mobile") {
			return fmt.Errorf("header row %q must carry CCCD and Mobile", header)
		}
		return nil
	})

	if len(data.Partners) > 0 {
		quanly := client.WithToken(data.Partners[0].Token)
		reporter.RunTest(flowMobileTemplate, "Template export passes RBAC for quanly", func() error {
			_, _, status, err := quanly.DownloadGet("/api/v1/employees/export-missing-mobile-template")
			if err != nil {
				return fmt.Errorf("request: %w", err)
			}
			if status != 200 {
				return fmt.Errorf("quanly must pass the role gate, got %d", status)
			}
			return nil
		})
	}

	if data.EmployeeTokenForAdv != "" {
		nhanvien := client.WithToken(data.EmployeeTokenForAdv)
		reporter.RunTest(flowMobileTemplate, "Template export rejected for nhanvien", func() error {
			_, status, err := nhanvien.GetExpectError("/api/v1/employees/export-missing-mobile-template")
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
