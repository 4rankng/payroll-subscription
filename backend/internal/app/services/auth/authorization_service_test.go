package auth

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type routeDef struct {
	method string
	path   string
}

// resolveConfigPaths returns absolute paths to the casbin model + policy
// files in repo configs/, regardless of the package dir the test runs in.
func resolveRepoRoot(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve caller for config path")
	}
	return filepath.Join(filepath.Dir(here), "..", "..", "..", "..")
}

func resolveConfigPaths(t *testing.T) (model, policy string) {
	t.Helper()
	repoRoot := resolveRepoRoot(t)
	return filepath.Join(repoRoot, "configs", "casbin_model.conf"),
		filepath.Join(repoRoot, "configs", "casbin_policy.csv")
}

func newTestAuthorizationService(t *testing.T) *AuthorizationService {
	t.Helper()
	model, policy := resolveConfigPaths(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc, err := NewAuthorizationService(model, policy, nil, logger)
	if err != nil {
		t.Fatalf("NewAuthorizationService: %v", err)
	}
	return svc
}

func joinRoute(parts ...string) string {
	var out []string
	for _, part := range parts {
		part = strings.Trim(part, "/")
		if part != "" {
			out = append(out, part)
		}
	}
	return "/" + strings.Join(out, "/")
}

func sampleRoutePath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			segments[i] = "1"
		}
	}
	return strings.Join(segments, "/")
}

func isEmployeeSelfServicePath(path string, method string) bool {
	if path == "/api/v1/me" {
		// GET reads the profile; PUT updates it.
		return method == "GET" || method == "PUT"
	}
	if path == "/api/v1/me/password" {
		// Self-service own-password change — allowed for nhanvien (PAYR-48).
		return method == "PUT"
	}
	return method == "GET" && strings.HasPrefix(path, "/api/v1/me/")
}

func discoverEmployeeSelfServiceRoutes(t *testing.T) []routeDef {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(resolveRepoRoot(t), "internal", "app", "bootstrap", "routes*.go"))
	if err != nil {
		t.Fatalf("glob bootstrap route files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no bootstrap route files found")
	}

	groupRe := regexp.MustCompile(`^\s*(\w+)\s*:=\s*(\w+)\.Group\("([^"]*)"\)`)
	routeRe := regexp.MustCompile(`^\s*(\w+)\.(GET|POST|PUT|PATCH|DELETE)\("([^"]*)"`)

	seen := make(map[string]bool)
	var routes []routeDef
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read route file %s: %v", file, err)
		}

		groups := map[string]string{
			"protected": "",
			"v1":        "",
		}
		for _, line := range strings.Split(string(content), "\n") {
			if match := groupRe.FindStringSubmatch(line); match != nil {
				name, parent, suffix := match[1], match[2], match[3]
				if prefix, ok := groups[parent]; ok {
					groups[name] = joinRoute(prefix, suffix)
				}
				continue
			}

			match := routeRe.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			groupName, method, routePath := match[1], match[2], match[3]
			prefix, ok := groups[groupName]
			if !ok {
				continue
			}

			fullPath := sampleRoutePath(joinRoute("/api/v1", prefix, routePath))
			if !isEmployeeSelfServicePath(fullPath, method) {
				continue
			}

			key := method + " " + fullPath
			if seen[key] {
				continue
			}
			seen[key] = true
			routes = append(routes, routeDef{method: method, path: fullPath})
		}
	}

	return routes
}

