package common

// ExtractDB safely extracts a *gorm.DB from an interface{}.
// Returns an error if tx is nil or not a *gorm.DB.
