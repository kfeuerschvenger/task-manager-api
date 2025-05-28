package errors

// AuthError represents an authentication error with a message.
// It implements the error interface.

type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

func NewAuthError(msg string) error {
	return &AuthError{Message: msg}
}

type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

func NewConflictError(msg string) error {
	return &ConflictError{Message: msg}
}