// TestExistingRoles_NotRegressed spot-checks that the PAYSUB role policy
// did not accidentally widen or narrow admin/quanly/giamdoc/nhanvien access.
func TestExistingRoles_NotRegressed(t *testing.T) {
	svc := newTestAuthorizationService(t)

	// Admin still has wildcard on /api/*
	assert.True(t, svc.CanAccess("admin", "/api/v1/users", "POST"))
	assert.True(t, svc.CanAccess("admin", "/api/v1/settings/1", "DELETE"))

	// Quanly keeps daily operations, denied on admin-only surfaces
	assert.True(t, svc.CanAccess("quanly", "/api/v1/employees", "GET"))
	assert.True(t, svc.CanAccess("quanly", "/api/v1/projects/1", "PUT"))
	assert.False(t, svc.CanAccess("quanly", "/api/v1/settings/1", "PUT"))
	assert.False(t, svc.CanAccess("quanly", "/api/v1/audit/logs", "GET"))
	assert.False(t, svc.CanAccess("quanly", "/api/v1/timesheets/bulk-approve", "POST"))

	// Giamdoc reads timesheets and drives the billing-report pipeline, denied on writes
	assert.True(t, svc.CanAccess("giamdoc", "/api/v1/timesheets", "GET"))
	assert.True(t, svc.CanAccess("giamdoc", "/api/v1/billing/reports/generate", "POST"))
	assert.False(t, svc.CanAccess("giamdoc", "/api/v1/timesheets", "POST"))
	assert.False(t, svc.CanAccess("giamdoc", "/api/v1/employees", "POST"))

	// Nhanvien keeps self-service, denied on operational surfaces
	assert.True(t, svc.CanAccess("nhanvien", "/api/v1/me", "GET"))
	assert.True(t, svc.CanAccess("nhanvien", "/api/v1/me", "PUT"))
	assert.False(t, svc.CanAccess("nhanvien", "/api/v1/employees", "GET"))
	assert.False(t, svc.CanAccess("nhanvien", "/api/v1/timesheets", "POST"))
}

// TestNhanvienRole_AllSelfServiceRoutesAreAllowed audits Gin route registration
// against Casbin policy for nhanvien-owned surfaces. It catches the class of
// regression where a new self-service route is added but the nhanvien policy is
// not updated, causing a 403 before the handler runs.
func TestNhanvienRole_AllSelfServiceRoutesAreAllowed(t *testing.T) {
	svc := newTestAuthorizationService(t)
	routes := discoverEmployeeSelfServiceRoutes(t)
	if len(routes) == 0 {
		t.Fatal("no nhanvien self-service routes discovered")
	}

	for _, route := range routes {
		assert.True(t,
			svc.CanAccess("nhanvien", route.path, route.method),
			"nhanvien should be allowed %s %s", route.method, route.path,
		)
	}
}

// TestQuanlyRole_EditRequests asserts quanly can list their own edit
// requests (GET /edit-requests, scoped by the handler to RequestedBy) but
// cannot read a specific request by id (the detail handler has no ownership
// check, so allowing it would be an IDOR hole) or approve/reject (admin-only).
// Regression test for the blanket deny that e7230cb added on the whole resource.
func TestQuanlyRole_EditRequests(t *testing.T) {
	svc := newTestAuthorizationService(t)

	// Quanly can list their own edit requests (handler scopes by RequestedBy)
	assert.True(t, svc.CanAccess("quanly", "/api/v1/timesheets/edit-requests", "GET"),
		"quanly should be allowed to list edit requests (handler scopes to own)")

	// Quanly must NOT reach detail/approve/reject (admin-only)
	quanlyDeny := []struct {
		path   string
		method string
	}{
		{"/api/v1/timesheets/edit-requests/1", "GET"},
		{"/api/v1/timesheets/edit-requests/1/approve", "PUT"},
		{"/api/v1/timesheets/edit-requests/1/reject", "PUT"},
	}
	for _, tc := range quanlyDeny {
		assert.False(t, svc.CanAccess("quanly", tc.path, tc.method),
			"quanly should be DENIED %s %s", tc.method, tc.path)
	}

	// Admin retains full access to the edit-request resource (no regression)
	adminAllow := []struct {
		path   string
		method string
	}{
		{"/api/v1/timesheets/edit-requests", "GET"},
		{"/api/v1/timesheets/edit-requests/1", "GET"},
		{"/api/v1/timesheets/edit-requests/1/approve", "PUT"},
		{"/api/v1/timesheets/edit-requests/1/reject", "PUT"},
	}
	for _, tc := range adminAllow {
		assert.True(t, svc.CanAccess("admin", tc.path, tc.method),
			"admin should be allowed %s %s", tc.method, tc.path)
	}
}

