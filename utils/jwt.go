package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kfeuerschvenger/task-manager-api/errors"
)

// jwtSecret reads JWT_SECRET at call time. Reading it in a package-level variable
// would capture the value before main() runs godotenv.Load(), signing tokens with an empty key.
func jwtSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.NewInternalServerError("JWT_SECRET is not set")
	}
	return []byte(secret), nil
}

// MinJWTSecretLength is the minimum JWT_SECRET length in bytes accepted at startup.
const MinJWTSecretLength = 32

// ValidateJWTSecret returns an error when JWT_SECRET is empty or shorter than MinJWTSecretLength bytes.
func ValidateJWTSecret() error {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return fmt.Errorf("JWT_SECRET is not set")
	}
	if len(secret) < MinJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d bytes long, got %d", MinJWTSecretLength, len(secret))
	}
	return nil
}

// GenerateJWT creates a new JWT token for the given user ID.
// The token is valid for 72 hours and includes the user ID in the claims.
func GenerateJWT(userID string) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// VerifyJWT checks the validity of a JWT token and returns the user ID if valid.
// It returns an error if the token is invalid or if the claims are not as expected.
func VerifyJWT(tokenString string) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return "", errors.ErrInvalidField("token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.ErrInvalidField("claims")
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", errors.ErrInvalidField("claims")
	}
	return userID, nil
}
