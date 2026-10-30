package dto

// RegisterRequest represents the data required for user registration.
// The password must be at least 8 characters and at most 72 bytes (a bcrypt limit).
type RegisterRequest struct {
	FirstName string `json:"first_name" validate:"required" example:"John"`
	LastName  string `json:"last_name" validate:"required" example:"Doe"`
	Email     string `json:"email" validate:"required" format:"email" example:"user@example.com"`
	Password  string `json:"password" validate:"required" minLength:"8" maxLength:"72" example:"SecurePassword123"`
}

// LoginRequest represents the data required for user login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required" format:"email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"SecurePassword123"`
}
