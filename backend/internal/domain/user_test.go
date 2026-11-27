package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_ValidateEmail(t *testing.T) {
	u := &User{}

	// Test valid email
	email := "test@example.com"
	u.Email = &email
	err := u.ValidateEmail()
	assert.NoError(t, err)

	// Email is optional, so empty email should be valid
	emptyEmail := ""
	u.Email = &emptyEmail
	err = u.ValidateEmail()
	assert.NoError(t, err)
}

func TestUser_ValidateUsername(t *testing.T) {
	u := &User{}

	// Test valid username
	u.Username = "testuser"
	err := u.ValidateUsername()
	assert.NoError(t, err)

	// Test empty username
	u.Username = ""
	err = u.ValidateUsername()
	assert.Error(t, err)

	// Test username too short
	u.Username = "ab"
	err = u.ValidateUsername()
	assert.Error(t, err)
}

func TestUser_ValidateFullname(t *testing.T) {
	u := &User{}

	// Test valid fullname
	u.Fullname = "John Doe"
	err := u.ValidateFullname()
	assert.NoError(t, err)

	// Test empty fullname
	u.Fullname = ""
	err = u.ValidateFullname()
	assert.Error(t, err)
}

func TestUser_ValidateRole(t *testing.T) {
	u := &User{}

	// Test valid roles
	u.Role = RoleAdmin
	err := u.ValidateRole()
	assert.NoError(t, err)

	u.Role = RoleQuanly
	err = u.ValidateRole()
	assert.NoError(t, err)

	u.Role = RoleGiamdoc
	err = u.ValidateRole()
	assert.NoError(t, err)

	u.Role = RoleNhanvien
	err = u.ValidateRole()
	assert.NoError(t, err)

	u.Role = RoleKetoan
	err = u.ValidateRole()
	assert.NoError(t, err)

	// Test invalid role
	u.Role = "invalid"
	err = u.ValidateRole()
	assert.Error(t, err)
}

func TestUser_IsKetoan(t *testing.T) {
	u := &User{Role: RoleKetoan}
	assert.True(t, u.IsKetoan())

	u.Role = RoleGiamdoc
	assert.False(t, u.IsKetoan())
}

func TestUser_IsQuanly(t *testing.T) {
	u := &User{Role: RoleQuanly}
	assert.True(t, u.IsQuanly())

	u.Role = RoleAdmin
	assert.False(t, u.IsQuanly())
}

func TestUser_IsAdmin(t *testing.T) {
	u := &User{
		Role: RoleAdmin,
	}

	assert.True(t, u.IsAdmin())

	u.Role = RoleNhanvien
	assert.False(t, u.IsAdmin())
}

func TestUser_IsGiamdoc(t *testing.T) {
	u := &User{
		Role: RoleGiamdoc,
	}

	assert.True(t, u.IsGiamdoc())

	u.Role = RoleAdmin
	assert.False(t, u.IsGiamdoc())
}

func TestUser_IsNhanvien(t *testing.T) {
	u := &User{
		Role: RoleNhanvien,
	}

	assert.True(t, u.IsNhanvien())

	u.Role = RoleAdmin
	assert.False(t, u.IsNhanvien())
}

func TestUser_GetAuditEntityType(t *testing.T) {
	u := User{}
	entityType := u.GetAuditEntityType()
	assert.Equal(t, "user", entityType)
}

func TestUser_GetAuditEntityID(t *testing.T) {
	u := User{ID: 456}
	entityID := u.GetAuditEntityID()
	assert.Equal(t, uint(456), entityID)
}
