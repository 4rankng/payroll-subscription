package validation

import (
	"errors"
)

var (
	// ErrInvalidEmail is returned when an email address is invalid
	ErrInvalidEmail = errors.New("invalid email address")
	// ErrInvalidID is returned when an ID is invalid
	ErrInvalidID = errors.New("invalid ID")
	// ErrEmptyValue is returned when a required value is empty
	ErrEmptyValue = errors.New("value cannot be empty")
)

// ValidateEmail validates an email address format.
// Returns nil if valid, error otherwise.

// IsValidEmail checks if an email address is valid.
// Returns true if valid, false otherwise.

// ParseUint parses a string to uint with validation.
// Returns error if the string is empty, not a valid number, or negative.

// ParseUintOrDefault parses a string to uint, returning a default value if parsing fails.

// ParseIDs parses a comma-separated string of IDs into a slice of uints.
// Invalid IDs are skipped. Returns empty slice if input is empty or all IDs are invalid.

// ValidateRequired checks if a string value is not empty.
// Returns error if empty, nil otherwise.

// ValidateMinLength checks if a string has at least the specified minimum length.

// ValidateMaxLength checks if a string does not exceed the specified maximum length.

// ValidateRange checks if a numeric value is within the specified range (inclusive).

// IsPositive checks if a number is greater than zero.

// IsNonNegative checks if a number is greater than or equal to zero.
