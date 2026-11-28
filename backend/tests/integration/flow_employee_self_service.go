package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const flowEmpSelfService = "EmployeeSelfService"

func runEmployeeSelfServiceTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Employee Self-Service")

	runProfileProvisioningTests(client, data, reporter, cfg)

	if data.EmployeeTokenForAdv == "" || data.EmployeeForAdvance == nil {
		reporter.Skip(flowEmpSelfService, "All tests", "no employee user account available")
		return
	}

	empClient := client.WithToken(data.EmployeeTokenForAdv)
	empID := data.EmployeeForAdvance.ID

	reporter.RunTest(flowEmpSelfService, "Get my profile (employee)", func() error {
		var resp interface{}
		if _, err := empClient.GetInto("/api/v1/me", &resp); err != nil {
			return fmt.Errorf("get profile: %w", err)
		}
		fmt.Printf("    Employee profile fetched for ID %d\n", empID)
		return nil
	})

	reporter.RunTest(flowEmpSelfService, "Get my timesheets", func() error {
		var resp interface{}
		if _, err := empClient.GetInto("/api/v1/me/timesheet?pageSize=10", &resp); err != nil {
			return fmt.Errorf("get timesheets: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowEmpSelfService, "Get my summary", func() error {
		var resp interface{}
		if _, err := empClient.GetInto("/api/v1/me/summary", &resp); err != nil {
			return fmt.Errorf("get summary: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowEmpSelfService, "Employee cannot access administrative payroll history", func() error {
		_, status, err := empClient.GetExpectError(fmt.Sprintf("/api/v1/employees/%d/payroll?pageSize=10", empID))
		if err != nil {
			return fmt.Errorf("get administrative payroll history: %w", err)
		}
		return AssertEqual("status", http.StatusForbidden, status)
	})
}

// runProfileProvisioningTests (PAYSUB-28): a nhanvien account must always
// resolve to an employee profile. Two failure shapes are pinned: a converted
// account left WITHOUT an employee row heals on the first self-service call,
// and a role conversion to nhanvien provisions the profile at the write path.
func runProfileProvisioningTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()
	password := "TestPass123!"

	cleanup := func(userID uint) {
		_ = execPayrollSQL(fmt.Sprintf("DELETE FROM employees WHERE user_id = %d", userID))
		_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/users/%d", userID))
	}

	// Shape 1: residue — converted to nhanvien without provisioning.
	healName := prefix + "_heal"
	var healed UserResponse
	if _, err := admin.PostInto("/api/v1/users", CreateUserRequest{
		Username: healName, Password: password, Fullname: "ITest Heal Profile", Role: "quanly",
	}, &healed); err != nil {
		reporter.Record(TestResult{Flow: flowEmpSelfService, Name: "create heal fixture", Status: StatusFail, Error: fmt.Errorf("create user: %w", err).Error()})
		return
	}
	defer cleanup(healed.ID)
	// Flip through SQL so no provisioning event fires — the exact residue of
	// legacy accounts converted to nhanvien before provisioning existed.
	if err := execPayrollSQL(fmt.Sprintf("UPDATE users SET role = 'nhanvien' WHERE id = %d", healed.ID)); err != nil {
		reporter.Record(TestResult{Flow: flowEmpSelfService, Name: "create heal fixture", Status: StatusFail, Error: fmt.Errorf("flip role: %w", err).Error()})
		return
	}

	healClient := NewAPIClient(client.BaseURL)
	if _, err := healClient.Login(healName, password); err != nil {
		reporter.Record(TestResult{Flow: flowEmpSelfService, Name: "login heal fixture", Status: StatusFail, Error: fmt.Errorf("login: %w", err).Error()})
		return
	}

	reporter.RunTest(flowEmpSelfService, "Missing employee profile heals on first self-service call", func() error {
		var me struct {
			Fullname string `json:"fullname"`
		}
		if _, err := healClient.GetInto("/api/v1/me", &me); err != nil {
			return fmt.Errorf("healed /me: %w", err)
		}
		if !strings.Contains(strings.ToLower(me.Fullname), "heal profile") {
			return fmt.Errorf("provisioned profile name = %q; want the account holder's name", me.Fullname)
		}
		// The provisioned row carries an identity placeholder — accounts
		// without a citizen ID must not collide on the unique CCCD key. The
		// /me DTO omits cccd by design, so assert on the stored row.
		storedCCCD, err := queryPayrollScalar(fmt.Sprintf("SELECT cccd FROM employees WHERE user_id = %d", healed.ID))
		if err != nil || storedCCCD == "" {
			return fmt.Errorf("provisioned employee cccd = %q (err=%v); want a placeholder identity", storedCCCD, err)
		}
		var summary interface{}
		if _, err := healClient.GetInto("/api/v1/me/summary", &summary); err != nil {
			return fmt.Errorf("healed /me/summary: %w", err)
		}
		return nil
	})

	// Shape 2: the write path — converting an account provisions immediately.
	convName := prefix + "_conv"
	var converted UserResponse
	if _, err := admin.PostInto("/api/v1/users", CreateUserRequest{
		Username: convName, Password: password, Fullname: "ITest Convert Profile", Role: "quanly",
	}, &converted); err != nil {
		reporter.Record(TestResult{Flow: flowEmpSelfService, Name: "create conversion fixture", Status: StatusFail, Error: fmt.Errorf("create user: %w", err).Error()})
		return
	}
	defer cleanup(converted.ID)

	reporter.RunTest(flowEmpSelfService, "Role conversion to nhanvien provisions the employee profile", func() error {
		if _, err := admin.PutInto(fmt.Sprintf("/api/v1/users/%d", converted.ID), map[string]string{"role": "nhanvien"}, &converted); err != nil {
			return fmt.Errorf("convert role: %w", err)
		}
		// Provisioning rides the event bus — poll briefly for the link.
		for range 20 {
			linked, err := queryPayrollScalar(fmt.Sprintf("SELECT id FROM employees WHERE user_id = %d", converted.ID))
			if err == nil && linked != "" {
				return nil
			}
			time.Sleep(250 * time.Millisecond)
		}
		return fmt.Errorf("no employee profile provisioned within 5s of the role change")
	})
}
