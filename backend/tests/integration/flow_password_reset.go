package main

import (
	"fmt"
	"strings"
	"time"
)

const flowPasswordReset = "PasswordReset"

// runPasswordResetTests exercises the self-service password-reset endpoints.
//
// Per Red Team C3, the cross-process harness cannot retrieve the emailed token
// (the SandboxProvider lives in the server process). So this integration test
// covers ONLY the no-token contracts:
//   - anti-enumeration: request returns 200 for known AND unknown emails when
//     the limiter bucket is not already exhausted
//   - confirm with a garbage token → 401, or 429 if the shared limiter fires
//   - confirm with a missing token → client error (400 validation, or 429 if
//     the password-reset rate limiter fires before validation)
//
// The single-use, transactional, and session-invalidation guarantees are
// covered by the unit tests in internal/app/services/passwordreset/.
func runPasswordResetTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Password Reset (email)")

	anonymous := NewAPIClient(client.BaseURL)

	// Resolve the admin's real email via /auth/me so the known-email test is real.
	admin := client.WithToken(data.AdminToken)
	var adminProfile struct {
		Email string `json:"email"`
	}
	if _, err := admin.GetInto("/api/v1/auth/me", &adminProfile); err != nil {
		reporter.RunTest(flowPasswordReset, "Setup: resolve admin email", func() error {
			return fmt.Errorf("get admin profile for email: %w", err)
		})
		// Without the email we still run the contract tests that don't need it.
		adminProfile.Email = ""
	}

	// Anti-enumeration: known email returns 200 on a fresh limiter bucket. On
	// repeated integration runs the shared per-email bucket may already be
	// exhausted, in which case middleware returns 429 before the handler.
	if adminProfile.Email != "" {
		reporter.RunTest(flowPasswordReset, "Request reset for known email returns 200 or 429", func() error {
			body := map[string]interface{}{"email": adminProfile.Email}
			_, status, err := anonymous.Post("/api/v1/auth/password-reset/request", body)
			if err != nil {
				return fmt.Errorf("request reset (known): %w", err)
			}
			if status != 200 && status != 429 {
				return fmt.Errorf("known email: status = %d, want 200 or 429", status)
			}
			return nil
		})
	}

	// Anti-enumeration: unknown email returns the SAME 200.
	reporter.RunTest(flowPasswordReset, "Request reset for unknown email returns same 200 (anti-enumeration)", func() error {
		body := map[string]interface{}{"email": "definitely-not-real-" + cfg.UniquePrefix() + "@example.com"}
		_, status, err := anonymous.Post("/api/v1/auth/password-reset/request", body)
		if err != nil {
			return fmt.Errorf("request reset (unknown): %w", err)
		}
		if status != 200 {
			return fmt.Errorf("unknown email: status = %d, want 200 (anti-enumeration)", status)
		}
		return nil
	})

	// Confirm with a garbage token normally returns 401 (invalid/expired). On
	// repeated integration runs, the same middleware can reject earlier with 429
	// because this endpoint shares the password-reset limiter and falls back to
	// an IP bucket when no email is present in the body.
	reporter.RunTest(flowPasswordReset, "Confirm with garbage token returns 401 or 429", func() error {
		body := map[string]interface{}{
			"token":        "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA-fake",
			"new_password": "SomeNewStrong!2026",
		}
		_, status, err := anonymous.Post("/api/v1/auth/password-reset/confirm", body)
		if err != nil {
			return fmt.Errorf("confirm garbage: unexpected transport error: %w", err)
		}
		if status != 401 && status != 429 {
			return fmt.Errorf("garbage token: status = %d, want 401 or 429", status)
		}
		return nil
	})

	// Confirm with a malformed body (missing token) should be rejected as a
	// client error. The password-reset rate limiter can fire before validation
	// on repeated integration runs, so 429 is also acceptable here.
	reporter.RunTest(flowPasswordReset, "Confirm with missing token returns client error", func() error {
		body := map[string]interface{}{"new_password": "SomeNewStrong!2026"}
		_, status, err := anonymous.Post("/api/v1/auth/password-reset/confirm", body)
		if err != nil {
			return fmt.Errorf("confirm missing token: unexpected transport error: %w", err)
		}
		if status != 400 && status != 429 {
			return fmt.Errorf("missing token: status = %d, want 400 or 429", status)
		}
		return nil
	})

	// A reset request for a known email must leave a delivery record in the
	// email history — that record is the evidence that the mail went out (the
	// request itself always returns the same 200). The send is async, so poll.
	reporter.RunTest(flowPasswordReset, "Reset email is recorded in email history", func() error {
		if adminProfile.Email == "" {
			return fmt.Errorf("no admin email to request a reset for")
		}
		body := map[string]interface{}{"email": adminProfile.Email}
		_, status, err := anonymous.Post("/api/v1/auth/password-reset/request", body)
		if err != nil {
			return fmt.Errorf("request reset: %w", err)
		}
		if status == 429 {
			// Shared per-email limiter exhausted by repeated runs — skip.
			fmt.Printf("    rate limited (429) — history assertion skipped\n")
			return nil
		}
		if status != 200 {
			return fmt.Errorf("request reset: status = %d, want 200", status)
		}

		type historyRecipient struct {
			Address string `json:"address"`
		}
		type historyEntry struct {
			Subject    string             `json:"subject"`
			Recipients []historyRecipient `json:"recipients"`
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			var entries []historyEntry
			if _, err := admin.GetInto("/api/v1/email/history?limit=10", &entries); err == nil {
				for _, entry := range entries {
					for _, r := range entry.Recipients {
						if r.Address == adminProfile.Email && strings.Contains(entry.Subject, "Đặt lại mật khẩu") {
							return nil
						}
					}
				}
			}
			time.Sleep(200 * time.Millisecond)
		}
		return fmt.Errorf("no password-reset email recorded in history within 5s")
	})
}
