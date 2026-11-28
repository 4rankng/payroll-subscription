package httputils

const (
	// DefaultIDBase is the base used for parsing IDs (10 = decimal)
	DefaultIDBase = 10
	// IDBitSize is the bit size for parsing IDs (64 = uint64, matching the
	// BIGINT UNSIGNED columns and the test fixture ID band beyond uint32)
	IDBitSize = 64
)

// ParseIDParam parses a uint ID parameter from the gin context
// Returns error if the parameter is missing or invalid

// ParseQueryID parses a uint ID from query string
// Returns 0, nil if the parameter is missing (optional), error if invalid

// Optional parameter

// ParseRequiredQueryID parses a uint ID from query string (required)
// Returns error if the parameter is missing or invalid

// ParseIDFromString parses a uint from a string
// Returns error if the string is empty or invalid
