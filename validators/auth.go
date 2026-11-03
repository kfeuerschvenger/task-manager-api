package validators

import (
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/kfeuerschvenger/task-manager-api/dto"
	"github.com/kfeuerschvenger/task-manager-api/errors"
	"github.com/kfeuerschvenger/task-manager-api/utils"
)

const (
	// MinPasswordChars is the minimum password length, counted in characters (runes).
	MinPasswordChars = 8
	// MaxPasswordBytes is the maximum password length in bytes; bcrypt rejects anything longer.
	MaxPasswordBytes = 72
)

// ValidateRegisterInput checks the validity of the registration input.
// It ensures the email format is correct, first and last names are at least 2 characters,
// and the password is at least 8 characters and at most 72 bytes long.
// If any validation fails, it returns an appropriate error.
func ValidateRegisterInput(req dto.RegisterRequest) error {
	req.Email = utils.CleanEmail(req.Email)

	if _, err := mail.ParseAddress(req.Email); err != nil {
		return errors.NewValidationError("invalid email format")
	}

	if len(strings.TrimSpace(req.FirstName)) < 2 {
		return errors.NewValidationError("first name must be at least 2 characters")
	}

	if len(strings.TrimSpace(req.LastName)) < 2 {
		return errors.NewValidationError("last name must be at least 2 characters")
	}

	return ValidatePassword(req.Password)
}

// ValidatePassword enforces the password policy shared by registration and the demo seed:
// at least 8 characters and at most 72 bytes, the limit bcrypt accepts.
func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < MinPasswordChars {
		return errors.NewValidationError("password must be at least 8 characters")
	}

	if len(password) > MaxPasswordBytes {
		return errors.NewValidationError("password must be at most 72 bytes")
	}

	return nil
}

// ValidateLoginInput checks the validity of the login input.
// It ensures the email format is correct and the password is not empty.
// If any validation fails, it returns an appropriate error.
func ValidateLoginInput(req dto.LoginRequest) error {
	req.Email = utils.CleanEmail(req.Email)

	if _, err := mail.ParseAddress(req.Email); err != nil {
		return errors.NewValidationError("invalid email format")
	}

	if len(req.Password) == 0 {
		return errors.NewValidationError("password cannot be empty")
	}

	return nil
}