// TestQuanlyRole_DeleteTimesheet asserts quanly can delete a single
// timesheet by id (DELETE /timesheets/:id). The DeleteTimesheet handler
// enforces that only pending_approval or rejected entries are deletable
// (IsPaid() is rejected first), so the policy must permit DELETE — otherwise
// the quanly UI hits 403 before the handler's own status guard can run.
// Admin-only bulk actions must remain denied.
func TestQuanlyRole_DeleteTimesheet(t *testing.T) {
	svc := newTestAuthorizationService(t)

	// Quanly can delete a timesheet by id (handler guards: pending/rejected + not paid)
	assert.True(t, svc.CanAccess("quanly", "/api/v1/timesheets/1", "DELETE"),
		"quanly should be allowed DELETE /timesheets/:id (handler enforces status)")

	// The :id allow must NOT weaken admin-only actions — explicit deny wins (deny-overrides)
	adminOnly := []struct {
		path   string
		method string
	}{
		{"/api/v1/timesheets/bulk-approve", "POST"},
		{"/api/v1/timesheets/bulk-reject", "POST"},
		{"/api/v1/timesheets/reject-unpaid", "POST"},
		{"/api/v1/timesheets/bulk-reset", "POST"},
		{"/api/v1/timesheets/approve-all", "POST"},
		{"/api/v1/timesheets/edit-requests/1", "DELETE"},
	}
	for _, tc := range adminOnly {
		assert.False(t, svc.CanAccess("quanly", tc.path, tc.method),
			"quanly should be DENIED %s %s (admin-only, deny-overrides)", tc.method, tc.path)
	}

	// Admin retains full delete access (no regression)
	assert.True(t, svc.CanAccess("admin", "/api/v1/timesheets/1", "DELETE"),
		"admin should be allowed DELETE /timesheets/:id")
}

// TestQuanlyRole_GetTimesheet asserts quanly can read a single timesheet
// by id (GET /timesheets/:id). The list rule (GET /timesheets) does not cover :id
// under keyMatch2, so without an explicit allow the quanly UI (edit-modal detail
// fetch, delete/update pre-fetch) hits 403 — which aborts the operation before the
// React Query cache invalidation runs, leaving stale data on screen. Same gap
// class as DELETE /timesheets/:id. Admin-only actions must remain denied.
func TestQuanlyRole_GetTimesheet(t *testing.T) {
	svc := newTestAuthorizationService(t)

	// Quanly can read a timesheet by id (read-only)
	assert.True(t, svc.CanAccess("quanly", "/api/v1/timesheets/1", "GET"),
		"quanly should be allowed GET /timesheets/:id")

	// The :id allow must NOT weaken admin-only actions — explicit deny wins (deny-overrides).
	// bulk-* and edit-requests/* are denied with "*" (all methods), which covers GET too.
	adminOnlyGET := []string{
		"/api/v1/timesheets/bulk-approve",
		"/api/v1/timesheets/bulk-reject",
		"/api/v1/timesheets/reject-unpaid",
		"/api/v1/timesheets/bulk-reset",
		"/api/v1/timesheets/approve-all",
		"/api/v1/timesheets/edit-requests/1",
	}
	for _, path := range adminOnlyGET {
		assert.False(t, svc.CanAccess("quanly", path, "GET"),
			"quanly should be DENIED GET %s (admin-only, deny-overrides)", path)
	}

	// Admin retains full read access (no regression)
	assert.True(t, svc.CanAccess("admin", "/api/v1/timesheets/1", "GET"),
		"admin should be allowed GET /timesheets/:id")
}
