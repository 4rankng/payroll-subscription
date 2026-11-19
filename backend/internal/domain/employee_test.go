package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmployee_ValidateFullname(t *testing.T) {
	e := &Employee{}

	// Test valid fullname
	e.Fullname = "John Doe"
	err := e.ValidateFullname()
	assert.NoError(t, err)

	// Test empty fullname
	e.Fullname = ""
	err = e.ValidateFullname()
	assert.Error(t, err)
}

func TestEmployee_ValidateCCCD(t *testing.T) {
	e := &Employee{}

	// Test valid CCCD
	e.CCCD = "123456789"
	err := e.ValidateCCCD()
	assert.NoError(t, err)

	// Test empty CCCD
	e.CCCD = ""
	err = e.ValidateCCCD()
	assert.Error(t, err)

	// Test invalid characters in CCCD
	e.CCCD = "12345!6789"
	err = e.ValidateCCCD()
	assert.Error(t, err)
}

func TestEmployee_ValidateEmail(t *testing.T) {
	e := &Employee{}

	// Test valid email
	email := "test@example.com"
	e.Email = &email
	err := e.ValidateEmail()
	assert.NoError(t, err)

	// Test invalid email
	invalidEmail := "invalid-email"
	e.Email = &invalidEmail
	err = e.ValidateEmail()
	assert.Error(t, err)
}

func TestEmployee_ValidateMobile(t *testing.T) {
	e := &Employee{}

	// Test valid mobile
	e.Mobile = "0123456789"
	err := e.ValidateMobile()
	assert.NoError(t, err)

	// Test invalid mobile characters
	e.Mobile = "123abc456"
	err = e.ValidateMobile()
	assert.Error(t, err)
}

func TestEmployee_ValidateMobile_RejectsCCCD(t *testing.T) {
	e := &Employee{CCCD: "001179010398", Mobile: "001179010398"}

	err := e.ValidateMobile()
	assert.Error(t, err, "a 12-digit CCCD in the mobile field must be rejected")

	// A 12-digit value is a CCCD even when it belongs to someone else.
	e.Mobile = "031204007929"
	assert.Error(t, e.ValidateMobile(), "any 12-digit value must be rejected as a mobile")

	// Ten- and eleven-digit numbers stay valid: they are real phone shapes and
	// the rule must not turn into general mobile-format validation.
	e.Mobile = "0357210887"
	assert.NoError(t, e.ValidateMobile())

	e.Mobile = "84357210887"
	assert.NoError(t, e.ValidateMobile())
}

func TestEmployee_IsValid(t *testing.T) {
	email := "test@example.com"

	// Test valid employee
	e := &Employee{
		Fullname: "John Doe",
		CCCD:     "123456789",
		Email:    &email,
		Mobile:   "0123456789",
	}
	err := e.IsValid()
	assert.NoError(t, err)

	// Test invalid fullname
	e2 := &Employee{
		Fullname: "",
		CCCD:     "123456789",
	}
	err = e2.IsValid()
	assert.Error(t, err)

	// Test invalid CCCD
	e3 := &Employee{
		Fullname: "John Doe",
		CCCD:     "",
	}
	err = e3.IsValid()
	assert.Error(t, err)
}

func TestEmployee_FormattedFullname(t *testing.T) {
	tests := []struct {
		name     string
		fullname string
		want     string
	}{
		{
			name:     "lowercase name",
			fullname: "john doe",
			want:     "John Doe",
		},
		{
			name:     "uppercase name",
			fullname: "JOHN DOE",
			want:     "John Doe",
		},
		{
			name:     "mixed case name",
			fullname: "JoHn DoE",
			want:     "John Doe",
		},
		{
			name:     "vietnamese name",
			fullname: "nguyễn văn a",
			want:     "Nguyễn Văn A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Employee{Fullname: tt.fullname}
			got := e.FormattedFullname()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEmployee_GetAuditEntityType(t *testing.T) {
	e := &Employee{}
	entityType := e.GetAuditEntityType()
	assert.Equal(t, "employee", entityType)
}

func TestEmployee_GetAuditEntityID(t *testing.T) {
	e := &Employee{ID: 123}
	entityID := e.GetAuditEntityID()
	assert.Equal(t, uint(123), entityID)
}
