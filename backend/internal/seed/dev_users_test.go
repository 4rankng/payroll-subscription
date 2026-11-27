package seed

import (
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
)

// TestDevUsersComposition pins the local-dev login roster: exactly the five
// deployment roles (PAYSUB D28), one account each, shared dev password.
func TestDevUsersComposition(t *testing.T) {
	want := []struct {
		username string
		role     domain.UserRole
	}{
		{"admin", domain.RoleAdmin},
		{"quanly", domain.RoleQuanly},
		{"ketoan", domain.RoleKetoan},
		{"giamdoc", domain.RoleGiamdoc},
		{"nhanvien", domain.RoleNhanvien},
	}

	assert.Len(t, devUsers, len(want))
	for i, w := range want {
		assert.Equal(t, w.username, devUsers[i].username, "devUsers[%d] username", i)
		assert.Equal(t, w.role, devUsers[i].role, "devUsers[%d] role", i)
	}
}

// TestDevUserPasswordIsDevOnly pins the shared dev password constant: it must
// stay the documented local-dev value and is never applied in production
// (EnsureDevUsers callers guard on APP_ENV).
func TestDevUserPasswordIsDevOnly(t *testing.T) {
	assert.Equal(t, "Abc123", devUserPassword)
}
