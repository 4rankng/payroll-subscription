package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidationErrorWithCode(t *testing.T) {
	code := "TEST_CODE"
	message := "Test validation error message"

	err := NewValidationErrorWithCode(code, message)

	assert.NotNil(t, err)
	assert.Equal(t, "VALIDATION_ERROR", err.Type)
	assert.Equal(t, code, err.Code)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, ErrValidation, err.Cause)
}
