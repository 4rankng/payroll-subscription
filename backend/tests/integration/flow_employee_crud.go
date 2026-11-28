package main

import (
	"fmt"
)

const flowEmployee = "EmployeeCRUD"

func runEmployeeCRUDTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Employee CRUD")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()
	testCCCD := prefix + "CCCD"

	var testEmpID uint

	// --- Create ---

	reporter.RunTest(flowEmployee, "Create test employee", func() error {
		body := CreateEmployeeRequest{
			Fullname: prefix + " Employee",
			CCCD:     testCCCD,
			Address:  "123 Test Street",
			Mobile:   uniqueMobile(),
		}
		var resp EmployeeResponse
		if _, err := admin.PostInto("/api/v1/employees", body, &resp); err != nil {
			return fmt.Errorf("create employee: %w", err)
		}
		testEmpID = resp.ID
		fmt.Printf("    Created employee ID %d\n", resp.ID)
		return AssertGreaterThan("id", uint(0), resp.ID)
	})

	var bankedEmpID uint

	defer func() {
		if testEmpID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/employees/%d", testEmpID))
		}
		if bankedEmpID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/employees/%d", bankedEmpID))
		}
	}()

	reporter.RunTest(flowEmployee, "Banks reference list returns seeded branches", func() error {
		var resp BanksResponse
		if _, err := admin.GetInto("/api/v1/banks", &resp); err != nil {
			return fmt.Errorf("list banks: %w", err)
		}
		if err := AssertSliceMinLen("banks", len(resp.Banks), 1); err != nil {
			return err
		}
		fmt.Printf("    Found %d bank branches\n", len(resp.Banks))
		return nil
	})

	reporter.RunTest(flowEmployee, "Create employee with complete bank set", func() error {
		var banks BanksResponse
		if _, err := admin.GetInto("/api/v1/banks", &banks); err != nil {
			return fmt.Errorf("list banks: %w", err)
		}
		if len(banks.Banks) == 0 {
			return fmt.Errorf("no seeded bank branches")
		}
		body := CreateEmployeeRequest{
			Fullname:          prefix + " Banked",
			CCCD:              testCCCD + "B",
			Mobile:            uniqueMobile(),
			BankID:            &banks.Banks[0].ID,
			BankAccountNumber: "0123456789",
			BankAccountName:   prefix + " BANKED",
		}
		var resp EmployeeResponse
		if _, err := admin.PostInto("/api/v1/employees", body, &resp); err != nil {
			return fmt.Errorf("create employee: %w", err)
		}
		bankedEmpID = resp.ID
		if resp.Bank == nil || resp.Bank.ID != banks.Banks[0].ID {
			return fmt.Errorf("create response did not echo bank branch")
		}
		return AssertEqual("bank_account_number", "0123456789", resp.BankAccountNumber)
	})

	reporter.RunTest(flowEmployee, "Bank fields round-trip through detail", func() error {
		if bankedEmpID == 0 {
			return fmt.Errorf("no banked employee ID")
		}
		var resp EmployeeResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d", bankedEmpID), &resp); err != nil {
			return fmt.Errorf("get employee: %w", err)
		}
		if resp.Bank == nil {
			return fmt.Errorf("bank branch missing on detail response")
		}
		if err := AssertGreaterThan("bank_id", uint(0), resp.Bank.ID); err != nil {
			return err
		}
		return AssertEqual("bank_account_name", prefix+" BANKED", resp.BankAccountName)
	})

	reporter.RunTest(flowEmployee, "Partial bank set is rejected with 400", func() error {
		body := CreateEmployeeRequest{
			Fullname:          prefix + " Partial",
			CCCD:              testCCCD + "P",
			Mobile:            uniqueMobile(),
			BankAccountNumber: "9876543210",
		}
		_, statusCode, err := admin.Post("/api/v1/employees", body)
		if statusCode == 400 {
			return nil
		}
		if err != nil {
			return fmt.Errorf("partial bank create: %w", err)
		}
		return fmt.Errorf("expected 400 for partial bank set, got %d", statusCode)
	})

	// --- Read ---

	reporter.RunTest(flowEmployee, "Get employee by ID", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		var resp EmployeeResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d", testEmpID), &resp); err != nil {
			return fmt.Errorf("get employee: %w", err)
		}
		if err := AssertEqual("id", testEmpID, resp.ID); err != nil {
			return err
		}
		return AssertEqual("cccd", testCCCD, resp.CCCD)
	})

	reporter.RunTest(flowEmployee, "Get employee by CCCD", func() error {
		var emp EmployeeResponse
		if _, err := admin.GetInto("/api/v1/employees/cccd/"+testCCCD, &emp); err != nil {
			return fmt.Errorf("get by CCCD: %w", err)
		}
		return AssertEqual("cccd", testCCCD, emp.CCCD)
	})

	reporter.RunTest(flowEmployee, "List employees", func() error {
		var list []EmployeeResponse
		if _, err := admin.GetInto("/api/v1/employees?pageSize=10", &list); err != nil {
			return fmt.Errorf("list employees: %w", err)
		}
		fmt.Printf("    Found %d employees\n", len(list))
		return AssertSliceMinLen("employees", len(list), 1)
	})

	reporter.RunTest(flowEmployee, "Get employee summary (all employees)", func() error {
		var summary EmployeesSummaryResponse
		if _, err := admin.GetInto("/api/v1/employees/summary", &summary); err != nil {
			return fmt.Errorf("employees summary: %w", err)
		}
		fmt.Printf("    Total: %d, Working: %d\n", summary.TotalEmployees, summary.TotalWorkingEmployees)
		return AssertGreaterThan("total_employees", int64(0), summary.TotalEmployees)
	})

	reporter.RunTest(flowEmployee, "Get employee individual summary", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		var summary EmployeeSummaryResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d/summary", testEmpID), &summary); err != nil {
			return fmt.Errorf("employee summary: %w", err)
		}
		fmt.Printf("    Total payroll payments: %d\n", summary.TotalPayrollPayments)
		return nil
	})

	reporter.RunTest(flowEmployee, "Get employee current projects", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		var projects interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d/current-projects", testEmpID), &projects); err != nil {
			return fmt.Errorf("current projects: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowEmployee, "Get unassigned employees", func() error {
		var list []EmployeeResponse
		if _, err := admin.GetInto("/api/v1/employees/unassigned", &list); err != nil {
			return fmt.Errorf("unassigned employees: %w", err)
		}
		fmt.Printf("    Unassigned employees: %d\n", len(list))
		return nil
	})

	reporter.RunTest(flowEmployee, "Export employees (binary)", func() error {
		data, _, statusCode, err := admin.DownloadGet("/api/v1/employees/export")
		if err != nil {
			return fmt.Errorf("export: %w", err)
		}
		if err := AssertGreaterOrEqual("status", 200, statusCode); err != nil {
			return err
		}
		return AssertGreaterThan("file_size", 0, len(data))
	})

	// --- Update ---

	reporter.RunTest(flowEmployee, "Update employee fullname", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		newName := prefix + " Updated Name"
		body := UpdateEmployeeRequest{Fullname: &newName}
		if _, _, err := admin.Put(fmt.Sprintf("/api/v1/employees/%d", testEmpID), body); err != nil {
			return fmt.Errorf("update employee: %w", err)
		}
		var resp EmployeeResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d", testEmpID), &resp); err != nil {
			return fmt.Errorf("get after update: %w", err)
		}
		return AssertContains("fullname", resp.Fullname, "Updated Name")
	})

	reporter.RunTest(flowEmployee, "Update employee mobile", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		mobile := uniqueMobile()
		body := UpdateEmployeeRequest{Mobile: &mobile}
		if _, status, err := admin.Put(fmt.Sprintf("/api/v1/employees/%d", testEmpID), body); err != nil {
			return fmt.Errorf("update mobile: %w", err)
		} else if status >= 400 {
			return fmt.Errorf("update mobile returned HTTP %d", status)
		}
		var resp EmployeeResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d", testEmpID), &resp); err != nil {
			return fmt.Errorf("get after mobile update: %w", err)
		}
		fmt.Printf("    Mobile after update: %q\n", resp.Mobile)
		return AssertEqual("mobile", mobile, resp.Mobile)
	})

	// --- Employee User Access ---

	reporter.RunTest(flowEmployee, "Get employee users (empty)", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		var users []EmployeeUserResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d/users", testEmpID), &users); err != nil {
			return fmt.Errorf("get employee users: %w", err)
		}
		fmt.Printf("    Employee users: %d\n", len(users))
		return nil
	})

	// --- Delete ---

	reporter.RunTest(flowEmployee, "Delete test employee", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/employees/%d", testEmpID)); err != nil {
			return fmt.Errorf("delete employee: %w", err)
		}
		testEmpID = 0
		fmt.Printf("    Test employee deleted\n")
		return nil
	})

	// --- Edge cases ---

	reporter.RunTest(flowEmployee, "Edge: create employee with missing CCCD", func() error {
		body := CreateEmployeeRequest{
			Fullname: prefix + " No CCCD",
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/employees", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowEmployee, "Edge: create employee with duplicate CCCD", func() error {
		if len(data.Employees) == 0 {
			return fmt.Errorf("no existing employees for duplicate test")
		}
		body := CreateEmployeeRequest{
			Fullname: prefix + " Duplicate CCCD",
			CCCD:     data.Employees[0].CCCD,
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/employees", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowEmployee, "Edge: get non-existent employee", func() error {
		_, statusCode, _ := admin.Get(fmt.Sprintf("/api/v1/employees/%d", nonexistentID))
		if statusCode < 400 {
			return fmt.Errorf("expected error for non-existent employee, got HTTP %d", statusCode)
		}
		return nil
	})

	// --- Global pool + self-service claim (partner duplicate prevention) ---
	// Uses a dedicated employee: the shared testEmpID is deleted by the time
	// these run, and claiming mutates the partner's scoped list.

	// Partner flow requires at least one partner account; skip silently otherwise.
	if len(data.Partners) > 0 {
		partner := client.WithToken(data.Partners[0].Token)
		poolCCCD := prefix + "POOL"
		var poolEmpID uint

		reporter.RunTest(flowEmployee, "Pool: create dedicated test employee", func() error {
			body := CreateEmployeeRequest{
				Fullname: prefix + " Pool Employee",
				CCCD:     poolCCCD,
				Mobile:   uniqueMobile(),
			}
			var resp EmployeeResponse
			if _, err := admin.PostInto("/api/v1/employees", body, &resp); err != nil {
				return fmt.Errorf("create pool employee: %w", err)
			}
			poolEmpID = resp.ID
			return AssertGreaterThan("id", uint(0), resp.ID)
		})

		defer func() {
			if poolEmpID != 0 {
				_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/employees/%d", poolEmpID))
			}
		}()

		reporter.RunTest(flowEmployee, "Duplicate check: global CCCD match with masked identifiers", func() error {
			if poolEmpID == 0 {
				return fmt.Errorf("no pool employee ID")
			}
			var resp struct {
				HasDuplicates bool `json:"has_duplicates"`
				Data          []struct {
					ID           uint     `json:"id"`
					Fullname     string   `json:"fullname"`
					CCCDMasked   string   `json:"cccd_masked"`
					MobileMasked string   `json:"mobile_masked"`
					MatchedOn    []string `json:"matched_on"`
				} `json:"data"`
			}
			if _, err := partner.GetInto("/api/v1/employees/duplicate-check?cccd="+poolCCCD, &resp); err != nil {
				return fmt.Errorf("duplicate check: %w", err)
			}
			if !resp.HasDuplicates {
				return fmt.Errorf("expected has_duplicates for CCCD %s", poolCCCD)
			}
			if len(resp.Data) == 0 || resp.Data[0].ID != poolEmpID {
				return fmt.Errorf("expected match on employee %d", poolEmpID)
			}
			masked := resp.Data[0].CCCDMasked
			if masked == "" || masked == poolCCCD {
				return fmt.Errorf("expected masked CCCD, got %q", masked)
			}
			if resp.Data[0].MobileMasked != "" {
				return fmt.Errorf("mobile must not be revealed on CCCD-only match")
			}
			return nil
		})

		reporter.RunTest(flowEmployee, "Duplicate check: no match", func() error {
			var resp struct {
				HasDuplicates bool `json:"has_duplicates"`
			}
			if _, err := partner.GetInto("/api/v1/employees/duplicate-check?cccd="+prefix+"UNKNOWN", &resp); err != nil {
				return fmt.Errorf("duplicate check: %w", err)
			}
			if resp.HasDuplicates {
				return fmt.Errorf("expected no duplicates for unknown CCCD")
			}
			return nil
		})

		reporter.RunTest(flowEmployee, "Duplicate check: requires identifier", func() error {
			_, statusCode, err := partner.Get("/api/v1/employees/duplicate-check")
			if err != nil {
				return fmt.Errorf("get: %w", err)
			}
			return AssertGreaterOrEqual("status", 400, statusCode)
		})

		reporter.RunTest(flowEmployee, "Global pool: partner sees all employees", func() error {
			if poolEmpID == 0 {
				return fmt.Errorf("no pool employee ID")
			}
			var employees []EmployeeResponse
			if _, err := partner.GetInto("/api/v1/employees?scope=global&pageSize=100&sortBy=created_at&sortOrder=desc", &employees); err != nil {
				return fmt.Errorf("global pool list: %w", err)
			}
			found := false
			for _, emp := range employees {
				if emp.ID == poolEmpID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("partner global pool must include admin-created employee %d", poolEmpID)
			}
			return nil
		})

		reporter.RunTest(flowEmployee, "Claim: partner self-service request-access", func() error {
			if poolEmpID == 0 {
				return fmt.Errorf("no pool employee ID")
			}
			var resp APIResponse
			if _, err := partner.PostInto(fmt.Sprintf("/api/v1/employees/%d/request-access", poolEmpID), map[string]string{}, &resp); err != nil {
				return fmt.Errorf("request access: %w", err)
			}

			// Idempotent: second claim succeeds with "already managed" message.
			if _, err := partner.PostInto(fmt.Sprintf("/api/v1/employees/%d/request-access", poolEmpID), map[string]string{}, &resp); err != nil {
				return fmt.Errorf("idempotent request access: %w", err)
			}

			// Scoped default list must now include the claimed employee.
			var scoped []EmployeeResponse
			if _, err := partner.GetInto("/api/v1/employees?pageSize=100&sortBy=created_at&sortOrder=desc", &scoped); err != nil {
				return fmt.Errorf("scoped list: %w", err)
			}
			found := false
			for _, emp := range scoped {
				if emp.ID == poolEmpID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("claimed employee %d must appear in partner scoped list", poolEmpID)
			}

			// Claiming must also authorize the detail endpoint that the partner UI
			// opens after a successful request-access action.
			var detail EmployeeDetailedResponse
			if _, err := partner.GetInto(fmt.Sprintf("/api/v1/employees/%d", poolEmpID), &detail); err != nil {
				return fmt.Errorf("claimed employee detail: %w", err)
			}
			if detail.ID != poolEmpID {
				return fmt.Errorf("claimed employee detail ID = %d, want %d", detail.ID, poolEmpID)
			}
			return nil
		})

		reporter.RunTest(flowEmployee, "Self-release: partner removes claimed employee from scoped list", func() error {
			if poolEmpID == 0 {
				return fmt.Errorf("no pool employee ID")
			}
			var me UserResponse
			if _, err := partner.GetInto("/api/v1/auth/me", &me); err != nil {
				return fmt.Errorf("partner identity: %w", err)
			}

			// The scoped list must annotate the claimed row as claimed — the
			// frontend remove affordance keys off it.
			var scoped []EmployeeResponse
			if _, err := partner.GetInto("/api/v1/employees?pageSize=100&sortBy=created_at&sortOrder=desc", &scoped); err != nil {
				return fmt.Errorf("scoped list: %w", err)
			}
			claimedSeen := false
			for _, emp := range scoped {
				if emp.ID == poolEmpID {
					if emp.IsClaimed == nil || !*emp.IsClaimed {
						return fmt.Errorf("claimed employee %d must carry is_claimed=true on the scoped list", poolEmpID)
					}
					claimedSeen = true
					break
				}
			}
			if !claimedSeen {
				return fmt.Errorf("claimed employee %d missing from scoped list before release", poolEmpID)
			}

			// Release: DELETE own access succeeds and drops the employee from
			// the scoped list while the global pool keeps it.
			if _, _, err := partner.Delete(fmt.Sprintf("/api/v1/employees/%d/users/%d", poolEmpID, me.ID)); err != nil {
				return fmt.Errorf("self-release: %w", err)
			}
			if _, err := partner.GetInto("/api/v1/employees?pageSize=100&sortBy=created_at&sortOrder=desc", &scoped); err != nil {
				return fmt.Errorf("scoped list after release: %w", err)
			}
			for _, emp := range scoped {
				if emp.ID == poolEmpID {
					return fmt.Errorf("released employee %d must leave the scoped list", poolEmpID)
				}
			}

			var pool []EmployeeResponse
			if _, err := partner.GetInto("/api/v1/employees?scope=global&pageSize=100&sortBy=created_at&sortOrder=desc", &pool); err != nil {
				return fmt.Errorf("global pool after release: %w", err)
			}
			stillPooled := false
			for _, emp := range pool {
				if emp.ID == poolEmpID {
					stillPooled = true
					break
				}
			}
			if !stillPooled {
				return fmt.Errorf("released employee %d must stay in the global pool", poolEmpID)
			}

			// No-row contract: releasing again is a silent 200 no-op.
			if _, _, err := partner.Delete(fmt.Sprintf("/api/v1/employees/%d/users/%d", poolEmpID, me.ID)); err != nil {
				return fmt.Errorf("idempotent self-release: %w", err)
			}

			// Re-claim to leave the partner scoped list as earlier tests found it.
			var resp APIResponse
			if _, err := partner.PostInto(fmt.Sprintf("/api/v1/employees/%d/request-access", poolEmpID), map[string]string{}, &resp); err != nil {
				return fmt.Errorf("re-claim after release: %w", err)
			}
			return nil
		})
	}

	reporter.RunTest(flowEmployee, "Edge: update non-existent employee", func() error {
		newName := "Ghost"
		body := UpdateEmployeeRequest{Fullname: &newName}
		_, statusCode, _ := admin.Put(fmt.Sprintf("/api/v1/employees/%d", nonexistentID), body)
		if statusCode < 400 {
			return fmt.Errorf("expected error updating non-existent employee, got HTTP %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowEmployee, "Edge: delete non-existent employee", func() error {
		_, statusCode, _ := admin.Delete(fmt.Sprintf("/api/v1/employees/%d", nonexistentID))
		if statusCode < 400 {
			return fmt.Errorf("expected error deleting non-existent employee, got HTTP %d", statusCode)
		}
		return nil
	})
}
