package errors

// ValidationError represents an error that occurs when validation fails.
// It implements the error interface and provides a message describing the validation issue.

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewValidationError(msg string) error {
	return &ValidationError{Message: msg}
}