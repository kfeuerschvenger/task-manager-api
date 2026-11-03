package errors

import "fmt"

// Provides custom error types and functions for handling common error scenarios in the application.
// Each category has its own type so callers can classify errors with errors.As instead of
// comparing message strings. Messages stay human readable and are safe to show to clients,
// except for InternalServerError, whose message must never be sent over the wire.

// NotFoundError is returned when a requested entity does not exist.
type NotFoundError struct {
	Entity string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", e.Entity)
}

// ForbiddenError is returned when the caller is not allowed to perform an action on an entity.
type ForbiddenError struct {
	Action string
	Entity string
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("unauthorized to %s %s", e.Action, e.Entity)
}

// ErrInvalidID reports a malformed identifier. It is a ValidationError.
func ErrInvalidID(entity string) error {
	return NewValidationError(fmt.Sprintf("invalid %s ID", entity))
}

// ErrNotFound reports that an entity does not exist. It is a *NotFoundError.
func ErrNotFound(entity string) error {
	return &NotFoundError{Entity: entity}
}

// ErrUnauthorizedAction reports that the caller may not perform an action. It is a *ForbiddenError.
func ErrUnauthorizedAction(action string, entity string) error {
	return &ForbiddenError{Action: action, Entity: entity}
}

// ErrInvalidField reports an invalid field value. It is a ValidationError.
func ErrInvalidField(field string) error {
	return NewValidationError(fmt.Sprintf("invalid %s", field))
}

// InternalServerError wraps an unexpected failure. Its message is for logs only.
// Cause keeps the underlying error available to errors.Is and errors.As.
type InternalServerError struct {
	Message string
	Cause   error
}

func (e *InternalServerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *InternalServerError) Unwrap() error {
	return e.Cause
}

func NewInternalServerError(msg string) error {
	return &InternalServerError{Message: msg}
}

// WrapInternalServerError wraps an underlying error as an InternalServerError.
func WrapInternalServerError(msg string, cause error) error {
	return &InternalServerError{Message: msg, Cause: cause}
}
