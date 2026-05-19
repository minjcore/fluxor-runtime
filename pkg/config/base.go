package config

// ============================================================================
// Interfaces
// ============================================================================

// Loader loads configuration from various sources
type Loader interface {
	Load(path string, target interface{}) error
}

// Validator validates configuration
type Validator interface {
	Validate(config interface{}) error
}

// ValidatorFunc is a function that validates configuration
type ValidatorFunc func(config interface{}) error

func (f ValidatorFunc) Validate(config interface{}) error {
	return f(config)
}
