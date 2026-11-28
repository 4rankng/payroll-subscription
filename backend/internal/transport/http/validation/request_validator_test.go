package validation

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewRequestValidator(t *testing.T) {
	v := NewRequestValidator()
	if v == nil {
		t.Error("NewRequestValidator() returned nil")
	}
}

// Set up request with query params

func TestValidateUserContext(t *testing.T) {
	v := NewRequestValidator()

	tests := []struct {
		name       string
		setupCtx   func(*gin.Context)
		wantUserID uint
		wantErr    bool
	}{
		{
			name: "valid user context",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", uint(123))
			},
			wantUserID: 123,
			wantErr:    false,
		},
		{
			name: "missing user_id",
			setupCtx: func(c *gin.Context) {
				// Don't set user_id
			},
			wantUserID: 0,
			wantErr:    true,
		},
		{
			name: "zero user_id",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", uint(0))
			},
			wantUserID: 0,
			wantErr:    true,
		},
		{
			name: "invalid type",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", "not a uint")
			},
			wantUserID: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			got, err := v.ValidateUserContext(c)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.wantUserID {
				t.Errorf("ValidateUserContext() = %v, want %v", got, tt.wantUserID)
			}
		})
	}
}
