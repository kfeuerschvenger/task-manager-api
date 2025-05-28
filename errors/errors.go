package errors

import "fmt"

// Provides custom error types and functions for handling common error scenarios in the application.
// It includes errors for invalid IDs, not found entities, unauthorized actions, and invalid fields.
// It also provides a generic internal server error type for unexpected errors.

func ErrInvalidID(entity string) error {
	return fmt.Errorf("invalid %s ID", entity)
}

func ErrNotFound(entity string) error {
	return fmt.Errorf("%s not found", entity)
}

func ErrUnauthorizedAction(action string, entity string) error {
	return fmt.Errorf("unauthorized to %s %s", action, entity)
}

func ErrInvalidField(field string) error {
	return fmt.Errorf("invalid %s", field)
}

type InternalServerError struct {
	Message string
}

func (e *InternalServerError) Error() string {
	return e.Message
}

func NewInternalServerError(msg string) error {
	return &InternalServerError{Message: msg}
}