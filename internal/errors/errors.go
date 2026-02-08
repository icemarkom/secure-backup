package errors

import "fmt"

// UserError represents an error with user-friendly messaging and actionable hints
type UserError struct {
	Message string // User-friendly error message
	Hint    string // Actionable hint to resolve the issue
	Cause   error  // Underlying error (optional)
}

// Error implements the error interface
func (e *UserError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s\nHint: %s", e.Message, e.Hint)
	}
	return e.Message
}

// Unwrap returns the underlying error for error chain support
func (e *UserError) Unwrap() error {
	return e.Cause
}

// New creates a new UserError with a message and hint
func New(message, hint string) *UserError {
	return &UserError{
		Message: message,
		Hint:    hint,
	}
}

// Wrap wraps an existing error with a user-friendly message and hint
func Wrap(err error, message, hint string) *UserError {
	return &UserError{
		Message: message,
		Hint:    hint,
		Cause:   err,
	}
}

// MissingFile creates an error for a missing file with a helpful suggestion
func MissingFile(path, suggestion string) *UserError {
	return &UserError{
		Message: fmt.Sprintf("File not found: %s", path),
		Hint:    suggestion,
	}
}

// InvalidConfig creates an error for invalid configuration
func InvalidConfig(field, issue, fix string) *UserError {
	return &UserError{
		Message: fmt.Sprintf("Invalid configuration for %s: %s", field, issue),
		Hint:    fix,
	}
}

// MissingRequired creates an error for a missing required parameter
func MissingRequired(param, suggestion string) *UserError {
	return &UserError{
		Message: fmt.Sprintf("Required parameter missing: %s", param),
		Hint:    suggestion,
	}
}
